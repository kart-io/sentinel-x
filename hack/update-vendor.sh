#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
STAGING_DIR="${ROOT}/staging/src"

echo "===> Updating replace directives..."

MOD_DIRS=$(find "${STAGING_DIR}" -name go.mod -exec dirname {} \;)

# 兼容 Linux/macOS 获取相对路径函数
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
    # 注意：base 必须是 target 的前缀路径
    echo "${target#$base/}"
  fi
}

# 生成 replace 语句临时文件
REPLACE_TMP="$(mktemp)"
for mod in ${MOD_DIRS}; do
  module=$(grep '^module ' "${mod}/go.mod" | awk '{print $2}')
  relpath=$(get_relpath "$mod" "$ROOT")
  echo "replace ${module} => ./${relpath}" >> "${REPLACE_TMP}"
done

# 追加 replace 到 go.mod（你也可以选择覆盖或其他方式）
echo >> "${ROOT}/go.mod"
cat "${REPLACE_TMP}" >> "${ROOT}/go.mod"
rm "${REPLACE_TMP}"

echo "===> Running go mod tidy..."
(cd "${ROOT}" && go mod tidy)

echo "===> Running go mod vendor..."
(cd "${ROOT}" && go mod vendor)

echo "Update vendor done."
