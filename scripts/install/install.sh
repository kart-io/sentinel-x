#!/usr/bin/env bash

SERVER_ROOT=$(dirname "${BASH_SOURCE[0]}")/../..
source "${SERVER_ROOT}/scripts/install/common.sh"

# Import other scripts
source "${SERVER_ROOT}/scripts/install/protobuf.sh"
source "${SERVER_ROOT}/scripts/install/tools.sh"

function install_all() {
  log::info "Starting full installation..."
  
  install_protobuf
  install_tools
  
  log::info "All tools installed successfully!"
}

# Check if arguments are provided
if [[ "$#" -gt 0 ]]; then
  tool_name=$1
  # Replace hyphens/dots with underscores
  func_name="install_${tool_name//[-.]/_}"
  
  if declare -f "$func_name" > /dev/null; then
    log::info "Installing $tool_name..."
    $func_name
    log::info "$tool_name installed successfully"
  else
    log::error "Unknown tool: $tool_name"
    log::info "Available single install steps via make tools.install.<toolname>:"
    log::info "  - golangci-lint, gofumpt, gci, gotests, mockgen, wire, grpcurl"
    log::info "  - buf, protoc-gen-go, protoc-gen-go-grpc, protoc-gen-validate, protoc-gen-openapiv2"
    exit 1
  fi
else
  # No arguments, install everything
  install_all
fi
