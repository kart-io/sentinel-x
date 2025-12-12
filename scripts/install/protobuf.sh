#!/usr/bin/env bash

SERVER_ROOT=$(dirname "${BASH_SOURCE[0]}")/../..
source "${SERVER_ROOT}/scripts/install/common.sh"

function install_buf() {
  if ! command -v buf &> /dev/null; then
      log::info "Installing buf..."
      go install github.com/bufbuild/buf/cmd/buf@latest
  else
      log::info "buf is already installed"
  fi
}

function install_protoc_gen_go() {
  if ! command -v protoc-gen-go &> /dev/null; then
      log::info "Installing protoc-gen-go..."
      go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  fi
}

function install_protoc_gen_go_grpc() {
  if ! command -v protoc-gen-go-grpc &> /dev/null; then
      log::info "Installing protoc-gen-go-grpc..."
      go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  fi
}

function install_protoc_gen_validate() {
  if ! command -v protoc-gen-validate &> /dev/null; then
      log::info "Installing protoc-gen-validate..."
      go install github.com/envoyproxy/protoc-gen-validate@latest
  fi
}

function install_protoc_gen_openapiv2() {
  if ! command -v protoc-gen-openapiv2 &> /dev/null; then
      log::info "Installing protoc-gen-openapiv2..."
      go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
  fi
}

function install_protobuf() {
  log::info "Installing protobuf tools..."
  install_buf
  install_protoc_gen_go
  install_protoc_gen_go_grpc
  install_protoc_gen_validate
  install_protoc_gen_openapiv2
  log::info "Protobuf tools installed successfully"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  install_protobuf
fi
