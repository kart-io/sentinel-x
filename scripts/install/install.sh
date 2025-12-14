#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

# Use PROJ_ROOT_DIR from init.sh if available, otherwise calculate it
if [ -z "${PROJ_ROOT_DIR}" ]; then
    PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
    source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"
fi

# Import other scripts
# Use absolute paths calculated from PROJ_ROOT_DIR or relative to this script
source "${PROJ_ROOT_DIR}/scripts/install/protobuf.sh"
source "${PROJ_ROOT_DIR}/scripts/install/tools.sh"

function install_all() {
  sentinel::log::info "Starting full installation..."

  install_protobuf
  install_tools

  sentinel::log::info "All tools installed successfully!"
}

# Check if arguments are provided
if [[ "$#" -gt 0 ]]; then
  tool_name=$1

  if [[ "$tool_name" == "-h" ]] || [[ "$tool_name" == "--help" ]]; then
      sentinel::log::info "Usage: $0 [tool_name]"
      sentinel::log::info "Install a specific tool or all tools if no argument is provided."
      sentinel::log::info "Available tools:"
      sentinel::log::info "  - golangci-lint, gofumpt, gci, gotests, mockgen, wire, grpcurl"
      sentinel::log::info "  - buf, protoc-gen-go, protoc-gen-go-grpc, protoc-gen-validate, protoc-gen-openapiv2"
      exit 0
  fi

  # Replace hyphens/dots with underscores
  func_name="install_${tool_name//[-.]/_}"

  if declare -f "$func_name" > /dev/null; then
    sentinel::log::info "Installing $tool_name..."
    $func_name
    sentinel::log::info "$tool_name installed successfully"
  else
    sentinel::log::error "Unknown tool: $tool_name"
    sentinel::log::info "Available single install steps via make tools.install.<toolname>:"
    sentinel::log::info "  - golangci-lint, gofumpt, gci, gotests, mockgen, wire, grpcurl"
    sentinel::log::info "  - buf, protoc-gen-go, protoc-gen-go-grpc, protoc-gen-validate, protoc-gen-openapiv2"
    exit 1
  fi
else
  # No arguments, install everything
  install_all
fi
