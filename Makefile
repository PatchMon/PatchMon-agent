# Makefile for PatchMon Agent

# Build variables
BINARY_NAME=patchmon-agent
BUILD_DIR=build
# Get version from git tags, fallback to "dev" if not available
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
# Strip debug info and set version variable
LDFLAGS=-ldflags "-s -w -X patchmon-agent/internal/config.AgentVersion=$(VERSION)"

# Go variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/$(BUILD_DIR)

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME) ./cmd/patchmon-agent

# Build for multiple architectures
.PHONY: build-all
build-all:
	@echo "Building for multiple architectures..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-amd64 ./cmd/patchmon-agent
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-arm64 ./cmd/patchmon-agent
	@GOOS=linux GOARCH=386 go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-386 ./cmd/patchmon-agent

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@golangci-lint run

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install the binary to system
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(GOBIN)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)

# Development server (for testing)
.PHONY: dev
dev: build
	@echo "Running in development mode..."
	@./$(BUILD_DIR)/$(BINARY_NAME) --log-level debug

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        Build the application"
	@echo "  build-all    Build for multiple architectures"
	@echo "  deps         Install dependencies"
	@echo "  test         Run tests"
	@echo "  test-coverage Run tests with coverage"
	@echo "  fmt          Format code"
	@echo "  lint         Lint code"
	@echo "  clean        Clean build artifacts"
	@echo "  install      Install binary to /usr/local/bin"
	@echo "  dev          Run in development mode"
	@echo "  help         Show this help message"
	@echo ""
