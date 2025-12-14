# ==============================================================================
# Makefile helper functions for generate
#

.PHONY: gen.proto
gen.proto: tools.verify.buf tools.verify.protoc-gen-go tools.verify.protoc-gen-go-grpc tools.verify.protoc-gen-validate tools.verify.protoc-gen-openapiv2 ## Generate Proto code.
	@echo "===========> Generating protos"
	@buf generate --path pkg/api

.PHONY: gen.clean
gen.clean: ## Clean generated protobuf files.
	@rm -rf pkg/api/**/*.pb.go
	@rm -rf pkg/api/**/*.pb.validate.go
	@rm -rf pkg/api/**/*.pb.gw.go
	@rm -rf pkg/api/**/*.swagger.json
