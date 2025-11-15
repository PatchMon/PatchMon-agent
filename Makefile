# Makefile for PatchMon Agent

# Build variables
BINARY_NAME=patchmon-agent
BUILD_DIR=build
# Use hardcoded version instead of git tags
VERSION=1.3.3
# Strip debug info and set version variable
LDFLAGS=-ldflags "-s -w -X patchmon-agent/internal/version.Version=$(VERSION)"
# Disable VCS stamping
BUILD_FLAGS=-buildvcs=false

#=============================================================================
# Build Variables
#=============================================================================
BINARY_NAME = patchmon-agent
BUILD_DIR   = build
VERSION     = 1.3.3

# Build flags
LDFLAGS     = -ldflags "-s -w -X patchmon-agent/internal/version.Version=$(VERSION)"
BUILD_FLAGS = -buildvcs=false

#=============================================================================
# Go Variables
#=============================================================================
GOBASE           = $(shell pwd)
GOBIN            = $(GOBASE)/$(BUILD_DIR)
GO_CMD          ?= $(shell which go)
GOLANGCI_LINT_CMD ?= $(shell which golangci-lint || echo /usr/local/go/bin/golangci-lint)

#=============================================================================
# Targets
#=============================================================================

.PHONY: all
all: build

#------------------------------------------------------------------------------
# Build Targets
#------------------------------------------------------------------------------

.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GO_CMD) build $(BUILD_FLAGS) $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME) ./cmd/patchmon-agent

.PHONY: build-all
build-all:
	@echo "Building for multiple architectures..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO_CMD) build $(BUILD_FLAGS) $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-amd64 ./cmd/patchmon-agent
	@GOOS=linux GOARCH=386 CGO_ENABLED=0 $(GO_CMD) build $(BUILD_FLAGS) $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-386 ./cmd/patchmon-agent
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO_CMD) build $(BUILD_FLAGS) $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-arm64 ./cmd/patchmon-agent
	@GOOS=linux GOARCH=arm CGO_ENABLED=0 $(GO_CMD) build $(BUILD_FLAGS) $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-arm ./cmd/patchmon-agent

#------------------------------------------------------------------------------
# Development Targets
#------------------------------------------------------------------------------

.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@$(GO_CMD) mod download
	@$(GO_CMD) mod tidy

.PHONY: test
test:
	@echo "Running tests..."
	@$(GO_CMD) test -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@$(GO_CMD) test -v -coverprofile=coverage.out ./...
	@$(GO_CMD) tool cover -html=coverage.out -o coverage.html

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@$(GO_CMD) fmt ./...

.PHONY: lint
lint:
	@echo "Linting code..."
	@PATH="/usr/local/bin/go/bin:$$PATH" GOFLAGS="$(BUILD_FLAGS)" $(GOLANGCI_LINT_CMD) run

#------------------------------------------------------------------------------
# Maintenance Targets
#------------------------------------------------------------------------------

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(GOBIN)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         Build the application"
	@echo "  build-all     Build for multiple architectures"
	@echo "  deps          Install dependencies"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage"
	@echo "  fmt           Format code"
	@echo "  lint          Lint code"
	@echo "  clean         Clean build artifacts"
	@echo "  install       Install binary to /usr/local/bin"
	@echo "  help          Show this help message"
	@echo ""
