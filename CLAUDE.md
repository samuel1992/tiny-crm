# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Run the server (default port 8080)
go run .

# Run on custom port
go run . --port 9090

# Build binary
go build -o tinycrm

# Create a user (required before first login)
go run . adduser <username> <password>

# Run all tests
go test ./...

# Run a single test
go test -run TestCompanyCreate ./...

# Cross-compile for Linux (requires Docker)
./build-linux.sh
```

## Architecture

Single-package Go application (`package main`) with three source files:

- **main.go** — HTTP server setup, route registration, and all API handler functions. Routes use Go 1.22+ `http.ServeMux` pattern matching (e.g., `GET /api/companies/{companyId}`). All API routes are wrapped with `basicAuthMiddleware`. The `setupRoutes(testing bool)` function accepts a bool to bypass auth in tests.
- **repository.go** — GORM models (structs with JSON/GORM tags) and all database operations. Uses SQLite (`tinycrm.db`). Models: `User`, `Company`, `Product`, `RemitInformation` (with `RemitInformationLine` children), `Invoice` (with `InvoiceLine` children, references `Company` for both company and client, `RemitInformation`, and `Product` via lines). Schema is auto-migrated on startup.
- **auth.go** — Basic HTTP authentication middleware using bcrypt password hashing.

The frontend is a single-page app served from `templates/index.html`. Invoice HTML templates in `templates/invoices/` use Go's `html/template` and are rendered server-side via `GET /api/invoices/{id}/open?template=<filename>`.

## Key Patterns

- Tests use `httptest.NewServer` with an in-memory SQLite database (`:memory:`). Auth is bypassed by passing `testing=true` to `setupRoutes`.
- Update operations for entities with child records (Invoice, RemitInformation) delete all existing children then re-create them in a transaction.
- The `Invoice` model has computed methods (`SubTotal()`, `Total()`, `Identification()`, `DueMonth()`, `Repr()`) used in templates.
- `Company` is used for both the issuing company and the client on an invoice (separate `CompanyID` and `ClientID` foreign keys).
