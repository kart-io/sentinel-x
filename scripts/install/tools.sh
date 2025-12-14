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

function install_tools() {
  sentinel::log::info "Installing utility tools..."
  install_golangci_lint
  install_gofumpt
  install_gci
  install_gotests
  install_mockgen
  install_wire
  install_grpcurl
  sentinel::log::info "Utility tools installed successfully"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    sentinel::log::info "Usage: $0"
    sentinel::log::info "Install development tools (golangci-lint, gofumpt, gci, etc.)"
    exit 0
  fi
  install_tools
fi
