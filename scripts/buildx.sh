#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

# Wrapper to run make image multiarch targets
# Usage: ./scripts/buildx.sh [CMD]

PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"

command=${1:-build}
shift

if [ "$command" == "build" ]; then
    sentinel::log::info "Building multiarch image..."
    make -C "${PROJ_ROOT_DIR}" image.build.multiarch "$@"
elif [ "$command" == "push" ]; then
    sentinel::log::info "Pushing multiarch image..."
    make -C "${PROJ_ROOT_DIR}" image.push.multiarch "$@"
else
    sentinel::log::error "Usage: $0 [build|push]"
    exit 1
fi
