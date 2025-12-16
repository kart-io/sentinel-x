# ==============================================================================
# Makefile helper functions for running services
#
# ENV: The environment to run in (dev, staging, prod). Default: dev.
#      It maps to config file suffix: configs/sentinel-<APP>-<ENV>.yaml
# ==============================================================================

ENV ?= dev

.PHONY: run
BIN ?= api
run: run.$(BIN) ## Run specific binary. Usage: make run BIN=api ENV=staging

# Define run_service macro parameter 1 is binary name, parameter 2 is run command
define run_service
	@echo "===========> Running $(1) in $(ENV) mode"
	@if [ -f configs/sentinel-$(1)-$(ENV).yaml ]; then \
		$(2) -c configs/sentinel-$(1)-$(ENV).yaml; \
	elif [ -f configs/$(1)-$(ENV).yaml ]; then \
		$(2) -c configs/$(1)-$(ENV).yaml; \
	elif [ -f configs/$(1).yaml ]; then \
		$(2) -c configs/$(1).yaml; \
	elif [ -f example/server/example/configs/sentinel-$(1).yaml ]; then \
		go run example/server/example/main.go -c example/server/example/configs/sentinel-$(1).yaml; \
	else \
		echo "Error: Config file not found for $(1) in $(ENV) mode"; \
		exit 1; \
	fi
endef

.PHONY: run.%
run.%: go.build.$(PLATFORM).% ## Run specified binary.
	$(call run_service,$*,$(LOCALBIN)/$*)

.PHONY: run.go
run.go: run.go.$(BIN) ## Run specific binary using go run. Usage: make run.go BIN=api ENV=staging

.PHONY: run.go.%
run.go.%: ## Run specified binary using go run.
	$(call run_service,$*,go run ./cmd/$*)
