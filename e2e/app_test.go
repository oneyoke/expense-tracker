package e2e

import (
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestApp(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("could not launch playwright: %v", err)
	}
	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("could not launch chromium: %v", err)
	}
	defer func() {
		if err := browser.Close(); err != nil {
			t.Logf("failed to close browser: %v", err)
		}
		if err := pw.Stop(); err != nil {
			t.Logf("failed to stop playwright: %v", err)
		}
	}()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}

	if _, err = page.Goto(appURL); err != nil {
		t.Fatalf("could not goto: %v", err)
	}

	// Assertions helper
	expect := playwright.NewPlaywrightAssertions()

	// 0. Login first
	// Wait for login form
	if err := expect.Locator(page.Locator(".login-form")).ToBeVisible(); err != nil {
		t.Fatalf("Login form not visible: %v", err)
	}

	// Fill in credentials
	if err = page.Locator("input[name=username]").Fill("testuser"); err != nil {
		t.Fatalf("failed to fill username: %v", err)
	}
	if err = page.Locator("input[name=password]").Fill("testpass123"); err != nil {
		t.Fatalf("failed to fill password: %v", err)
	}

	// Submit login
	if err = page.Locator(".login-btn").Click(); err != nil {
		t.Fatalf("failed to click login: %v", err)
	}

	// Wait for redirect to expenses page
	if err := expect.Locator(page.Locator(".list-screen")).ToBeVisible(); err != nil {
		t.Fatalf("Did not redirect to expenses page after login: %v", err)
	}

	// 1. Verify Homepage
	// Check for "Spent this month" text
	if err := expect.Locator(page.Locator(".summary small")).ToHaveText("Spent this month"); err != nil {
		t.Fatalf("Homepage assertion failed: %v", err)
	}

	// 2. Create Expense
	// Click add button
	if err = page.Locator(".fab-add").Click(); err != nil {
		t.Fatalf("failed to click add: %v", err)
	}

	// Wait for form
	if err := expect.Locator(page.Locator("#expense-form")).ToBeVisible(); err != nil {
		t.Fatalf("Form not visible: %v", err)
	}

	// Enter Amount: 12.50 using keypad
	// Note: buttons have text "1", "2", etc.
	// We use exact match or text match.
	keys := []string{"1", "2", ".", "5", "0"}
	for _, key := range keys {
		// Using text=key selector
		if err = page.Locator("button:text-is('" + key + "')").Click(); err != nil {
			t.Fatalf("failed to click key %s: %v", key, err)
		}
	}

	// Verify amount display
	if err := expect.Locator(page.Locator("#display-amount")).ToHaveText("12.50"); err != nil {
		t.Fatalf("Amount display mismatch: %v", err)
	}

	// Description
	if err = page.Locator("input[name=description]").Fill("Lunch Test"); err != nil {
		t.Fatalf("failed to fill description: %v", err)
	}

	// Category
	// The selector for options accepts value or label.
	if _, err = page.Locator("select[name=category]").SelectOption(playwright.SelectOptionValues{
		Values: &[]string{"food"},
	}); err != nil {
		t.Fatalf("failed to select category: %v", err)
	}

	// Submit
	if err = page.Locator("button.submit").Click(); err != nil {
		t.Fatalf("failed to submit: %v", err)
	}

	// 3. Verify in List
	// Wait for expense item to appear
	if err := expect.Locator(page.Locator(".expense-item")).ToHaveCount(1); err != nil {
		t.Fatalf("Expense item count mismatch: %v", err)
	}

	item := page.Locator(".expense-item").First()
	if err := expect.Locator(item.Locator(".expense-details strong")).ToHaveText("Lunch Test"); err != nil {
		t.Fatalf("Description mismatch: %v", err)
	}
	if err := expect.Locator(item.Locator(".expense-amount")).ToContainText("12.50"); err != nil {
		t.Fatalf("Amount mismatch: %v", err)
	}
}
