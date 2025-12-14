
# Default <prod_image> is debian:bookworm-slim
ARG prod_image=debian:bookworm-slim

# builder stage
FROM golang:1.25 as builder
WORKDIR /workspace

# Run this with docker build --build-arg goproxy=$(go env GOPROXY) to override the goproxy
ARG goproxy=https://proxy.golang.org
ARG OS
ARG ARCH

# Run this with docker build.
ENV GOPROXY=$goproxy

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy staging if exists (for local replacements)
COPY staging/ staging/

# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the sources
COPY pkg/ pkg/
COPY internal/ internal/
# Copy api folder if it exists at root (it does in sentinel-x, inside pkg but to be safe)
# In sentinel-x, api is inside pkg/api, so COPY pkg/ covers it.
# But just in case we have other root directories:
# COPY third_party/ third_party/

COPY cmd/scheduler cmd/scheduler

# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${OS:-linux} GOARCH=${ARCH} go build -a -o scheduler ./cmd/scheduler

# Production image
FROM ${prod_image}
LABEL maintainer="<developer@sentinel-x.io>"

WORKDIR /opt/sentinel-x

# setting timezone otherwise the build will fail
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime &&       echo "Asia/Shanghai" > /etc/timezone

COPY --from=builder /workspace/scheduler /opt/sentinel-x/bin/

# Use uid of nonroot user (65532) because kubernetes expects numeric user when applying pod security policies
USER 65532
ENTRYPOINT ["/opt/sentinel-x/bin/scheduler"]
