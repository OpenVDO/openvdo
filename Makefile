.PHONY: help build run dev test clean migrate-up migrate-down docker-up docker-down deps tidy

# Variables
APP_NAME := openvdo
BUILD_DIR := ./bin
MAIN_FILE := ./cmd/server/main.go

# Default target
help:
	@echo "Available commands:"
	@echo "  build       - Build the application"
	@echo "  run         - Run the application"
	@echo "  dev         - Run with hot reload using air"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  migrate-up  - Run database migrations"
	@echo "  migrate-down- Rollback database migrations"
	@echo "  docker-up   - Start services with docker-compose"
	@echo "  docker-down - Stop services with docker-compose"
	@echo "  deps        - Download dependencies"
	@echo "  tidy        - Clean up dependencies"
	@echo "  swagger     - Generate Swagger documentation"
	@echo "  tools       - Install development tools"

# Install dependencies
deps:
	go mod download

# Tidy dependencies
tidy:
	go mod tidy

# Install development tools
tools:
	go install github.com/air-verse/air@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/swaggo/swag/cmd/swag@latest

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	go run $(MAIN_FILE)

# Development mode with hot reload
dev:
	@echo "Running $(APP_NAME) in development mode with hot reload..."
	@if ! command -v air >/dev/null 2>&1; then \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	$(shell go env GOPATH)/bin/air

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -rf tmp/

# Database migration up
migrate-up:
	@echo "Running database migrations..."
	@if ! command -v migrate >/dev/null 2>&1; then \
		echo "Installing migrate..."; \
		go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	fi
	$(shell go env GOPATH)/bin/migrate -path ./migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

# Database migration down
migrate-down:
	@echo "Rolling back database migrations..."
	@if ! command -v migrate >/dev/null 2>&1; then \
		echo "Installing migrate..."; \
		go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	fi
	$(shell go env GOPATH)/bin/migrate -path ./migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down

# Create new migration
migration-new:
	@read -p "Enter migration name: " name; \
	$(shell go env GOPATH)/bin/migrate create -ext sql -dir ./migrations -seq $$name

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	@if ! command -v swag >/dev/null 2>&1; then \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	$(shell go env GOPATH)/bin/swag init -g cmd/server/main.go -o docs

# Start docker services
docker-up:
	@echo "Starting docker services..."
	docker-compose up -d postgres redis

# Stop docker services
docker-down:
	@echo "Stopping docker services..."
	docker-compose down

# Start all services with docker
docker-full:
	@echo "Starting all services with docker..."
	docker-compose up --build

# Setup project
setup: deps tools
	@echo "Setting up the project..."
	cp .env.example .env
	@echo "Setup complete! Please configure your .env file."

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Check for security issues
security:
	@echo "Checking for security issues..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...