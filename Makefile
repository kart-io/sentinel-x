GO_MOD_NAME := "github.com/kart-io/sentinel-x"
GO_MOD_DOMAIN := $(shell echo $(GO_MOD_NAME) | awk -F '/' '{print $$1}')

# Binary names
API_BINARY=sentinel-api
USER_CENTER_BINARY=sentinel-user-center

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH ?= $(shell git branch --show-current 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Build flags with version injection
LDFLAGS=-ldflags "\
	-X 'github.com/kart-io/version.serviceName=$(API_BINARY)' \
	-X 'github.com/kart-io/version.gitVersion=$(VERSION)' \
	-X 'github.com/kart-io/version.gitCommit=$(GIT_COMMIT)' \
	-X 'github.com/kart-io/version.gitBranch=$(GIT_BRANCH)' \
	-X 'github.com/kart-io/version.buildDate=$(BUILD_DATE)' \
	-s -w"

# Output directory
OUTPUT_DIR=bin


.PHONY: tidy
tidy:
	go mod tidy && go mod vendor

.PHONY: update
update:
	bash hack/update-vendor.sh

.PHONY: update-goagent
update-goagent:
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: publish-goagent
publish-goagent:
	bash hack/sync-to-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: update-logger
update-logger:
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main

.PHONY: publish-logger
publish-logger:
	bash hack/sync-to-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main

# =============================================================================
# API Server Build Commands
# =============================================================================

.PHONY: build
build:
	@echo "Building $(API_BINARY)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(API_BINARY) ./cmd/api
	@echo "Build complete: $(OUTPUT_DIR)/$(API_BINARY)"

.PHONY: build-dev
build-dev:
	@echo "Building $(API_BINARY) (development)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(OUTPUT_DIR)/$(API_BINARY) ./cmd/api
	@echo "Build complete: $(OUTPUT_DIR)/$(API_BINARY)"

.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean complete"

.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: run
run: build
	@echo "Starting $(API_BINARY)..."
	./$(OUTPUT_DIR)/$(API_BINARY)

.PHONY: run-dev
run-dev:
	@echo "Running $(API_BINARY) in development mode..."
	$(GOCMD) run ./cmd/api -c configs/sentinel-api-dev.yaml

.PHONY: run-config
run-config: build
	@echo "Starting $(API_BINARY) with config file..."
	@./$(OUTPUT_DIR)/$(API_BINARY) -c configs/sentinel-api.yaml

# =============================================================================
# User Center Build Commands
# =============================================================================

.PHONY: build-user-center
build-user-center:
	@echo "Building $(USER_CENTER_BINARY)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(USER_CENTER_BINARY) ./cmd/user-center
	@echo "Build complete: $(OUTPUT_DIR)/$(USER_CENTER_BINARY)"

.PHONY: run-user-center
run-user-center: build-user-center
	@echo "Starting $(USER_CENTER_BINARY)..."
	@./$(OUTPUT_DIR)/$(USER_CENTER_BINARY) -c configs/user-center.yaml

.PHONY: version
version: build
	@./$(OUTPUT_DIR)/$(API_BINARY) --version

.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded"

.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(API_BINARY):$(VERSION) -f cmd/api/Dockerfile .
	@echo "Docker image built: $(API_BINARY):$(VERSION)"

# =============================================================================
# Tools
# =============================================================================

.PHONY: tools.install
tools.install:
	@bash scripts/install/install.sh

.PHONY: tools.install.%
tools.install.%:
	@bash scripts/install/install.sh $*

# =============================================================================
# Proto Commands
# =============================================================================

.PHONY: gen.proto
gen.proto:
	@buf generate --path pkg/api

# =============================================================================
# Example Server Commands
# =============================================================================

.PHONY: run-example
run-example:
	go run example/server/example/main.go -c example/server/example/configs/sentinel-example.yaml

.PHONY: run-auth-example
run-auth-example:
	go run example/auth/main.go

# =============================================================================
# Code Quality
# =============================================================================

.PHONY: fmt
fmt:
	@gofumpt -version || go install mvdan.cc/gofumpt@latest
	gofumpt -w -d .
	@gci -v || go install github.com/daixiang0/gci@latest
	gci write -s standard -s default -s 'Prefix($(GO_MOD_DOMAIN))' --skip-generated .

.PHONY: lint
lint:
	golangci-lint run ./...

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help:
	@echo "Sentinel-X Makefile Commands:"
	@echo ""
	@echo "  Build & Run:"
	@echo "    build             - Build API server with version injection"
	@echo "    build-dev         - Build API server without optimization (dev)"
	@echo "    run               - Build and run the API server"
	@echo "    run-dev           - Run in dev mode (no DB/Redis required)"
	@echo "    run-config        - Run with production config (requires DB/Redis)"
	@echo "    clean             - Clean build artifacts"
	@echo "    version           - Display version information"
	@echo ""
	@echo "  User Center:"
	@echo "    build-user-center - Build User Center service"
	@echo "    run-user-center   - Run User Center service"
	@echo ""
	@echo "  Testing:"
	@echo "    test              - Run all tests"
	@echo "    test-coverage     - Run tests with coverage report"
	@echo ""
	@echo "  Dependencies:"
	@echo "    deps              - Download dependencies"
	@echo "    tidy              - Run go mod tidy && go mod vendor"
	@echo "    update            - Update vendor directory"
	@echo ""
	@echo "  Staging Sync:"
	@echo "    update-goagent    - Sync goagent from upstream"
	@echo "    publish-goagent   - Publish goagent to upstream"
	@echo "    update-logger     - Sync logger from upstream"
	@echo "    publish-logger    - Publish logger to upstream"
	@echo ""
	@echo "  Examples:"
	@echo "    run-example       - Run the example server (HTTP:8081, gRPC:9091)"
	@echo "    run-auth-example  - Run the auth/authz demo server (HTTP:8082)"
	@echo ""
	@echo "  Code Quality:"
	@echo "    fmt               - Format code with gofumpt and gci"
	@echo "    lint              - Run golangci-lint"
	@echo ""
	@echo "  Docker:"
	@echo "    docker-build      - Build Docker image"
	@echo ""
	@echo "  Proto:"
	@echo "    gen.proto         - Generate Go code from proto files"
	@echo ""
	@echo "  Help:"
	@echo "    help              - Show this help message"
