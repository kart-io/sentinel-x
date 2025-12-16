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
