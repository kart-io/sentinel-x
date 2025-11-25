#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
STAGING_SRC="${ROOT_DIR}/staging/src/github.com/kart-io"

# 模块名和对应仓库地址，扩展时请在这里添加
MODULES=("goagent")
REPOS=("https://github.com/kart-io/goagent.git")

echo "==> Syncing staging modules..."

mkdir -p "${STAGING_SRC}"

for i in "${!MODULES[@]}"; do
  module="${MODULES[i]}"
  repo="${REPOS[i]}"
  MODULE_DIR="${STAGING_SRC}/${module}"

  if [ -d "${MODULE_DIR}/.git" ]; then
    echo "Updating existing repo ${module}..."
    git -C "${MODULE_DIR}" pull --ff-only
  else
    echo "Cloning repo ${module}..."
    git clone "${repo}" "${MODULE_DIR}"
  fi
done

echo "Staging modules sync complete."
