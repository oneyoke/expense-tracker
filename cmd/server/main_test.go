package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"expense-tracker/internal/handlers"
	"expense-tracker/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupRouter(t *testing.T) {
	// Setup dependencies
	db, err := storage.NewDB(":memory:")
	require.NoError(t, err, "failed to create database")
	defer db.Close()

	// Use relative paths for tests running in cmd/server
	h := handlers.NewHandlers(db, "../../web/templates", false)

	// Ensure template directory exists, otherwise skip handler initialization if it panics (handlers might check for templates)
	if _, err := os.Stat("../../web/templates"); os.IsNotExist(err) {
		t.Skip("Template directory not found, skipping router test")
	}

	// Create router - this triggers the panic if routing conflict exists
	mux := setupRouter(h, "../../web/static")

	// Verify routes
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		allowAlt   []int // Alternative acceptable status codes
	}{
		{
			name:       "Root redirects to /expenses",
			method:     "GET",
			path:       "/",
			wantStatus: http.StatusFound,
		},
		{
			name:       "Static file access",
			method:     "GET",
			path:       "/static/style.css",
			wantStatus: http.StatusOK,
			allowAlt:   []int{http.StatusNotFound}, // File might not exist in test env
		},
		{
			name:       "List Expenses requires auth",
			method:     "GET",
			path:       "/expenses",
			wantStatus: http.StatusFound, // Should redirect to login
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			// Check if status matches expected or any alternative
			if len(tt.allowAlt) > 0 {
				acceptableStatuses := append([]int{tt.wantStatus}, tt.allowAlt...)
				assert.Contains(t, acceptableStatuses, w.Code,
					"%s %s returned unexpected status", tt.method, tt.path)
			} else {
				assert.Equal(t, tt.wantStatus, w.Code,
					"%s %s returned unexpected status", tt.method, tt.path)
			}
		})
	}
}

