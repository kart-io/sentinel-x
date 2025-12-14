# ==============================================================================
# Makefile helper functions for tools
#

TOOLS_DIR ?= $(OUTPUT_DIR)/tools
TOOLS_INSTALL_Script := $(SCRIPTS_DIR)/install/install.sh

.PHONY: tools.install
tools.install: ## Install all tools.
	@bash $(TOOLS_INSTALL_Script)

.PHONY: tools.install.%
tools.install.%: ## Install specific tool.
	@bash $(TOOLS_INSTALL_Script) $*

.PHONY: tools.verify.%
tools.verify.%: ## Verify specific tool is installed.
	@if ! which $* &>/dev/null; then \
		echo "$* is not installed. Installing..."; \
		$(MAKE) tools.install.$*; \
	fi
