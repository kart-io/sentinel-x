#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
STAGING_DIR="${ROOT}/staging/src"

echo "===> Updating replace directives..."

MOD_DIRS=$(find "${STAGING_DIR}" -name go.mod -exec dirname {} \;)

# 1. 先删除所有现有的 staging 模块 replace 指令
for mod in ${MOD_DIRS}; do
  module=$(grep '^module ' "${mod}/go.mod" | awk '{print $2}')
  # 删除现有的 replace 指令（如果存在）
  go mod edit -dropreplace="${module}" "${ROOT}/go.mod" 2>/dev/null || true
done

# 2. 兼容 Linux/macOS 获取相对路径函数
get_relpath() {
  local target="$1"
  local base="$2"

  # 优先使用 realpath 或 grealpath 的 --relative-to
  if command -v realpath >/dev/null 2>&1 && realpath --help 2>&1 | grep -q -- '--relative-to'; then
    realpath --relative-to="$base" "$target"
  elif command -v grealpath >/dev/null 2>&1; then
    grealpath --relative-to="$base" "$target"
  else
    # fallback：去掉 base 路径前缀，简单字符串截取
    echo "${target#$base/}"
  fi
}

# 3. 添加新的 replace 指令
for mod in ${MOD_DIRS}; do
  module=$(grep '^module ' "${mod}/go.mod" | awk '{print $2}')
  relpath=$(get_relpath "$mod" "$ROOT")
  # 使用 go mod edit 添加 replace，避免重复
  go mod edit -replace="${module}=./${relpath}" "${ROOT}/go.mod"
done

echo "===> Running go mod tidy..."
(cd "${ROOT}" && go mod tidy)

echo "===> Running go mod vendor..."
(cd "${ROOT}" && go mod vendor)

echo "Update vendor done."
