package handlers

import (
	"context"
	"errors"
	"expense-tracker/internal/auth"
	"expense-tracker/internal/models"
	"expense-tracker/internal/storage"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Context key type to avoid collisions.
type contextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey contextKey = "user"
	// SessionCookieName is the name of the session cookie.
	SessionCookieName = "session"
	// SessionDuration is how long sessions last (30 days).
	SessionDuration = 30 * 24 * time.Hour
)

// Handlers holds dependencies for HTTP handlers.
type Handlers struct {
	db           *storage.DB
	templateDir  string
	secureCookie bool
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db *storage.DB, templateDir string, secureCookie bool) *Handlers {
	return &Handlers{db: db, templateDir: templateDir, secureCookie: secureCookie}
}

// GetUserFromContext retrieves the authenticated user from request context.
func GetUserFromContext(r *http.Request) *models.User {
	if user, ok := r.Context().Value(UserContextKey).(*models.User); ok {
		return user
	}
	return nil
}

// AuthMiddleware wraps handlers to require authentication.
// It also implements rolling sessions: if a session is past the halfway point
// of its lifetime, it automatically renews the session.
func (h *Handlers) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		sessionInfo, err := h.db.ValidateSessionWithInfo(cookie.Value)
		if err != nil {
			// Invalid or expired session, clear the cookie
			h.clearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Rolling session: renew if past halfway point
		// This keeps active users logged in while still expiring inactive sessions
		now := time.Now()
		timeUntilExpiry := sessionInfo.ExpiresAt.Sub(now)
		halfSessionDuration := SessionDuration / 2

		if timeUntilExpiry < halfSessionDuration {
			// Session is in the second half of its lifetime, renew it
			newExpiresAt := now.Add(SessionDuration)
			if err := h.db.RenewSession(cookie.Value, newExpiresAt); err == nil {
				// Update the cookie expiration too
				http.SetCookie(w, &http.Cookie{
					Name:     SessionCookieName,
					Value:    cookie.Value,
					Path:     "/",
					MaxAge:   int(SessionDuration.Seconds()),
					HttpOnly: true,
					Secure:   h.secureCookie,
					SameSite: http.SameSiteLaxMode,
				})
			}
			// If renewal fails, just continue with the current session
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, sessionInfo.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoginViewModel holds data for the login page.
type LoginViewModel struct {
	Error string
}

// LoginForm renders the login page.
func (h *Handlers) LoginForm(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to expenses
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		if _, err := h.db.ValidateSession(cookie.Value); err == nil {
			http.Redirect(w, r, "/expenses", http.StatusFound)
			return
		}
	}
	h.render(w, r, "login.html", LoginViewModel{})
}

// Login handles the login form submission.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, "login.html", LoginViewModel{Error: "Invalid form submission"})
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		h.render(w, r, "login.html", LoginViewModel{Error: "Username and password are required"})
		return
	}

	user, err := h.db.GetUserByUsername(username)
	if err != nil || !auth.CheckPassword(password, user.PasswordHash) {
		h.render(w, r, "login.html", LoginViewModel{Error: "Invalid username or password"})
		return
	}

	// Generate session token
	token, err := auth.GenerateSessionToken()
	if err != nil {
		log.Printf("Failed to generate session token: %v", err)
		h.render(w, r, "login.html", LoginViewModel{Error: "An error occurred. Please try again."})
		return
	}

	// Create session in database
	expiresAt := time.Now().Add(SessionDuration)
	if err := h.db.CreateSession(token, user.ID, expiresAt); err != nil {
		log.Printf("Failed to create session: %v", err)
		h.render(w, r, "login.html", LoginViewModel{Error: "An error occurred. Please try again."})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/expenses", http.StatusFound)
}

// Logout handles user logout.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if err := h.db.DeleteSession(cookie.Value); err != nil {
			log.Printf("Failed to delete session: %v", err)
		}
	}
	h.clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *Handlers) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// CategoryDef defines the properties of a category.
type CategoryDef struct {
	ID    string
	Name  string
	Icon  string
	Color string
}

var categories = []CategoryDef{
	{"food", "Food", "🍽️", "#60a5fa"},
	{"transport", "Transport", "🚌", "#a78bfa"},
	{"entertainment", "Entertainment", "🎮", "#f472b6"},
	{"utilities", "Utilities", "💡", "#fbbf24"},
	{"housing", "Housing", "🏠", "#818cf8"},
	{"gifts", "Gifts", "🎁", "#fb7185"},
	{"other", "Other", "📦", "#94a3b8"},
}

// CategoryStyle defines the visual style for a category.
type CategoryStyle struct {
	Icon  string
	Color string
}

func getCategoryStyle(category string) CategoryStyle {
	catLower := strings.ToLower(category)
	for _, c := range categories {
		if c.ID == catLower {
			return CategoryStyle{Icon: c.Icon, Color: c.Color}
		}
	}
	return CategoryStyle{Icon: "📦", Color: "#94a3b8"}
}

// ExpenseItem represents an expense in the list view.
type ExpenseItem struct {
	models.Expense
	Time          string
	CategoryStyle CategoryStyle
	IsIncome      bool
}

// ExpenseGroup groups expenses by date.
type ExpenseGroup struct {
	Title string
	Date  string
	Total float64
	Items []ExpenseItem
}

// ListViewModel is the data passed to the list view template.
type ListViewModel struct {
	Total  float64
	Groups []ExpenseGroup
}

// FormViewModel is the data passed to the create/edit form template.
type FormViewModel struct {
	Expense       *models.Expense
	IsEdit        bool
	FormattedDate string
	Categories    []CategoryDef
}

// ListExpenses renders the list of expenses.
func (h *Handlers) ListExpenses(w http.ResponseWriter, r *http.Request) {
	expenses, err := h.db.ListExpenses()
	if err != nil {
		log.Printf("ListExpenses error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	groupsMap := make(map[string]*ExpenseGroup)
	var totalSpent float64

	for _, e := range expenses {
		dateStr := e.Date.Format("2006-01-02")
		if _, ok := groupsMap[dateStr]; !ok {
			groupsMap[dateStr] = &ExpenseGroup{Date: dateStr, Title: formatGroupTitle(e.Date)}
		}
		group := groupsMap[dateStr]
		group.Total += e.Amount
		totalSpent += e.Amount

		group.Items = append(group.Items, ExpenseItem{
			Expense:       e,
			Time:          e.Date.Format("15:04"),
			CategoryStyle: getCategoryStyle(e.Category),
			IsIncome:      strings.Contains(e.Description, "[Income]"),
		})
	}

	groups := make([]ExpenseGroup, 0, len(groupsMap))
	for _, g := range groupsMap {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Date > groups[j].Date })

	h.render(w, r, "list.html", ListViewModel{Total: totalSpent, Groups: groups})
}

// CreateExpenseForm renders the form to create a new expense.
func (h *Handlers) CreateExpenseForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "create.html", FormViewModel{
		IsEdit:     false,
		Categories: categories,
	})
}

// EditExpenseForm renders the form to edit an existing expense.
func (h *Handlers) EditExpenseForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if expense, err := h.db.GetExpense(id); err == nil {
		h.render(w, r, "create.html", FormViewModel{
			Expense:       expense,
			IsEdit:        true,
			FormattedDate: expense.Date.Format("2006-01-02T15:04:05"),
			Categories:    categories,
		})
	} else {
		http.Error(w, "Expense not found", http.StatusNotFound)
	}
}

// CreateExpense handles the creation of a new expense.
func (h *Handlers) CreateExpense(w http.ResponseWriter, r *http.Request) {
	amount, desc, cat, date, err := parseForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.db.CreateExpense(amount, desc, cat, date); err != nil {
		log.Printf("CreateExpense error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Location", `{"path":"/expenses", "target":"#content"}`)
}

// UpdateExpense handles the update of an existing expense.
func (h *Handlers) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	amount, desc, cat, date, err := parseForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.db.UpdateExpense(&models.Expense{
		ID: id, Amount: amount, Description: desc, Category: cat, Date: date,
	}); err != nil {
		log.Printf("UpdateExpense error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Location", `{"path":"/expenses", "target":"#content"}`)
}

func parseForm(r *http.Request) (amount float64, desc, category string, date time.Time, err error) {
	if err := r.ParseForm(); err != nil {
		return 0, "", "", time.Time{}, err
	}
	amount, _ = strconv.ParseFloat(r.FormValue("amount"), 64)
	desc = r.FormValue("description")
	if desc == "" {
		desc = "Expense"
	}
	dateStr := r.FormValue("date")
	if dateStr == "" {
		return 0, "", "", time.Time{}, errors.New("date is required")
	}
	date, err = time.Parse("2006-01-02T15:04", dateStr)
	if err != nil {
		return 0, "", "", time.Time{}, err
	}
	return amount, desc, category, date, nil
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request, viewName string, data any) {
	tmpl, err := template.ParseFiles(filepath.Join(h.templateDir, "base.html"), filepath.Join(h.templateDir, viewName))
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	target := "base.html"
	if r.Header.Get("HX-Request") == "true" {
		target = "content"
	}
	if err := tmpl.ExecuteTemplate(w, target, data); err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

func formatGroupTitle(date time.Time) string {
	dateStr := date.Format("2006-01-02")
	nowStr := time.Now().Format("2006-01-02")

	if dateStr == nowStr {
		return "TODAY"
	}
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if dateStr == yesterdayStr {
		return "YESTERDAY"
	}
	return strings.ToUpper(date.Format("Mon, 02 Jan '06"))
}
