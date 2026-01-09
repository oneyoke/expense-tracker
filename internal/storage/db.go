package storage

import (
	"database/sql"
	"time"

	"expense-tracker/internal/models"

	// Import sqlite driver
	_ "modernc.org/sqlite"
)

// DB wraps a sql.DB connection.
type DB struct {
	conn *sql.DB
}

// NewDB opens a database connection and runs migrations.
func NewDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS expenses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			amount REAL NOT NULL,
			description TEXT NOT NULL,
			category TEXT NOT NULL,
			date DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return err
		}
	}

	// Add user_id column to expenses if it doesn't exist (for backwards compatibility)
	// We ignore the error here because the column might already exist
	_, _ = db.conn.Exec(`ALTER TABLE expenses ADD COLUMN user_id INTEGER REFERENCES users(id)`)

	// Add last_activity column to sessions for rolling sessions
	_, _ = db.conn.Exec(`ALTER TABLE sessions ADD COLUMN last_activity DATETIME DEFAULT CURRENT_TIMESTAMP`)

	return nil
}

// CreateExpense inserts a new expense into the database.
func (db *DB) CreateExpense(amount float64, description, category string, date time.Time) error {
	if date.IsZero() {
		date = time.Now()
	}
	_, err := db.conn.Exec(
		"INSERT INTO expenses (amount, description, category, date) VALUES (?, ?, ?, ?)",
		amount, description, category, date,
	)
	return err
}

// GetExpense retrieves a single expense by ID.
func (db *DB) GetExpense(id int64) (*models.Expense, error) {
	row := db.conn.QueryRow(
		"SELECT id, amount, description, category, date FROM expenses WHERE id = ?",
		id,
	)

	var e models.Expense
	if err := row.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date); err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateExpense updates an existing expense in the database.
func (db *DB) UpdateExpense(e *models.Expense) error {
	_, err := db.conn.Exec(
		"UPDATE expenses SET amount = ?, description = ?, category = ?, date = ? WHERE id = ?",
		e.Amount, e.Description, e.Category, e.Date, e.ID,
	)
	return err
}

// ListExpenses retrieves expenses for the current month from the database, ordered by date descending.
func (db *DB) ListExpenses() ([]models.Expense, error) {
	// Calculate start of current month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, date FROM expenses WHERE date >= ? ORDER BY date DESC",
		startOfMonth,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, rows.Err()
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// CreateUser creates a new user with the given username and password hash.
func (db *DB) CreateUser(username, passwordHash string) (*models.User, error) {
	result, err := db.conn.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetUserByID(id)
}

// GetUserByID retrieves a user by ID.
func (db *DB) GetUserByID(id int64) (*models.User, error) {
	row := db.conn.QueryRow(
		"SELECT id, username, password_hash, created_at FROM users WHERE id = ?",
		id,
	)

	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByUsername retrieves a user by username.
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	row := db.conn.QueryRow(
		"SELECT id, username, password_hash, created_at FROM users WHERE username = ?",
		username,
	)

	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateSession creates a new session for a user.
func (db *DB) CreateSession(token string, userID int64, expiresAt time.Time) error {
	now := time.Now()
	_, err := db.conn.Exec(
		"INSERT INTO sessions (token, user_id, expires_at, last_activity) VALUES (?, ?, ?, ?)",
		token, userID, expiresAt, now,
	)
	return err
}

// SessionInfo holds session validation data.
type SessionInfo struct {
	User         *models.User
	LastActivity time.Time
	ExpiresAt    time.Time
}

// ValidateSession checks if a session token is valid and returns the associated user.
func (db *DB) ValidateSession(token string) (*models.User, error) {
	info, err := db.ValidateSessionWithInfo(token)
	if err != nil {
		return nil, err
	}
	return info.User, nil
}

// ValidateSessionWithInfo checks if a session token is valid and returns session details.
func (db *DB) ValidateSessionWithInfo(token string) (*SessionInfo, error) {
	row := db.conn.QueryRow(`
		SELECT u.id, u.username, u.password_hash, u.created_at, s.last_activity, s.expires_at
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP
	`, token)

	var u models.User
	var lastActivity, expiresAt time.Time
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &lastActivity, &expiresAt); err != nil {
		return nil, err
	}
	return &SessionInfo{
		User:         &u,
		LastActivity: lastActivity,
		ExpiresAt:    expiresAt,
	}, nil
}

// RenewSession updates the last_activity and expires_at for a session.
func (db *DB) RenewSession(token string, newExpiresAt time.Time) error {
	now := time.Now()
	_, err := db.conn.Exec(
		"UPDATE sessions SET last_activity = ?, expires_at = ? WHERE token = ?",
		now, newExpiresAt, token,
	)
	return err
}

// DeleteSession removes a session by token.
func (db *DB) DeleteSession(token string) error {
	_, err := db.conn.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// CleanExpiredSessions removes all expired sessions.
func (db *DB) CleanExpiredSessions() error {
	_, err := db.conn.Exec("DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP")
	return err
}

// UserCount returns the number of users in the database.
func (db *DB) UserCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}
