#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

# Wrapper to run make image targets
# Usage: ./scripts/image.sh [CMD]
# CMD: build, push, etc. Default: build

PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"

command=${1:-build}
shift

if [ "$command" == "build" ]; then
    sentinel::log::info "Building image..."
    make -C "${PROJ_ROOT_DIR}" image.build "$@"
elif [ "$command" == "push" ]; then
    sentinel::log::info "Pushing image..."
    make -C "${PROJ_ROOT_DIR}" image.push "$@"
else
    sentinel::log::error "Usage: $0 [build|push]"
    exit 1
fi
