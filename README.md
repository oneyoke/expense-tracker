# Expense Tracker

A simple, mobile-first expense tracking web application. Built with **Go** and **HTMX**.

## Features

- 📱 **Mobile-First Design**: Optimized for mobile usage with a responsive layout.
- ⚡ **Fast & Lightweight**: Server-side rendering with Go and HTMX for smooth interactions.
- 💰 **Expense Tracking**: Quick expense entry with a custom keypad.
- 📊 **Overview**: Daily grouping and monthly summaries.
- 🎨 **Modern UI**: Clean, mobile-first design with custom CSS.

## Project Structure

```
├── cmd/
│   └── server/           # Application entry point
├── e2e/                  # End-to-end tests
├── internal/
│   ├── handlers/         # HTTP handlers and view logic
│   ├── models/           # Data models
│   └── storage/          # Database layer (SQLite)
├── web/
│   ├── static/           # Static assets (CSS)
│   └── templates/        # HTML templates
├── Dockerfile            # Multi-stage build
├── docker-compose.yml    # Docker composition
└── expenses.db           # SQLite database (ignored by git)
```

## Prerequisites

- **Go 1.25+** (for local development)
- **Docker** (optional, for containerized run)

## Quick Start

### Using Docker (Recommended)

```bash
docker-compose up --build
```
The app will be available at [http://localhost:8080](http://localhost:8080).

### Running Locally

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Run the application:
   ```bash
   go run ./cmd/server
   ```

3. Open your browser at [http://localhost:8080](http://localhost:8080).

## Configuration

The application can be configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | The port to listen on | `8080` |
| `DB_PATH` | Path to the SQLite database file | `expenses.db` |

## Testing

### Unit Tests

Run unit tests for internal packages:

```bash
go test ./internal/...
```

### End-to-End (E2E) Tests

E2E tests use [Playwright for Go](https://github.com/playwright-community/playwright-go).

1. Install Playwright browsers (first time only):
   ```bash
   go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps
   ```

2. Run the tests:
   ```bash
   go test -v ./e2e/...
   ```

## Tech Stack

- **Backend**: Go (Golang)
- **Database**: SQLite (via [modernc.org/sqlite](https://modernc.org/sqlite) - CGo-free)
- **Frontend**: 
  - HTML Templates (Go `html/template`)
  - [HTMX](https://htmx.org) for interactivity
  - Custom CSS for styling
- **Testing**:
  - [Playwright Go](https://github.com/playwright-community/playwright-go) for E2E testing

## License

MIT
