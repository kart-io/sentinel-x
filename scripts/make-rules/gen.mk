# ==============================================================================
# Makefile helper functions for generate
#

.PHONY: gen.proto
gen.proto: tools.verify.buf tools.verify.protoc-gen-go tools.verify.protoc-gen-go-grpc tools.verify.protoc-gen-validate tools.verify.protoc-gen-openapi
	@echo "===========> Generating protos"
	@buf generate --path pkg/api
