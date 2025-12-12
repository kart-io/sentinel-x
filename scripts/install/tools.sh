#!/usr/bin/env bash

SERVER_ROOT=$(dirname "${BASH_SOURCE[0]}")/../..
source "${SERVER_ROOT}/scripts/install/common.sh"

function install_golangci_lint() {
  if ! command -v golangci-lint &> /dev/null; then
      log::info "Installing golangci-lint..."
      go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  else
      log::info "golangci-lint is already installed"
  fi
}

function install_gofumpt() {
  if ! command -v gofumpt &> /dev/null; then
      log::info "Installing gofumpt..."
      go install mvdan.cc/gofumpt@latest
  fi
}

function install_gci() {
  if ! command -v gci &> /dev/null; then
      log::info "Installing gci..."
      go install github.com/daixiang0/gci@latest
  fi
}

function install_gotests() {
  if ! command -v gotests &> /dev/null; then
      log::info "Installing gotests..."
      go install github.com/cweill/gotests/gotests@latest
  fi
}

function install_mockgen() {
  if ! command -v mockgen &> /dev/null; then
      log::info "Installing mockgen..."
      go install go.uber.org/mock/mockgen@latest
  fi
}

function install_wire() {
  if ! command -v wire &> /dev/null; then
      log::info "Installing wire..."
      go install github.com/google/wire/cmd/wire@latest
  fi
}

function install_grpcurl() {
  if ! command -v grpcurl &> /dev/null; then
      log::info "Installing grpcurl..."
      go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
  fi
}

function install_tools() {
  log::info "Installing utility tools..."
  install_golangci_lint
  install_gofumpt
  install_gci
  install_gotests
  install_mockgen
  install_wire
  install_grpcurl
  log::info "Utility tools installed successfully"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  install_tools
fi
