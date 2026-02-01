# AGENTS.md - Coding Agent Instructions

## Project Overview
This is a **Go 1.25.6** project - Telegram Bot with ESP32 BLE Scanner integration.
The project uses **PocketBase** (SQLite embedded) for easy data management and real-time features.

## Build/Test/Lint Commands

```bash
# Run the application (uses run.sh to handle permission restrictions)
./run.sh

# Build
GOPATH=$(pwd)/.go GOCACHE=$(pwd)/.cache go build -o app main.go

# Run with go run
GOPATH=$(pwd)/.go GOCACHE=$(pwd)/.cache go run main.go

# Database Migrations (PocketBase Go Migrations)
go run scripts/migrate/main.go

# Collection Setup (Initial Setup)
go run scripts/setup_collections/main.go

# Run all tests
go test ./...

# Format code
go fmt ./...

# Tidy dependencies (Note: may require local permission for cache)
go mod tidy
```

## Code Style Guidelines

### Imports
- Group imports: stdlib → third-party → local packages
- Separate groups with blank line
- Use full import paths for local packages: `med-pulse-bot/package`

Example:
```go
import (
    "context"
    "fmt"
    "log"

    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "med-pulse-bot/config"
)
```

### Formatting
- Use `gofmt` for all Go code
- Use `goimports` for import management
- Maximum line length: 120 characters
- Use tabs for indentation (Go standard)

### Types & Naming
- PascalCase for exported (public) names: `DetectRequest`, `LoadConfig`
- camelCase for unexported (private) names: `handleDetect`, `targetChatID`
- Acronyms in ALLCAPS: `URL`, `ID`, `MAC`, `RSSI`, `API`

### Error Handling
- Use `github.com/pocketbase/pocketbase/core` for backend extensions.
- Always check and handle errors explicitly.
- Use `fmt.Errorf("...: %w", err)` for error wrapping.
- Return errors up the call stack, don't log and swallow
- Never use naked returns

### Context Usage
- Pass `context.Context` as first parameter to functions doing I/O
- Use `context.WithTimeout()` for database operations
- Always call `defer cancel()` after creating context

### Concurrency
- Use goroutines only with clear lifecycle management
- Use channels for communication between goroutines
- Always handle context cancellation in long-running operations
- Run race detector during tests: `go test -race ./...`

### Functions
- Keep functions small and focused (single responsibility)
- Document all exported functions with Go doc comments
- Limit function parameters (use structs for 3+ related params)

## Project Structure

```
med-pulse-bot/
├── main.go              # Application entry point
├── config/              # Configuration loading
│   └── config.go
├── internal/
│   ├── repository/      # Repository interfaces & PocketBase REST implementation
│   ├── services/        # Business logic
│   └── handlers/        # API Handlers
├── bot/                 # Telegram bot logic
│   └── bot.go
├── migrations/          # PocketBase Go migration files (core API)
├── pb_migrations/       # PocketBase JSON migration files
├── scripts/             # Utility scripts (isolated packages)
│   ├── migrate/         # DB Migration script
│   └── setup_collections/ # Collection setup script
├── firmware/            # ESP32 Arduino/PlatformIO code
│   └── scanner.ino
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── .env                 # Environment variables
└── run.sh               # Run script with embedded path exports
```

## Dependencies

Key packages in use:
- `github.com/pocketbase/pocketbase` - Core Backend & Auth
- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram API
- `github.com/joho/godotenv` - Environment variable loading

## Testing

- Write table-driven tests for multiple test cases
- Name test files: `xxx_test.go`
- Name test functions: `TestXxx` (camelCase after Test)
- Use subtests: `t.Run("description", func(t *testing.T){})`
- Target 80%+ test coverage

## Security

- Never commit `.env` files or secrets.
- Use `POCKETBASE_TOKEN` from the environment for admin operations.
- All database operations should be validated against the PocketBase schema.
- Validate all input data

## Environment Variables

Required in `.env`:
- `POCKETBASE_URL` - URL of the PocketBase instance (e.g., http://192.168.100.100:8090)
- `POCKETBASE_TOKEN` - Admin Auth Token for schema changes
- `TELEGRAM_BOT_TOKEN` - Bot token from @BotFather
- `AUTHORIZED_CHAT_ID` - Telegram chat ID for notifications
