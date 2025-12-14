# ==============================================================================
# Makefile for Sentinel-X
# ==============================================================================

# Project Root
PROJ_ROOT_DIR := $(shell pwd)
GO_MOD_NAME := "github.com/kart-io/sentinel-x"
GO_MOD_DOMAIN := $(shell echo $(GO_MOD_NAME) | awk -F '/' '{print $$1}')

# BINS can be overridden by "make build BINS=sentinel-api"
BINS ?= api user-center

# Options
define USAGE_OPTIONS

\033[35mOptions:\033[0m
  BINS             The binaries to build. Default is all of cmd.
                   Example: make build BINS="api"
  IMAGES           Backend images to make.
                   Example: make image IMAGES="api"
  V                Set to 1 enable verbose build. Default is 0.
endef
export USAGE_OPTIONS

# Include make rules
include scripts/make-rules/common.mk
include scripts/make-rules/golang.mk
include scripts/make-rules/image.mk
include scripts/make-rules/tools.mk
include scripts/make-rules/gen.mk

# ==============================================================================
# Targets
# ==============================================================================

## @ Build

.PHONY: all
all: build ## Build all binaries.

.PHONY: build
build: tidy go.build ## Build source code for host platform.

.PHONY: clean
clean: go.clean ## Clean build artifacts.

.PHONY: deps
deps: tidy tools.install ## Install dependencies and tools.

.PHONY: tidy
tidy: ## Tidy go.mod and vendor.
	$(GO) mod tidy && $(GO) mod vendor

.PHONY: update
update: ## Update vendor dependencies.
	bash hack/update-vendor.sh

## @ Image

.PHONY: image
image: image.build ## Build docker images.

.PHONY: image.multiarch
image.multiarch: image.build.multiarch ## Build docker images for multiple platforms.

.PHONY: push
push: image.push ## Build docker images and push to registry.

.PHONY: push.multiarch
push.multiarch: image.push.multiarch ## Build docker images for multiple platforms and push to registry.

## @ Test & Quality

.PHONY: test
test: go.test ## Run unit tests.

.PHONY: test-coverage
test-coverage: go.test.cover ## Run unit tests with coverage.

.PHONY: lint
lint: tidy go.lint ## Run linters.

.PHONY: fmt
fmt: go.fmt ## Format source code.

.PHONY: verify-sonic
verify-sonic: ## Verify Sonic JSON integration.
	bash scripts/verify-sonic.sh

## @ Code Generation

.PHONY: clean.proto
clean.proto: gen.clean ## Clean generated protobuf files.

## @ Development

.PHONY: run-api
run-api: go.build.api ## Run API server (Dev).
	@echo "Starting sentinel-api..."
	./bin/api -c configs/sentinel-api-dev.yaml

.PHONY: run-user-center
run-user-center: go.build.user-center ## Run User Center (Dev).
	@echo "Starting sentinel-user-center..."
	./bin/user-center -c configs/user-center.yaml

.PHONY: update-goagent
update-goagent: ## Sync goagent from upstream (Staging).
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: update-logger
update-logger: ## Sync logger from upstream (Staging).
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main

.PHONY: run-example
run-example: ## Run example server.
	go run example/server/example/main.go -c example/server/example/configs/sentinel-example.yaml

## @ Help

.PHONY: help
help: Makefile ## Display this help info.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<TARGETS> <OPTIONS>\033[0m\n\n\033[35mTargets:\033[0m\n"} /^[0-9A-Za-z._-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^\$$\([0-9A-Za-z_-]+\):.*?##/ { gsub("_","-", $$1); printf "  \033[36m%-45s\033[0m %s\n", tolower(substr($$1, 3, length($$1)-7)), $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo -e "$$USAGE_OPTIONS"

.PHONY: targets
targets: Makefile ## Show all Sub-makefile targets.
	@for mk in `echo $(MAKEFILE_LIST) | sed 's/Makefile //g'`; do echo -e \\n\\033[35m$$mk\\033[0m; awk -F':.*##' '/^[0-9A-Za-z._-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^\$$\([0-9A-Za-z_-]+\):.*?##/ { gsub("_","-", $$1); printf "  \033[36m%-45s\033[0m %s\n", tolower(substr($$1, 3, length($$1)-7)), $$2 }' $$mk;done;
