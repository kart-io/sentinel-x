#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

# Use PROJ_ROOT_DIR from init.sh if available, otherwise calculate it
if [ -z "${PROJ_ROOT_DIR}" ]; then
    PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
    source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"
fi

function install_buf() {
  if ! command -v buf &> /dev/null; then
      onex::log::info "Installing buf..."
      go install github.com/bufbuild/buf/cmd/buf@latest
  else
      onex::log::info "buf is already installed"
  fi
}

function install_protoc_gen_go() {
  if ! command -v protoc-gen-go &> /dev/null; then
      onex::log::info "Installing protoc-gen-go..."
      go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  fi
}

function install_protoc_gen_go_grpc() {
  if ! command -v protoc-gen-go-grpc &> /dev/null; then
      onex::log::info "Installing protoc-gen-go-grpc..."
      go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  fi
}

function install_protoc_gen_validate() {
  if ! command -v protoc-gen-validate &> /dev/null; then
      onex::log::info "Installing protoc-gen-validate..."
      go install github.com/envoyproxy/protoc-gen-validate@latest
  fi
}

function install_protoc_gen_openapiv2() {
  if ! command -v protoc-gen-openapiv2 &> /dev/null; then
      onex::log::info "Installing protoc-gen-openapiv2..."
      go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
  fi
}

function install_protobuf() {
  onex::log::info "Installing protobuf tools..."
  install_buf
  install_protoc_gen_go
  install_protoc_gen_go_grpc
  install_protoc_gen_validate
  install_protoc_gen_openapiv2
  onex::log::info "Protobuf tools installed successfully"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    onex::log::info "Usage: $0"
    onex::log::info "Install Protocol Buffers tools (buf, protoc-gen-go, etc.)"
    exit 0
  fi
  install_protobuf
fi
