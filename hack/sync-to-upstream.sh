#!/usr/bin/env bash

# sync-to-upstream.sh
# This script syncs the local staging directory content to the remote upstream repository.
# Usage: ./sync-to-upstream.sh <local_path> <remote_repo> [branch]

set -euo pipefail

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <local_path> <remote_repo> [branch]"
    exit 1
fi

STAGING_PATH="$1"
REMOTE_REPO="$2"
BRANCH="${3:-master}"
TMP_DIR=$(mktemp -d)

# Ensure we are at the project root
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SOURCE_DIR="${ROOT_DIR}/${STAGING_PATH}"

if [ ! -d "${SOURCE_DIR}" ]; then
    echo "Error: Source directory ${SOURCE_DIR} does not exist."
    exit 1
fi

echo "===> Publishing ${STAGING_PATH} to ${REMOTE_REPO} (${BRANCH})..."

# 1. Clone the remote repository to a temporary location
echo "-> Cloning remote repository..."
if ! git -c http.proxy= -c https.proxy= clone --depth 1 --branch "${BRANCH}" "${REMOTE_REPO}" "${TMP_DIR}"; then
     echo "Error: Failed to clone ${REMOTE_REPO}."
     rm -rf "${TMP_DIR}"
     exit 1
fi

# 2. Sync content
echo "-> Syncing content..."
# Remove all files in tmp dir except .git
find "${TMP_DIR}" -mindepth 1 -maxdepth 1 -not -name '.git' -exec rm -rf {} +

# Copy content from staging to tmp dir
# Exclude .git if it exists in source (it shouldn't in K8s style, but just in case)
cp -R "${SOURCE_DIR}/." "${TMP_DIR}/"

# 3. Check for changes and commit
cd "${TMP_DIR}"

if [[ -z $(git status --porcelain) ]]; then
    echo "-> No changes detected. Remote is up to date."
else
    git add -A
    
    # Create a commit message
    COMMIT_MSG="Sync from sentinel-x staging

Synced from sentinel-x commit: $(git -C "${ROOT_DIR}" rev-parse --short HEAD)
Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"

    git commit -m "${COMMIT_MSG}"
    
    echo "-> Pushing changes to remote..."
    git push origin "${BRANCH}"
    
    echo "===> Successfully published to ${REMOTE_REPO}"
fi

# Cleanup
rm -rf "${TMP_DIR}"
echo "Done."
