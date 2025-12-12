#!/usr/bin/env bash

# Common variables
OS=$(go env GOOS)
ARCH=$(go env GOARCH)
GOPATH=$(go env GOPATH)
GOBIN=$(go env GOBIN)
[ -z "$GOBIN" ] && GOBIN=$GOPATH/bin

function log::info() {
  printf "\033[32m[INFO] %s\033[0m\n" "$*"
}

function log::error() {
  printf "\033[31m[ERROR] %s\033[0m\n" "$*"
}
