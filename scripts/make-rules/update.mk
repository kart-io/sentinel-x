# ==============================================================================
# Makefile helper functions for update
#
# ==============================================================================

.PHONY: update
update: ## Update vendor dependencies.
	bash hack/update-vendor.sh

.PHONY: update-goagent
update-goagent: ## Sync goagent from upstream (Staging).
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: update-logger
update-logger: ## Sync logger from upstream (Staging).
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main
