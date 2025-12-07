#!/usr/bin/env bash

# sync-from-upstream.sh
# This script pulls the latest code for a module from its remote repository
# into the local staging directory.
# Usage: ./sync-from-upstream.sh <local_path> <remote_repo> [branch]

set -euo pipefail

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <local_path> <remote_repo> [branch]"
    exit 1
fi

SOURCE_MODULE_PATH="$1"
REMOTE_REPO="$2"
BRANCH="${3:-master}"
TMP_DIR=$(mktemp -d)

# Ensure we are at the project root
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TARGET_DIR="${ROOT_DIR}/${SOURCE_MODULE_PATH}"

echo "==> Updating ${SOURCE_MODULE_PATH} from ${REMOTE_REPO} (${BRANCH})..."

# 1. Clone the remote repository to a temporary location
echo "-> Cloning remote repository to temporary directory..."
if ! git -c http.proxy= -c https.proxy= clone --depth 1 --branch "${BRANCH}" "${REMOTE_REPO}" "${TMP_DIR}"; then
    echo "Error: Failed to clone ${REMOTE_REPO}. Please check your network connection or repository access."
    rm -rf "${TMP_DIR}"
    exit 1
fi

# 2. Clear current staging content (preserving .git if it existed, though it shouldn't)
echo "-> Clearing existing content in ${TARGET_DIR}..."
# Create target dir if it doesn't exist
mkdir -p "${TARGET_DIR}"

(
    cd "${TARGET_DIR}" || exit 1
    # Use find to delete everything except .git (if any)
    find . -mindepth 1 -maxdepth 1 -not -name '.git' -exec rm -rf {} +
)

# 3. Copy new content from temporary clone to staging
echo "-> Copying new content into ${TARGET_DIR}..."
# Copy contents of TMP_DIR to TARGET_DIR, excluding .git
# rsync is safer but cp -a is standard. We want to exclude .git from source.
# The clone has .git. We don't want to copy it.
find "${TMP_DIR}" -mindepth 1 -maxdepth 1 -not -name '.git' -exec cp -R {} "${TARGET_DIR}/" \;

# 4. Run go mod tidy and go mod vendor from the root to ensure consistency
echo "-> Running go mod tidy and go mod vendor..."
(
    cd "${ROOT_DIR}" || exit 1
    go mod tidy
    go mod vendor
)

echo "==> ${SOURCE_MODULE_PATH} updated successfully."

# Cleanup
rm -rf "${TMP_DIR}"
