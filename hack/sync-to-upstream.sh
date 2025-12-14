#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

# sync-to-upstream.sh
# This script syncs the local staging directory content to the remote upstream repository.
# Usage: ./hack/sync-to-upstream.sh <local_path> <remote_repo> [branch]

PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"

if [ "$#" -lt 2 ]; then
    onex::log::error "Usage: $0 <local_path> <remote_repo> [branch]"
    exit 1
fi

STAGING_PATH="$1"
REMOTE_REPO="$2"
BRANCH="${3:-master}"
TMP_DIR=$(mktemp -d)

SOURCE_DIR="${PROJ_ROOT_DIR}/${STAGING_PATH}"

if [ ! -d "${SOURCE_DIR}" ]; then
    onex::log::error "Error: Source directory ${SOURCE_DIR} does not exist."
    exit 1
fi

onex::log::info "===> Publishing ${STAGING_PATH} to ${REMOTE_REPO} (${BRANCH})..."

# 1. Clone the remote repository to a temporary location
onex::log::info "-> Cloning remote repository..."
if ! git -c http.proxy= -c https.proxy= clone --depth 1 --branch "${BRANCH}" "${REMOTE_REPO}" "${TMP_DIR}"; then
     onex::log::error "Error: Failed to clone ${REMOTE_REPO}."
     rm -rf "${TMP_DIR}"
     exit 1
fi

# 2. Sync content
onex::log::info "-> Syncing content..."
# Remove all files in tmp dir except .git
find "${TMP_DIR}" -mindepth 1 -maxdepth 1 -not -name '.git' -exec rm -rf {} +

# Copy content from staging to tmp dir
# Exclude .git if it exists in source (it shouldn't in K8s style, but just in case)
cp -R "${SOURCE_DIR}/." "${TMP_DIR}/"

# 3. Check for changes and commit
cd "${TMP_DIR}"

if [[ -z $(git status --porcelain) ]]; then
    onex::log::info "-> No changes detected. Remote is up to date."
else
    git add -A

    # Create a commit message
    COMMIT_MSG="Sync from sentinel-x staging

Synced from sentinel-x commit: $(git -C "${PROJ_ROOT_DIR}" rev-parse --short HEAD)
Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"

    git commit -m "${COMMIT_MSG}"

    onex::log::info "-> Pushing changes to remote..."
    git push origin "${BRANCH}"

    onex::log::info "===> Successfully published to ${REMOTE_REPO}"
fi

# Cleanup
rm -rf "${TMP_DIR}"
onex::log::info "Done."
