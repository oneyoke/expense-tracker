package storage

import (
	"testing"
	"time"

	"expense-tracker/internal/auth"
)

func TestDB(t *testing.T) {
	// Use in-memory database for testing
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	t.Run("CreateExpense", func(t *testing.T) {
		err := db.CreateExpense(10.50, "Lunch", "food", time.Now())
		if err != nil {
			t.Errorf("Failed to create expense: %v", err)
		}
	})

	t.Run("UniqueDate", func(t *testing.T) {
		now := time.Now()
		// First insert should succeed
		err := db.CreateExpense(10.00, "First", "test", now)
		if err != nil {
			t.Fatalf("Failed to create first expense: %v", err)
		}

		// Second insert with same timestamp should fail due to unique index
		err = db.CreateExpense(20.00, "Second", "test", now)
		if err == nil {
			t.Error("Expected error on duplicate date, got nil")
		}
	})

	t.Run("ListExpenses", func(t *testing.T) {
		// Use distinct times to avoid collision with UniqueDate test
		baseTime := time.Now().Add(time.Hour)

		// Create a few more expenses
		if err := db.CreateExpense(20.00, "Bus", "transport", baseTime.Add(time.Minute)); err != nil {
			t.Fatalf("Failed to create Bus expense: %v", err)
		}
		if err := db.CreateExpense(5.00, "Coffee", "food", baseTime.Add(2*time.Minute)); err != nil {
			t.Fatalf("Failed to create Coffee expense: %v", err)
		}

		expenses, err := db.ListExpenses()
		if err != nil {
			t.Errorf("Failed to list expenses: %v", err)
		}

		// We expect 1 from CreateExpense (Lunch), 1 from UniqueDate (First), and 2 here (Bus, Coffee) = 4 total
		if len(expenses) != 4 {
			t.Errorf("Expected 4 expenses, got %d", len(expenses))
		}

		// Check order (latest first). Coffee was added last with latest timestamp
		if len(expenses) > 0 && expenses[0].Amount != 5.00 {
			t.Errorf("Expected first expense to be Coffee (5.00), got %.2f", expenses[0].Amount)
		}
	})
}

func TestSessions(t *testing.T) {
	// Use in-memory database for testing
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create a test user
	password, err := auth.HashPassword("testpass")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	user, err := db.CreateUser("testuser", password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("CreateAndValidateSession", func(t *testing.T) {
		token, err := auth.GenerateSessionToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		err = db.CreateSession(token, user.ID, expiresAt)
		if err != nil {
			t.Errorf("Failed to create session: %v", err)
		}

		// Validate the session
		sessionUser, err := db.ValidateSession(token)
		if err != nil {
			t.Errorf("Failed to validate session: %v", err)
		}
		if sessionUser.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", sessionUser.Username)
		}
	})

	t.Run("ValidateSessionWithInfo", func(t *testing.T) {
		token, _ := auth.GenerateSessionToken()
		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		db.CreateSession(token, user.ID, expiresAt)

		// Get session info
		info, err := db.ValidateSessionWithInfo(token)
		if err != nil {
			t.Errorf("Failed to validate session with info: %v", err)
		}

		if info.User.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", info.User.Username)
		}

		// Check that last_activity is recent
		timeSinceActivity := time.Since(info.LastActivity)
		if timeSinceActivity > 5*time.Second {
			t.Errorf("LastActivity should be recent, but was %v ago", timeSinceActivity)
		}
	})

	t.Run("RenewSession", func(t *testing.T) {
		token, _ := auth.GenerateSessionToken()
		originalExpiry := time.Now().Add(30 * 24 * time.Hour)
		db.CreateSession(token, user.ID, originalExpiry)

		// Wait a moment to ensure timestamps differ
		time.Sleep(10 * time.Millisecond)

		// Get original session info
		originalInfo, _ := db.ValidateSessionWithInfo(token)

		// Renew the session
		newExpiry := time.Now().Add(60 * 24 * time.Hour)
		err := db.RenewSession(token, newExpiry)
		if err != nil {
			t.Errorf("Failed to renew session: %v", err)
		}

		// Get updated session info
		updatedInfo, err := db.ValidateSessionWithInfo(token)
		if err != nil {
			t.Errorf("Failed to validate renewed session: %v", err)
		}

		// Verify last_activity was updated
		if !updatedInfo.LastActivity.After(originalInfo.LastActivity) {
			t.Errorf("LastActivity should be updated after renewal")
		}

		// Verify expires_at was updated
		if !updatedInfo.ExpiresAt.After(originalInfo.ExpiresAt) {
			t.Errorf("ExpiresAt should be extended after renewal")
		}
	})

	t.Run("DeleteSession", func(t *testing.T) {
		token, _ := auth.GenerateSessionToken()
		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		db.CreateSession(token, user.ID, expiresAt)

		// Verify session exists
		_, err := db.ValidateSession(token)
		if err != nil {
			t.Errorf("Session should exist: %v", err)
		}

		// Delete session
		err = db.DeleteSession(token)
		if err != nil {
			t.Errorf("Failed to delete session: %v", err)
		}

		// Verify session is gone
		_, err = db.ValidateSession(token)
		if err == nil {
			t.Errorf("Expected error after deleting session, got nil")
		}
	})
}
