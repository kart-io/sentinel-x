#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

# Use PROJ_ROOT_DIR from init.sh if available, otherwise calculate it
if [ -z "${PROJ_ROOT_DIR}" ]; then
    PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
    source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"
fi

function install_golangci_lint() {
  if ! command -v golangci-lint &> /dev/null; then
      sentinel::log::info "Installing golangci-lint..."
      go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  else
      sentinel::log::info "golangci-lint is already installed"
  fi
}

function install_gofumpt() {
  if ! command -v gofumpt &> /dev/null; then
      sentinel::log::info "Installing gofumpt..."
      go install mvdan.cc/gofumpt@latest
  fi
}

function install_gci() {
  if ! command -v gci &> /dev/null; then
      sentinel::log::info "Installing gci..."
      go install github.com/daixiang0/gci@latest
  fi
}

function install_gotests() {
  if ! command -v gotests &> /dev/null; then
      sentinel::log::info "Installing gotests..."
      go install github.com/cweill/gotests/gotests@latest
  fi
}

function install_mockgen() {
  if ! command -v mockgen &> /dev/null; then
      sentinel::log::info "Installing mockgen..."
      go install go.uber.org/mock/mockgen@latest
  fi
}

function install_wire() {
  if ! command -v wire &> /dev/null; then
      sentinel::log::info "Installing wire..."
      go install github.com/google/wire/cmd/wire@latest
  fi
}

function install_grpcurl() {
  if ! command -v grpcurl &> /dev/null; then
      sentinel::log::info "Installing grpcurl..."
      go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
  fi
}

function install_goimports() {
  if ! command -v goimports &> /dev/null; then
      sentinel::log::info "Installing goimports..."
      go install golang.org/x/tools/cmd/goimports@latest
  fi
}

function install_tools() {
  sentinel::log::info "Installing utility tools..."
  install_golangci_lint
  install_gofumpt
  install_gci
  install_gotests
  install_mockgen
  install_wire
  install_grpcurl
  install_goimports
  install_protoc_go_inject_tag
  install_client_gen
  install_lister_gen
  install_informer_gen
  install_deepcopy_gen
  install_controller_gen
  sentinel::log::info "Utility tools installed successfully"
}

function install_protoc_go_inject_tag() {
  if ! command -v protoc-go-inject-tag &> /dev/null; then
      sentinel::log::info "Installing protoc-go-inject-tag..."
      go install github.com/favadi/protoc-go-inject-tag@latest
  fi
}

K8S_CODEGEN_VERSION="v0.32.0"

function install_client_gen() {
  if ! command -v client-gen &> /dev/null; then
      sentinel::log::info "Installing client-gen..."
      go install k8s.io/code-generator/cmd/client-gen@${K8S_CODEGEN_VERSION}
  fi
}

function install_lister_gen() {
  if ! command -v lister-gen &> /dev/null; then
      sentinel::log::info "Installing lister-gen..."
      go install k8s.io/code-generator/cmd/lister-gen@${K8S_CODEGEN_VERSION}
  fi
}

function install_informer_gen() {
  if ! command -v informer-gen &> /dev/null; then
      sentinel::log::info "Installing informer-gen..."
      go install k8s.io/code-generator/cmd/informer-gen@${K8S_CODEGEN_VERSION}
  fi
}

function install_deepcopy_gen() {
  if ! command -v deepcopy-gen &> /dev/null; then
      sentinel::log::info "Installing deepcopy-gen..."
      go install k8s.io/code-generator/cmd/deepcopy-gen@${K8S_CODEGEN_VERSION}
  fi
}

function install_controller_gen() {
  if ! command -v controller-gen &> /dev/null; then
      sentinel::log::info "Installing controller-gen..."
      go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
  fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    sentinel::log::info "Usage: $0"
    sentinel::log::info "Install development tools (golangci-lint, gofumpt, gci, etc.)"
    exit 0
  fi
  install_tools
fi
