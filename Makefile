# ==============================================================================
# Makefile for Sentinel-X
# ==============================================================================

# Project Root
PROJ_ROOT_DIR := $(shell pwd)
GO_MOD_NAME := "github.com/kart-io/sentinel-x"
GO_MOD_DOMAIN := $(shell echo $(GO_MOD_NAME) | awk -F '/' '{print $$1}')

# BINS can be overridden by "make build BINS=sentinel-api"
BINS ?= api user-center

# Include make rules
include scripts/make-rules/common.mk
include scripts/make-rules/golang.mk
include scripts/make-rules/image.mk
include scripts/make-rules/tools.mk
include scripts/make-rules/gen.mk

# ==============================================================================
# Targets
# ==============================================================================

.PHONY: all
all: build

.PHONY: build
build: go.build

.PHONY: clean
clean: go.clean

.PHONY: test
test: go.test

.PHONY: test-coverage
test-coverage: go.test.cover

.PHONY: lint
lint: go.lint

.PHONY: fmt
fmt: go.fmt

.PHONY: tidy
tidy:
	$(GO) mod tidy && $(GO) mod vendor

.PHONY: update
update:
	bash hack/update-vendor.sh

# Shorthand for building specific images
.PHONY: image
image: image.build

# Helpers for running components (dev mode)
.PHONY: run-api
run-api: go.build.api
	@echo "Starting sentinel-api..."
	./bin/api -c configs/sentinel-api-dev.yaml

.PHONY: run-user-center
run-user-center: go.build.user-center
	@echo "Starting sentinel-user-center..."
	./bin/user-center -c configs/user-center.yaml

# Staging sync for internal dev
.PHONY: update-goagent
update-goagent:
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: update-logger
update-logger:
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main

# Example server commands
.PHONY: run-example
run-example:
	go run example/server/example/main.go -c example/server/example/configs/sentinel-example.yaml

.PHONY: help
help:
	@echo "Sentinel-X Makefile Commands:"
	@echo "  build             - Build all binaries ($(BINS))"
	@echo "  clean             - Clean build artifacts"
	@echo "  test              - Run tests"
	@echo "  lint              - Run linters"
	@echo "  fmt               - Format code"
	@echo "  image             - Build Docker images"
	@echo "  gen.proto         - Generate Proto code"
	@echo "  run-api           - Run API server (Dev)"
	@echo "  run-user-center   - Run User Center (Dev)"
	@echo "  deps              - Manage dependencies"
