# Makefile for Sentinel-X API Server

# Binary name
BINARY_NAME=sentinel-api

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH ?= $(shell git branch --show-current 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
SERVICE_NAME=sentinel-api

# Build flags
LDFLAGS=-ldflags "\
	-X 'github.com/kart-io/version.serviceName=$(SERVICE_NAME)' \
	-X 'github.com/kart-io/version.gitVersion=$(VERSION)' \
	-X 'github.com/kart-io/version.gitCommit=$(GIT_COMMIT)' \
	-X 'github.com/kart-io/version.gitBranch=$(GIT_BRANCH)' \
	-X 'github.com/kart-io/version.buildDate=$(BUILD_DATE)' \
	-s -w"

# Output directory
OUTPUT_DIR=bin

.PHONY: all build clean test run help version deps

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/api
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

## build-dev: Build without optimization for development
build-dev:
	@echo "Building $(BINARY_NAME) (development)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/api
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./cmd/api/...

## run: Build and run the server
run: build
	@echo "Starting $(BINARY_NAME)..."
	./$(OUTPUT_DIR)/$(BINARY_NAME)

## run-dev: Run in development mode (without building)
run-dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	$(GOCMD) run ./cmd/api

## run-config: Run with example config file
run-config: build
	@echo "Starting $(BINARY_NAME) with config file..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) -c configs/sentinel-api.yaml

## version: Display version information
version: build
	@./$(OUTPUT_DIR)/$(BINARY_NAME) --version

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w ./cmd/api
	@echo "Formatting complete"

## lint: Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./cmd/api/... || echo "golangci-lint not installed"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) -f cmd/api/Dockerfile .
	@echo "Docker image built: $(BINARY_NAME):$(VERSION)"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
