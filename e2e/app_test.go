package e2e

import (
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite provides a test suite for end-to-end tests
type E2ETestSuite struct {
	suite.Suite
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
	expect  playwright.PlaywrightAssertions
}

// SetupSuite runs once before all tests
func (suite *E2ETestSuite) SetupSuite() {
	pw, err := playwright.Run()
	require.NoError(suite.T(), err, "could not launch playwright")
	suite.pw = pw

	browser, err := pw.Chromium.Launch()
	require.NoError(suite.T(), err, "could not launch chromium")
	suite.browser = browser

	suite.expect = playwright.NewPlaywrightAssertions()
}

// TearDownSuite runs once after all tests
func (suite *E2ETestSuite) TearDownSuite() {
	if suite.browser != nil {
		suite.browser.Close()
	}
	if suite.pw != nil {
		suite.pw.Stop()
	}
}

// SetupTest runs before each test
func (suite *E2ETestSuite) SetupTest() {
	page, err := suite.browser.NewPage()
	require.NoError(suite.T(), err, "could not create page")
	suite.page = page

	_, err = suite.page.Goto(appURL)
	require.NoError(suite.T(), err, "could not navigate to app")
}

// TearDownTest runs after each test
func (suite *E2ETestSuite) TearDownTest() {
	if suite.page != nil {
		suite.page.Close()
	}
}

func (suite *E2ETestSuite) login() {
	// Wait for login form
	err := suite.expect.Locator(suite.page.Locator(".login-form")).ToBeVisible()
	require.NoError(suite.T(), err, "login form not visible")

	// Fill in credentials
	err = suite.page.Locator("input[name=username]").Fill("testuser")
	require.NoError(suite.T(), err, "failed to fill username")

	err = suite.page.Locator("input[name=password]").Fill("testpass123")
	require.NoError(suite.T(), err, "failed to fill password")

	// Submit login
	err = suite.page.Locator(".login-btn").Click()
	require.NoError(suite.T(), err, "failed to click login")

	// Wait for redirect to expenses page
	err = suite.expect.Locator(suite.page.Locator(".list-screen")).ToBeVisible()
	require.NoError(suite.T(), err, "did not redirect to expenses page after login")
}

func (suite *E2ETestSuite) TestCompleteUserFlow() {
	// Login
	suite.login()

	// Verify Homepage
	err := suite.expect.Locator(suite.page.Locator(".summary small")).ToHaveText("Spent this month")
	require.NoError(suite.T(), err, "homepage assertion failed")

	// Create Expense - Click add button
	err = suite.page.Locator(".fab-add").Click()
	require.NoError(suite.T(), err, "failed to click add button")

	// Wait for form
	err = suite.expect.Locator(suite.page.Locator("#expense-form")).ToBeVisible()
	require.NoError(suite.T(), err, "expense form not visible")

	// Enter Amount: 12.50 using keypad
	keys := []string{"1", "2", ".", "5", "0"}
	for _, key := range keys {
		err = suite.page.Locator("button:text-is('" + key + "')").Click()
		require.NoError(suite.T(), err, "failed to click key %s", key)
	}

	// Verify amount display
	err = suite.expect.Locator(suite.page.Locator("#display-amount")).ToHaveText("12.50")
	require.NoError(suite.T(), err, "amount display mismatch")

	// Fill description
	err = suite.page.Locator("input[name=description]").Fill("Lunch Test")
	require.NoError(suite.T(), err, "failed to fill description")

	// Select category
	_, err = suite.page.Locator("select[name=category]").SelectOption(playwright.SelectOptionValues{
		Values: &[]string{"food"},
	})
	require.NoError(suite.T(), err, "failed to select category")

	// Submit
	err = suite.page.Locator("button.submit").Click()
	require.NoError(suite.T(), err, "failed to submit expense")

	// Verify in List - Wait for expense item to appear
	err = suite.expect.Locator(suite.page.Locator(".expense-item")).ToHaveCount(1)
	require.NoError(suite.T(), err, "expense item count mismatch")

	item := suite.page.Locator(".expense-item").First()
	err = suite.expect.Locator(item.Locator(".expense-details strong")).ToHaveText("Lunch Test")
	require.NoError(suite.T(), err, "description mismatch")

	err = suite.expect.Locator(item.Locator(".expense-amount")).ToContainText("12.50")
	require.NoError(suite.T(), err, "amount mismatch")
}

// TestE2ESuite runs the e2e test suite
func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
