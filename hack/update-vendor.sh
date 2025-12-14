#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"

STAGING_DIR="${PROJ_ROOT_DIR}/staging/src"

sentinel::log::info "===> Updating replace directives..."

MOD_DIRS=$(find "${STAGING_DIR}" -name go.mod -exec dirname {} \;)

# 1. 先删除所有现有的 staging 模块 replace 指令
for mod in ${MOD_DIRS}; do
  module=$(grep '^module ' "${mod}/go.mod" | awk '{print $2}')
  # 删除现有的 replace 指令（如果存在）
  go mod edit -dropreplace="${module}" "${PROJ_ROOT_DIR}/go.mod" 2>/dev/null || true
done

# 2. 添加新的 replace 指令 (Using common utility for path calculation)
for mod in ${MOD_DIRS}; do
  module=$(grep '^module ' "${mod}/go.mod" | awk '{print $2}')
  relpath=$(sentinel::util::get_relpath "$mod" "$PROJ_ROOT_DIR")
  # 使用 go mod edit 添加 replace，避免重复
  go mod edit -replace="${module}=./${relpath}" "${PROJ_ROOT_DIR}/go.mod"
done

sentinel::log::info "===> Running go mod tidy..."
(cd "${PROJ_ROOT_DIR}" && go mod tidy)

sentinel::log::info "===> Running go mod vendor..."
(cd "${PROJ_ROOT_DIR}" && go mod vendor)

sentinel::log::info "Update vendor done."
