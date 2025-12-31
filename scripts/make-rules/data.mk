# ==============================================================================
# Makefile helper functions for data download
#

DATA_DIR ?= $(PROJ_ROOT_DIR)
MILVUS_DOCS_URL := https://github.com/milvus-io/milvus-docs/releases/download/v2.4.6-preview/milvus_docs_2.4.x_en.zip
MILVUS_DOCS_ZIP := $(DATA_DIR)/milvus_docs_2.4.x_en.zip
MILVUS_DOCS_DIR := $(DATA_DIR)/milvus_docs

.PHONY: data.download.milvus
data.download.milvus: ## Download Milvus documentation.
	@echo "===========> Downloading Milvus documentation"
	@wget -O $(MILVUS_DOCS_ZIP) $(MILVUS_DOCS_URL)
	@echo "===========> Download completed: $(MILVUS_DOCS_ZIP)"

.PHONY: data.extract.milvus
data.extract.milvus: ## Extract Milvus documentation.
	@echo "===========> Extracting Milvus documentation"
	@unzip -o $(MILVUS_DOCS_ZIP) -d $(DATA_DIR)
	@echo "===========> Extraction completed: $(MILVUS_DOCS_DIR)"

.PHONY: data.setup.milvus
data.setup.milvus: data.download.milvus data.extract.milvus ## Download and extract Milvus documentation.
	@echo "===========> Milvus documentation setup completed"

.PHONY: data.clean.milvus
data.clean.milvus: ## Clean Milvus documentation.
	@echo "===========> Cleaning Milvus documentation"
	@rm -rf $(MILVUS_DOCS_ZIP) $(MILVUS_DOCS_DIR)
	@echo "===========> Clean completed"
