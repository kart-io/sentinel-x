GO_MOD_NAME := "github.com/kart-io/sentinel-x"
GO_MOD_DOMAIN := $(shell echo $(GO_MOD_NAME) | awk -F '/' '{print $$1}')


.PHONY: tidy
tidy:
	go mod tidy && go mod vendor

.PHONY: update
update:
	bash hack/update-vendor.sh

.PHONY: update-goagent
update-goagent:
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: publish-goagent
publish-goagent:
	bash hack/sync-to-upstream.sh staging/src/github.com/kart-io/goagent https://github.com/kart-io/goagent master

.PHONY: update-logger
update-logger:
	bash hack/sync-from-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main

.PHONY: publish-logger
publish-logger:
	bash hack/sync-to-upstream.sh staging/src/github.com/kart-io/logger https://github.com/kart-io/logger main

# Example server commands
.PHONY: run-example
run-example:
	go run example/server/example/main.go -c example/server/example/configs/sentinel-example.yaml

.PHONY: run-auth-example
run-auth-example:
	go run example/auth/main.go

.PHONY: fmt
fmt:
	@gofumpt -version || go install mvdan.cc/gofumpt@latest
	gofumpt -w -d .
	@gci -v || go install github.com/daixiang0/gci@latest
	#gci write -s standard -s default -s 'Prefix($(GO_MOD_DOMAIN))' -s 'Prefix($(GO_MOD_NAME))' --skip-generated .
	gci write -s standard -s default -s 'Prefix($(GO_MOD_DOMAIN))' --skip-generated .

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: help
help:
	@echo "Sentinel-X Makefile Commands:"
	@echo ""
	@echo "  Vendor Management:"
	@echo "    tidy              - Run go mod tidy && go mod vendor"
	@echo "    update            - Update vendor directory"
	@echo ""
	@echo "  Goagent Sync:"
	@echo "    update-goagent    - Sync goagent from upstream"
	@echo "    publish-goagent   - Publish goagent to upstream"
	@echo ""
	@echo "  Logger Sync:"
	@echo "    update-logger     - Sync logger from upstream"
	@echo "    publish-logger    - Publish logger to upstream"
	@echo ""
	@echo "  Example Server:"
	@echo "    run-example       - Run the example server (HTTP:8081, gRPC:9091)"
	@echo "    run-auth-example  - Run the auth/authz demo server (HTTP:8082)"
	@echo ""
	@echo "  Help:"
	@echo "    help              - Show this help message"
