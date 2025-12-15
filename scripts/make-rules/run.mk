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

.PHONY: run.%
run.%: go.build.$(PLATFORM).% ## Run specified binary.
	$(eval BINARY := $*)
	@echo "===========> Running $(BINARY) in $(ENV) mode"
	@if [ -f configs/sentinel-$(BINARY)-$(ENV).yaml ]; then \
		$(LOCALBIN)/$(BINARY) -c configs/sentinel-$(BINARY)-$(ENV).yaml; \
	elif [ -f configs/$(BINARY)-$(ENV).yaml ]; then \
		$(LOCALBIN)/$(BINARY) -c configs/$(BINARY)-$(ENV).yaml; \
	elif [ -f configs/$(BINARY).yaml ]; then \
		$(LOCALBIN)/$(BINARY) -c configs/$(BINARY).yaml; \
	elif [ -f example/server/example/configs/sentinel-$(BINARY).yaml ]; then \
		go run example/server/example/main.go -c example/server/example/configs/sentinel-$(BINARY).yaml; \
	else \
		echo "Error: Config file not found for $(BINARY) in $(ENV) mode"; \
		exit 1; \
	fi

# Support old hyphenated targets for backward compatibility if needed,
# or strictly switch to dot notation.
# The user asked to "extract", so consistent dot notation is better.
