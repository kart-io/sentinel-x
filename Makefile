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
include scripts/make-rules/run.mk
include scripts/make-rules/update.mk
include scripts/make-rules/deploy.mk

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

.PHONY: fmt
fmt: go.fmt ## Format source code.

.PHONY: lint
lint: go.lint ## Run lint.

.PHONY: tidy
tidy: ## Tidy go.mod and vendor.
	$(GO) mod tidy && $(GO) mod vendor

## @ Development

## @ Help

.PHONY: help
help: Makefile ## Display this help info.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<TARGETS> <OPTIONS>\033[0m\n\n\033[35mTargets:\033[0m\n"} /^[0-9A-Za-z._-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^\$$\([0-9A-Za-z_-]+\):.*?##/ { gsub("_","-", $$1); printf "  \033[36m%-45s\033[0m %s\n", tolower(substr($$1, 3, length($$1)-7)), $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo -e "$$USAGE_OPTIONS"

.PHONY: targets
targets: Makefile ## Show all Sub-makefile targets.
	@for mk in `echo $(MAKEFILE_LIST) | sed 's/Makefile //g'`; do echo -e \\n\\033[35m$$mk\\033[0m; awk -F':.*##' '/^[0-9A-Za-z._-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^\$$\([0-9A-Za-z_-]+\):.*?##/ { gsub("_","-", $$1); printf "  \033[36m%-45s\033[0m %s\n", tolower(substr($$1, 3, length($$1)-7)), $$2 }' $$mk;done;
