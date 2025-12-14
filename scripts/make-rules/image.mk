# ==============================================================================
# Docker Image Makefile
# ==============================================================================

DOCKER := docker
DOCKER_SUPPORTED_API_VERSION ?= 1.32

EXTRA_ARGS ?= --no-cache
_DOCKER_BUILD_EXTRA_ARGS :=

# Track code version with Docker Label.
DOCKER_LABELS ?= git-describe="$(shell date -u +v%Y%m%d)-$(shell git describe --tags --always --dirty)"

ifdef HTTP_PROXY
_DOCKER_BUILD_EXTRA_ARGS += --build-arg HTTP_PROXY=${HTTP_PROXY}
endif

ifneq ($(EXTRA_ARGS), )
_DOCKER_BUILD_EXTRA_ARGS += $(EXTRA_ARGS)
endif

# Determine image files by looking into cmd/*
CMD_DIRS ?= $(wildcard ${PROJ_ROOT_DIR}/cmd/*)

# Filter out directories without Go files
IMAGES ?= $(filter-out tools, $(foreach dir, $(CMD_DIRS), $(notdir $(if $(wildcard $(dir)/*.go), $(dir),))))
ifeq (${IMAGES},)
  $(error Could not determine IMAGES, set PROJ_ROOT_DIR or run in source dir)
endif

.PHONY: image.verify
image.verify: ## Verify docker version.
	$(eval API_VERSION := $(shell $(DOCKER) version | grep -E 'API version: {1,6}[0-9]' | head -n1 | awk '{print $$3} END { if (NR==0) print 0}' ))
	$(eval PASS := $(shell echo "$(API_VERSION) > $(DOCKER_SUPPORTED_API_VERSION)" | bc))
	@if [ $(PASS) -ne 1 ]; then \
		$(DOCKER) -v ;\
		echo "Unsupported docker version. Docker API version should be greater than $(DOCKER_SUPPORTED_API_VERSION)"; \
		exit 1; \
	fi

.PHONY: image.daemon.verify
image.daemon.verify: ## Verify docker daemon version.
	$(eval PASS := $(shell $(DOCKER) version | grep -q -E 'Experimental: {1,5}true' && echo 1 || echo 0))
	@if [ $(PASS) -ne 1 ]; then \
		echo "Experimental features of Docker daemon is not enabled. Please add \"experimental\": true in '/etc/docker/daemon.json' and then restart Docker daemon."; \
		exit 1; \
	fi

.PHONY: image.dockerfile
image.dockerfile: $(addprefix image.dockerfile., $(IMAGES)) ## Generate all dockerfiles.

.PHONY: image.dockerfile.%
image.dockerfile.%: ## Generate specified dockerfiles.
	$(eval IMAGE := $(lastword $(subst ., ,$*)))
	@$(SCRIPTS_DIR)/gen-dockerfile.sh $(GENERATED_DOCKERFILE_DIR) $(IMAGE)
ifeq ($(V),1)
	echo "DBG: Generating Dockerfile at $(GENERATED_DOCKERFILE_DIR)/$(IMAGE)"
endif

.PHONY: image.build
image.build: image.verify $(addprefix image.build., $(addprefix $(IMAGE_PLAT)., $(IMAGES))) ## Build all docker images.

.PHONY: image.build.multiarch
image.build.multiarch: image.verify $(foreach p,$(PLATFORMS),$(addprefix image.build., $(addprefix $(p)., $(IMAGES)))) ## Build all docker images with all supported arch.

.PHONY: image.build.%
image.build.%: image.dockerfile.% ## Build specified docker image.
	$(eval IMAGE := $(word 2,$(subst ., ,$*)))
	$(eval PLAT := $(word 1,$(subst ., ,$*)))
	$(eval IMAGE_PLAT := $(subst _,/,$(PLAT)))
	$(eval OS := $(word 1,$(subst _, ,$(PLAT))))
	$(eval ARCH := $(word 2,$(subst _, ,$(PLAT))))
	$(eval DOCKERFILE := Dockerfile)
	$(eval DST_DIR := $(TMP_DIR)/$(IMAGE))
	$(eval IMAGE_TAG := $(subst +,-,$(VERSION)))
	@echo "===========> Building docker image $(IMAGE) $(IMAGE_TAG) for $(IMAGE_PLAT)"
	@mkdir -p $(TMP_DIR)/$(IMAGE)
	@rsync -a --exclude='_output' --exclude='.git' --exclude='.idea' --exclude='.vscode' $(PROJ_ROOT_DIR)/ $(TMP_DIR)/$(IMAGE)/
	$(eval BUILD_SUFFIX := $(_DOCKER_BUILD_EXTRA_ARGS) --pull \
		-f $(GENERATED_DOCKERFILE_DIR)/$(IMAGE)/$(DOCKERFILE) \
		--build-arg OS=$(OS) \
		--build-arg ARCH=$(ARCH) \
		--build-arg goproxy=$(shell go env GOPROXY) \
		--label $(DOCKER_LABELS) \
		-t $(REGISTRY_PREFIX)/$(IMAGE)-$(ARCH):$(IMAGE_TAG) \
		$(TMP_DIR)/$(IMAGE))
	@if [ $(shell $(GO) env GOARCH) != $(ARCH) ] ; then \
		$(DOCKER) build --platform $(IMAGE_PLAT) $(BUILD_SUFFIX) ; \
	else \
		$(DOCKER) build $(BUILD_SUFFIX) ; \
	fi
	@-rm -rf $(TMP_DIR)/$(IMAGE)

.PHONY: image.push
image.push: image.verify $(addprefix image.push., $(addprefix $(IMAGE_PLAT)., $(IMAGES))) ## Build and push all docker images to docker registry.

.PHONY: image.push.%
image.push.%: image.build.% ## Build and push specified docker image.
	@echo "===========> Pushing image $(IMAGE) $(IMAGE_TAG) to $(REGISTRY_PREFIX)"
	$(DOCKER) push $(REGISTRY_PREFIX)/$(IMAGE)-$(ARCH):$(IMAGE_TAG)
