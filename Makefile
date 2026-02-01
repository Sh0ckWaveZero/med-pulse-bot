.PHONY: test test-race build run fmt lint clean

# Build the application
build:
	go build -o app main.go

# Run the application
run:
	./run.sh

# Run all tests
test:
	go test ./...

# Run tests with race detector (recommended)
test-race:
	go test -race ./...

# Run tests with verbose output
test-v:
	go test -v ./internal/services ./internal/handlers

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run ./...

# Tidy dependencies
tidy:
	go mod tidy

# Download dependencies
download:
	go mod download

# Clean build artifacts
clean:
	rm -f app

# Run all checks (format, test, build)
check: fmt test build

# Development mode - auto restart on changes (requires air)
dev:
	air

# Database Migration
migrate-db:
	@echo "🔧 Running database migration..."
	@./scripts/migrate_add_target_device_fields.sh

migrate-db-go:
	@echo "🔧 Running database migration (Go)..."
	@go run scripts/migrate/main.go

# Docker operations
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-restart:
	docker-compose restart

docker-logs:
	docker-compose logs -f app

docker-logs-all:
	docker-compose logs -f
