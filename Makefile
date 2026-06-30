.PHONY: build run test lint fmt clean docker-up docker-down docker-logs compose-up staging-up staging-down staging-logs staging-compose-up deps

APP_NAME=mcp-api
BINARY_NAME=mcp-api

build:
	@echo "Building $(APP_NAME)..."
	@go build -o bin/$(BINARY_NAME) .

run:
	@echo "Running $(APP_NAME)..."
	@go run .

test:
	@echo "Running tests..."
	@go test -v ./... -count=1

lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
	fi

fmt:
	@echo "Formatting code..."
	@gofmt -s -w .
	@go mod tidy

clean:
	@echo "Cleaning..."
	@rm -rf bin/

docker-up:
	@echo "Starting services with docker-compose..."
	@docker compose --env-file stack.env up -d

docker-down:
	@echo "Stopping services..."
	@docker compose --env-file stack.env down

docker-logs:
	@docker compose --env-file stack.env logs -f

compose-up:
	@echo "Running docker compose up --build..."
	docker compose --env-file stack.env up --build

staging-up:
	@echo "Starting staging services..."
	@docker compose -f docker-compose.staging.yml --env-file stack.staging.env up -d

staging-down:
	@echo "Stopping staging services..."
	@docker compose -f docker-compose.staging.yml --env-file stack.staging.env down

staging-logs:
	@docker compose -f docker-compose.staging.yml --env-file stack.staging.env logs -f

staging-compose-up:
	@echo "Running staging docker compose up --build..."
	docker compose -f docker-compose.staging.yml --env-file stack.staging.env up --build

deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed!"
