# ==============================================================================
# Makefile helper functions for golang
#

GO := go
GO_BUILD_FLAGS += -trimpath
GO_BUILD_LDFLAGS += -s -w -X 'github.com/kart-io/version.gitVersion=$(VERSION)' \
	-X 'github.com/kart-io/version.gitCommit=$(GIT_COMMIT)' \
	-X 'github.com/kart-io/version.gitBranch=$(GIT_BRANCH)' \
	-X 'github.com/kart-io/version.buildDate=$(BUILD_DATE)'

.PHONY: go.build.%
go.build.%:
	$(eval BINARY := $(word 2,$(subst ., ,$*)))
	$(eval PLAT := $(word 1,$(subst ., ,$*)))
	$(eval OS := $(word 1,$(subst _, ,$(PLAT))))
	$(eval ARCH := $(word 2,$(subst _, ,$(PLAT))))
	@echo "===========> Building binary $(BINARY) $(VERSION) for $(OS) $(ARCH)"
	@mkdir -p $(LOCALBIN)
	@CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) $(GO) build $(GO_BUILD_FLAGS) -ldflags "$(GO_BUILD_LDFLAGS)" -o $(LOCALBIN)/$(BINARY) $(PROJ_ROOT_DIR)/cmd/$(BINARY)

.PHONY: go.build
go.build: $(addprefix go.build., $(addprefix $(PLATFORM)., $(BINS)))

.PHONY: go.clean
go.clean:
	@echo "===========> Cleaning all build output"
	@-rm -vrf $(OUTPUT_DIR)

.PHONY: go.test
go.test:
	@echo "===========> Run unit test"
	@$(GO) test -cover -coverprofile=coverage.out -timeout=10m -shuffle=on -race ./...

.PHONY: go.test.cover
go.test.cover: go.test
	@$(GO) tool cover -func=coverage.out | awk -v target=$(COVERAGE) -f $(SCRIPTS_DIR)/lib/coverage.awk

.PHONY: go.fmt
go.fmt: tools.verify.gofumpt tools.verify.gci tools.verify.goimports
	@echo "===========> Formating codes"
	@gofumpt -w $(shell find . -type f -name '*.go' -not -path "*/vendor/*" -not -name '*.pb.go')
	@goimports -w -local $(GO_MOD_NAME) $(shell find . -type f -name '*.go' -not -path "*/vendor/*" -not -name '*.pb.go')
	@gci write -s standard -s default -s 'Prefix($(GO_MOD_DOMAIN))' --skip-generated $(shell find . -type f -name '*.go' -not -path "*/vendor/*" -not -name '*.pb.go') > /dev/null

.PHONY: go.lint
go.lint: tools.verify.golangci-lint
	@echo "===========> Run golangci to lint source codes"
	@golangci-lint run -c $(PROJ_ROOT_DIR)/.golangci.yaml ./...
