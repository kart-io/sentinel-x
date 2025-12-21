# ==============================================================================
# Makefile helper functions for swagger
#

.PHONY: swagger.serve
swagger.serve:
	@$(GO) run cmd/swagger/main.go

.PHONY: swagger.gen
swagger.gen:
	@echo "===========> Generating swagger API docs"
	@if [ -d "cmd/api" ]; then \
		echo "Generating docs for api..."; \
		cd cmd/api && swag init -g main.go -o ../../api/swagger/apisvc --instanceName apisvc --packageName apisvc --parseDependency --parseInternal -d .,../../internal/api; \
	fi
	@if [ -d "cmd/user-center" ]; then \
		echo "Generating docs for user-center..."; \
		cd cmd/user-center && swag init -g main.go -o ../../api/swagger/user-center --instanceName usercenter --packageName usercenter --parseDependency --parseInternal -d .,../../internal/user-center; \
	fi
