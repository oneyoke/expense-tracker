# Expense Tracker

A simple, mobile-first expense tracking web application. Built with **Go** and **HTMX**.

## Features

- 📱 **Mobile-First Design**: Optimized for mobile usage with a responsive layout.
- ⚡ **Fast & Lightweight**: Server-side rendering with Go and HTMX for smooth interactions.
- 💰 **Quick Entry**: Specialized keypad interface for rapid expense recording.
- 📋 **Expense History**: Chronological feed of expenses grouped by day.
- 📊 **Statistics**: Monthly spending breakdown by category with percentages and totals.
- 🎨 **Modern UI**: Clean, minimalistic interface using custom CSS.

## Project Structure

```
├── cmd/
│   ├── adduser/          # CLI tool for user management
│   └── server/           # Application entry point
├── e2e/                  # End-to-end tests
├── internal/
│   ├── auth/             # Authentication logic
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
| `SECURE_COOKIE` | Set to `true` to use secure cookies (HTTPS only) | `false` |
| `ADMIN_USER` | Username for the initial admin user (created on first run) | `admin` |
| `ADMIN_PASSWORD` | Password for the initial admin user. If not set, a random one is generated and printed to logs. | Random |

## User Management

### Initial Setup (Bootstrapping)

On the first run, if no users exist in the database, the application will attempt to create an initial admin user.
- If `ADMIN_USER` and `ADMIN_PASSWORD` environment variables are set, it uses those credentials.
- If not set, it creates a user `admin` with a **randomly generated password**, which is printed to the server logs.

### CLI Tool

To add a new user manually, use the `adduser` CLI tool:

```bash
go run ./cmd/adduser -user <username> -password <password>
```

You can also specify the database path if it differs from the default `expenses.db`:

```bash
go run ./cmd/adduser -user <username> -password <password> -db path/to/expenses.db
```

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
