package storage

import (
	"testing"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DBTestSuite provides a test suite for database operations
type DBTestSuite struct {
	suite.Suite
	db *DB
}

// SetupTest runs before each test
func (suite *DBTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	require.NoError(suite.T(), err, "failed to create test database")
	suite.db = db
}

// TearDownTest runs after each test
func (suite *DBTestSuite) TearDownTest() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *DBTestSuite) TestCreateExpense() {
	err := suite.db.CreateExpense(10.50, "Lunch", "food", time.Now())
	assert.NoError(suite.T(), err)
}

func (suite *DBTestSuite) TestCreateMultipleExpensesWithSameTimestamp() {
	now := time.Now()

	// First insert should succeed
	err := suite.db.CreateExpense(10.00, "First", "test", now)
	require.NoError(suite.T(), err)

	// Second insert with same timestamp (currently no unique constraint, so this will succeed)
	err = suite.db.CreateExpense(20.00, "Second", "test", now)
	assert.NoError(suite.T(), err)
}

func (suite *DBTestSuite) TestListExpenses() {
	baseTime := time.Now().Add(time.Hour)

	// Create test expenses
	expenses := []struct {
		amount      float64
		description string
		category    string
		offset      time.Duration
	}{
		{20.00, "Bus", "transport", time.Minute},
		{5.00, "Coffee", "food", 2 * time.Minute},
		{15.00, "Snack", "food", 3 * time.Minute},
	}

	for _, exp := range expenses {
		err := suite.db.CreateExpense(exp.amount, exp.description, exp.category, baseTime.Add(exp.offset))
		require.NoError(suite.T(), err, "failed to create expense: %s", exp.description)
	}

	result, err := suite.db.ListExpenses()
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 3, "expected 3 expenses")

	// Check order (latest first). Snack was added last with latest timestamp
	if len(result) > 0 {
		assert.Equal(suite.T(), 15.00, result[0].Amount, "expected first expense to be Snack")
		assert.Equal(suite.T(), "Snack", result[0].Description)
	}
}

func (suite *DBTestSuite) TestListExpensesCurrentMonth() {
	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 15, 12, 0, 0, 0, now.Location())
	lastMonth := time.Date(now.Year(), now.Month()-1, 15, 12, 0, 0, 0, now.Location())
	twoMonthsAgo := time.Date(now.Year(), now.Month()-2, 15, 12, 0, 0, 0, now.Location())

	// Create expenses in different months
	testExpenses := []struct {
		amount      float64
		description string
		category    string
		date        time.Time
	}{
		{100.00, "Current Month 1", "food", currentMonth},
		{150.00, "Current Month 2", "transport", currentMonth.Add(24 * time.Hour)},
		{200.00, "Last Month", "food", lastMonth},
		{300.00, "Two Months Ago", "utilities", twoMonthsAgo},
	}

	for _, exp := range testExpenses {
		err := suite.db.CreateExpense(exp.amount, exp.description, exp.category, exp.date)
		require.NoError(suite.T(), err, "failed to create expense: %s", exp.description)
	}

	// List expenses should only return current month
	expenses, err := suite.db.ListExpenses()
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), expenses, 2, "expected only current month expenses")

	// Verify all returned expenses are from current month
	for _, exp := range expenses {
		assert.Equal(suite.T(), now.Month(), exp.Date.Month(), "expense month mismatch")
		assert.Equal(suite.T(), now.Year(), exp.Date.Year(), "expense year mismatch")
	}

	// Verify the expenses are the correct ones (ordered by date DESC)
	if assert.Len(suite.T(), expenses, 2) {
		assert.Equal(suite.T(), "Current Month 2", expenses[0].Description)
		assert.Equal(suite.T(), 150.00, expenses[0].Amount)
		assert.Equal(suite.T(), "Current Month 1", expenses[1].Description)
		assert.Equal(suite.T(), 100.00, expenses[1].Amount)
	}
}

// SessionTestSuite provides a test suite for session operations
type SessionTestSuite struct {
	suite.Suite
	db   *DB
	user *models.User
}

// SetupTest runs before each test
func (suite *SessionTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	require.NoError(suite.T(), err, "failed to create test database")
	suite.db = db

	// Create a test user
	password, err := auth.HashPassword("testpass")
	require.NoError(suite.T(), err, "failed to hash password")

	user, err := suite.db.CreateUser("testuser", password)
	require.NoError(suite.T(), err, "failed to create test user")
	suite.user = user
}

// TearDownTest runs after each test
func (suite *SessionTestSuite) TearDownTest() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *SessionTestSuite) TestCreateAndValidateSession() {
	token, err := auth.GenerateSessionToken()
	require.NoError(suite.T(), err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = suite.db.CreateSession(token, suite.user.ID, expiresAt)
	require.NoError(suite.T(), err)

	// Validate the session
	sessionUser, err := suite.db.ValidateSession(token)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "testuser", sessionUser.Username)
}

func (suite *SessionTestSuite) TestValidateSessionWithInfo() {
	token, err := auth.GenerateSessionToken()
	require.NoError(suite.T(), err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = suite.db.CreateSession(token, suite.user.ID, expiresAt)
	require.NoError(suite.T(), err)

	// Get session info
	info, err := suite.db.ValidateSessionWithInfo(token)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "testuser", info.User.Username)

	// Check that last_activity is recent
	timeSinceActivity := time.Since(info.LastActivity)
	assert.Less(suite.T(), timeSinceActivity, 5*time.Second, "LastActivity should be recent")
}

func (suite *SessionTestSuite) TestRenewSession() {
	token, err := auth.GenerateSessionToken()
	require.NoError(suite.T(), err)

	originalExpiry := time.Now().Add(30 * 24 * time.Hour)
	err = suite.db.CreateSession(token, suite.user.ID, originalExpiry)
	require.NoError(suite.T(), err)

	// Wait a moment to ensure timestamps differ
	time.Sleep(10 * time.Millisecond)

	// Get original session info
	originalInfo, err := suite.db.ValidateSessionWithInfo(token)
	require.NoError(suite.T(), err)

	// Renew the session
	newExpiry := time.Now().Add(60 * 24 * time.Hour)
	err = suite.db.RenewSession(token, newExpiry)
	require.NoError(suite.T(), err)

	// Get updated session info
	updatedInfo, err := suite.db.ValidateSessionWithInfo(token)
	require.NoError(suite.T(), err)

	// Verify last_activity was updated
	assert.True(suite.T(), updatedInfo.LastActivity.After(originalInfo.LastActivity),
		"LastActivity should be updated after renewal")

	// Verify expires_at was updated
	assert.True(suite.T(), updatedInfo.ExpiresAt.After(originalInfo.ExpiresAt),
		"ExpiresAt should be extended after renewal")
}

func (suite *SessionTestSuite) TestDeleteSession() {
	token, err := auth.GenerateSessionToken()
	require.NoError(suite.T(), err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = suite.db.CreateSession(token, suite.user.ID, expiresAt)
	require.NoError(suite.T(), err)

	// Verify session exists
	_, err = suite.db.ValidateSession(token)
	require.NoError(suite.T(), err, "session should exist before deletion")

	// Delete session
	err = suite.db.DeleteSession(token)
	require.NoError(suite.T(), err)

	// Verify session is gone
	_, err = suite.db.ValidateSession(token)
	assert.Error(suite.T(), err, "expected error after deleting session")
}

// Test suite runners
func TestDBSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}

func TestSessionSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}
