package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StatsCategoryItem represents a category with its spending statistics.
type StatsCategoryItem struct {
	Category      string
	Total         float64
	Count         int
	Percentage    float64
	CategoryStyle CategoryStyle
}

// StatsViewModel is the data passed to the statistics view template.
type StatsViewModel struct {
	Year           int
	Month          int
	MonthName      string
	Total          float64
	Categories     []StatsCategoryItem
	Expenses       []ExpenseItem
	PrevYear       int
	PrevMonth      int
	NextYear       int
	NextMonth      int
	IsCurrentMonth bool
}

// Statistics renders the statistics page.
func (h *Handlers) Statistics(w http.ResponseWriter, r *http.Request) {
	// Get year and month from query params, default to current month
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	// Get category totals
	categoryTotals, err := h.db.GetCategoryTotalsByMonth(year, month)
	if err != nil {
		log.Printf("GetCategoryTotalsByMonth error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get expenses for the month
	expenses, err := h.db.GetExpensesByMonth(year, month)
	if err != nil {
		log.Printf("GetExpensesByMonth error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Calculate total and prepare category items
	var total float64
	categoryItems := make([]StatsCategoryItem, 0, len(categoryTotals))
	for _, ct := range categoryTotals {
		total += ct.Total
	}

	// Calculate percentages and add style
	for _, ct := range categoryTotals {
		percentage := 0.0
		if total > 0 {
			percentage = (ct.Total / total) * 100
		}
		categoryItems = append(categoryItems, StatsCategoryItem{
			Category:      ct.Category,
			Total:         ct.Total,
			Count:         ct.Count,
			Percentage:    percentage,
			CategoryStyle: getCategoryStyle(ct.Category),
		})
	}

	// Prepare expense items
	expenseItems := make([]ExpenseItem, 0, len(expenses))
	for _, e := range expenses {
		expenseItems = append(expenseItems, ExpenseItem{
			ID:            e.ID,
			Amount:        e.Amount,
			Description:   e.Description,
			Category:      e.Category,
			Time:          e.Date.Format("Jan 02, 15:04"),
			CategoryStyle: getCategoryStyle(e.Category),
			IsIncome:      strings.Contains(e.Description, "[Income]"),
		})
	}

	// Calculate previous and next month
	prevDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	nextDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)

	// Check if this is the current month
	isCurrentMonth := year == now.Year() && month == int(now.Month())

	monthName := time.Month(month).String()

	h.render(w, r, "stats.html", StatsViewModel{
		Year:           year,
		Month:          month,
		MonthName:      monthName,
		Total:          total,
		Categories:     categoryItems,
		Expenses:       expenseItems,
		PrevYear:       prevDate.Year(),
		PrevMonth:      int(prevDate.Month()),
		NextYear:       nextDate.Year(),
		NextMonth:      int(nextDate.Month()),
		IsCurrentMonth: isCurrentMonth,
	})
}
