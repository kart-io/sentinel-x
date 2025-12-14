#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"

if [ $# -ne 2 ];then
    onex::log::error "Usage: gen-dockerfile.sh ${DOCKERFILE_DIR} ${IMAGE_NAME}"
    exit 1
fi

DOCKERFILE_DIR=$1/$2
IMAGE_NAME=$2

# Sentinel-X generic config
ONEX_ALL_IN_ONE_IMAGE_NAME=sentinel-allinone

MOUNT_STAGING=""
COPY_STAGING=""
# Check if staging directory exists for local replacements
if [ -d "${PROJ_ROOT_DIR}/staging" ]; then
    COPY_STAGING="COPY staging/ staging/"
fi

function cat_multistage_dockerfile()
{
	cat << EOF

# Default <prod_image> is BASE_IMAGE
ARG prod_image=BASE_IMAGE

# builder stage
FROM golang:1.25 as builder
WORKDIR /workspace

# Run this with docker build --build-arg goproxy=\$(go env GOPROXY) to override the goproxy
ARG goproxy=https://proxy.golang.org
ARG OS
ARG ARCH

# Run this with docker build.
ENV GOPROXY=\$goproxy

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy staging if exists (for local replacements)
${COPY_STAGING}

# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the sources
COPY pkg/ pkg/
COPY internal/ internal/
# Copy api folder if it exists at root (it does in sentinel-x, inside pkg but to be safe)
# In sentinel-x, api is inside pkg/api, so COPY pkg/ covers it.
# But just in case we have other root directories:
# COPY third_party/ third_party/

COPY cmd/${IMAGE_NAME} cmd/${IMAGE_NAME}

# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=\${OS:-linux} GOARCH=\${ARCH} go build -a -o ${IMAGE_NAME} ./cmd/${IMAGE_NAME}

# Production image
FROM \${prod_image}
LABEL maintainer="<developer@sentinel-x.io>"

WORKDIR /opt/sentinel-x

# setting timezone otherwise the build will fail
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
      echo "Asia/Shanghai" > /etc/timezone

COPY --from=builder /workspace/${IMAGE_NAME} /opt/sentinel-x/bin/

# Use uid of nonroot user (65532) because kubernetes expects numeric user when applying pod security policies
USER 65532
ENTRYPOINT ["/opt/sentinel-x/bin/${IMAGE_NAME}"]
EOF
}

function get_base_image() {
  local image_name=$1
  local base_image="debian:bookworm-slim"

  case "${image_name}" in
    "${ONEX_ALL_IN_ONE_IMAGE_NAME}")
      base_image="debian:bookworm-slim"
      ;;
  esac

  echo "${base_image}"
}

cat_func=cat_multistage_dockerfile
[[ ! -d ${DOCKERFILE_DIR} ]] && mkdir -p ${DOCKERFILE_DIR}

BASE_IMAGE=$(get_base_image ${IMAGE_NAME})

# generate dockerfile
# onex-allinone does not need multiple stages? (Simplified logic here, assuming multi-stage for all)
cat_multistage_dockerfile | \
    sed -e "s/BASE_IMAGE/${BASE_IMAGE}/g" -e "s/IMAGE_NAME/${IMAGE_NAME}/g" > ${DOCKERFILE_DIR}/Dockerfile
