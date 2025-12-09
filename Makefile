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
