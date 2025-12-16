# ==============================================================================
# Makefile helper functions for generate
#

.PHONY: gen.proto
.PHONY: gen.proto
gen.proto: tools.verify.buf tools.verify.protoc-gen-go tools.verify.protoc-gen-go-grpc tools.verify.protoc-gen-validate tools.verify.protoc-gen-openapiv2 tools.verify.protoc-go-inject-tag ## Generate Proto code.
	@echo "===========> Generating protos"
	@buf generate --path pkg/api
	@echo "===========> Injecting tags"
	@if ls pkg/api/user-center/v1/*.pb.go > /dev/null 2>&1; then \
		protoc-go-inject-tag -input=pkg/api/user-center/v1/*.pb.go; \
	fi

.PHONY: gen.clean
gen.clean: ## Clean generated protobuf files.
	@find pkg/api -name "*.pb.go" -delete
	@find pkg/api -name "*.pb.validate.go" -delete
	@find pkg/api -name "*.pb.gw.go" -delete
	@find pkg/api -name "*.swagger.json" -delete

.PHONY: gen.k8s
gen.k8s: tools.verify.client-gen tools.verify.lister-gen tools.verify.informer-gen tools.verify.deepcopy-gen tools.verify.controller-gen ## Generate Kubernetes code (client, lister, informer, deepcopy).
	@echo "===========> Generating kubernetes code"
	@# Usage: make gen.k8s GROUPS_VERSIONS="group:version ..."
	@# Example: make gen.k8s GROUPS_VERSIONS="db:v1 log:v1"
	@if [ -n "$(GROUPS_VERSIONS)" ]; then \
		bash hack/update-codegen.sh $(GROUPS_VERSIONS); \
	else \
		echo "Warning: GROUPS_VERSIONS is empty. Skipping k8s generation."; \
		echo "Usage: make gen.k8s GROUPS_VERSIONS=\"group:version\""; \
	fi
