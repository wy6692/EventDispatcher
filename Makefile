.PHONY: all build test clean run simulate docker-build docker-up docker-down lint

APP_NAME := dispatcher
BIN_DIR := bin
MAIN_PKG := ./cmd/dispatcher

all: test build

build:
	@echo "==> Building binary..."
	@go build -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PKG)

test:
	@echo "==> Running tests..."
	@go test -v -race -cover ./...

run:
	@echo "==> Running application..."
	@go run $(MAIN_PKG)

clean:
	@echo "==> Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)

simulate:
	@echo "==> Building and running simulator..."
	@go build -o $(BIN_DIR)/simulate ./cmd/simulate
	@$(BIN_DIR)/simulate

docker-build:
	@echo "==> Building Docker image..."
	@docker build -t event-dispatcher:latest .

docker-up:
	@echo "==> Starting container with docker-compose..."
	@docker compose up -d

docker-down:
	@echo "==> Stopping container..."
	@docker compose down

lint:
	@echo "==> Running golangci-lint (if installed)..."
	@golangci-lint run || go vet ./...
