#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

#
function onex::util::sourced_variable {
  # Call this function to tell shellcheck that a variable is supposed to
  # be used from other calling context. This helps quiet an "unused
  # variable" warning from shellcheck and also document your code.
  true
}

function onex::util::sortable_date() {
  date "+%Y%m%d-%H%M%S"
}

# returns 0 if target is in the given items, 1 otherwise.
function onex::util::array_contains() {
  local search="$1"
  local element
  shift
  for element; do
    if [[ "${element}" == "${search}" ]]; then
      return 0
     fi
  done
  return 1
}

function onex::util::wait_for_url() {
  local url=$1
  local prefix=${2:-}
  local wait=${3:-1}
  local times=${4:-30}
  local maxtime=${5:-1}

  command -v curl >/dev/null || {
    onex::log::usage "curl must be installed"
    exit 1
  }

  local i
  for i in $(seq 1 "${times}"); do
    local out
    if out=$(curl --max-time "${maxtime}" -gkfs "${url}" 2>/dev/null); then
      onex::log::status "On try ${i}, ${prefix}: ${out}"
      return 0
    fi
    sleep "${wait}"
  done
  onex::log::error "Timed out waiting for ${prefix} to answer at ${url}; tried ${times} waiting ${wait} between each"
  return 1
}

function onex::util::wait_for_url_with_bearer_token() {
  local url=$1
  local token=$2
  local prefix=${3:-}
  local wait=${4:-1}
  local times=${5:-30}
  local maxtime=${6:-1}

  onex::util::wait_for_url "${url}" "${prefix}" "${wait}" "${times}" "${maxtime}" -H "Authorization: Bearer ${token}"
}

# returns 0 if the shell command get output, 1 otherwise.
function onex::util::wait_for_success(){
  local wait_time="$1"
  local sleep_time="$2"
  local cmd="$3"
  while [ "$wait_time" -gt 0 ]; do
    if eval "$cmd"; then
      return 0
    else
      sleep "$sleep_time"
      wait_time=$((wait_time-sleep_time))
    fi
  done
  return 1
}

# See: http://stackoverflow.com/questions/3338030/multiple-bash-traps-for-the-same-signal
function onex::util::trap_add() {
  local trap_add_cmd
  trap_add_cmd=$1
  shift

  for trap_add_name in "$@"; do
    local existing_cmd
    local new_cmd

    # Grab the currently defined trap commands for this trap
    existing_cmd=$(trap -p "${trap_add_name}" |  awk -F"'" '{print $2}')

    if [[ -z "${existing_cmd}" ]]; then
      new_cmd="${trap_add_cmd}"
    else
      new_cmd="${trap_add_cmd};${existing_cmd}"
    fi

    # Assign the test. Disable the shellcheck warning telling that trap
    # commands should be single quoted to avoid evaluating them at this
    # point instead evaluating them at run time. The logic of adding new
    # commands to a single trap requires them to be evaluated right away.
    # shellcheck disable=SC2064
    trap "${new_cmd}" "${trap_add_name}"
  done
}

# Opposite of onex::util::ensure-temp-dir()
function onex::util::cleanup-temp-dir() {
  rm -rf "${ONEX_TEMP}"
}

# ONEX_TEMP
function onex::util::ensure-temp-dir() {
  if [[ -z ${ONEX_TEMP-} ]]; then
    ONEX_TEMP=$(mktemp -d 2>/dev/null || mktemp -d -t onex.XXXXXX)
    onex::util::trap_add onex::util::cleanup-temp-dir EXIT
  fi
}

function onex::util::ensure-bash-version() {
  if [ "${BASH_VERSINFO[0]}" -lt 3 ]; then
    onex::log::error "Bash version must be 3 or higher"
    exit 1
  fi
}

function onex::util::ensure-gnu-date() {
  if ! command -v date >/dev/null; then
    onex::log::error "date command not found"
    exit 1
  fi
  if date --version 2>&1 | grep -q "GNU coreutils"; then
    DATE=date
  elif command -v gdate >/dev/null; then
    DATE=gdate
  else
    onex::log::error "GNU date not found. Please install coreutils (brew install coreutils on Mac)."
    exit 1
  fi
}

function onex::util::host_os() {
  local host_os
  case "$(uname -s)" in
    Darwin)
      host_os=darwin
      ;;
    Linux)
      host_os=linux
      ;;
    *)
      onex::log::error "Unsupported host OS.  Must be Linux or Mac OS X."
      exit 1
      ;;
  esac
  echo "${host_os}"
}

function onex::util::host_arch() {
  local host_arch
  case "$(uname -m)" in
    x86_64*)
      host_arch=amd64
      ;;
    i?86_64*)
      host_arch=amd64
      ;;
    amd64*)
      host_arch=amd64
      ;;
    aarch64*)
      host_arch=arm64
      ;;
    arm64*)
      host_arch=arm64
      ;;
    arm*)
      host_arch=arm
      ;;
    i?86*)
      host_arch=x86
      ;;
    s390x*)
      host_arch=s390x
      ;;
    ppc64le*)
      host_arch=ppc64le
      ;;
    *)
      onex::log::error "Unsupported host arch. Must be x86_64, 386, arm, arm64, s390x or ppc64le."
      exit 1
      ;;
  esac
  echo "${host_arch}"
}

# this info to figure out where the final binaries are placed.
function onex::util::host_platform() {
  echo "$(onex::util::host_os)/$(onex::util::host_arch)"
}

# $PROJ_ROOT_DIR must be set
function onex::util::find-binary-for-platform() {
  local -r lookfor="$1"
  local -r platform="$2"
  local locations=(
    "${PROJ_ROOT_DIR}/_output/bin/${lookfor}"
    "${PROJ_ROOT_DIR}/_output/dockerized/bin/${platform}/${lookfor}"
    "${PROJ_ROOT_DIR}/_output/local/bin/${platform}/${lookfor}"
    "${PROJ_ROOT_DIR}/platforms/${platform}/${lookfor}"
  )

  # if we're looking for the host platform, add local non-platform-qualified search paths
  if [[ "${platform}" = "$(onex::util::host_platform)" ]]; then
    locations+=(
      "${PROJ_ROOT_DIR}/_output/local/go/bin/${lookfor}"
      "${PROJ_ROOT_DIR}/_output/dockerized/go/bin/${lookfor}"
    );
  fi

  # looks for $1 in the $PATH
  if which "${lookfor}" >/dev/null; then
    local -r local_bin="$(which "${lookfor}")"
    locations+=( "${local_bin}"  );
  fi

  # List most recently-updated location.
  local -r bin=$( (ls -t "${locations[@]}" 2>/dev/null || true) | head -1 )

  if [[ -z "${bin}" ]]; then
    onex::log::error "Failed to find binary ${lookfor} for platform ${platform}"
    return 1
  fi

  echo -n "${bin}"
}

# $PROJ_ROOT_DIR must be set
function onex::util::find-binary() {
  onex::util::find-binary-for-platform "$1" "$(onex::util::host_platform)"
}

# UPDATEME: When add new api group.
function onex::util::group-version-to-pkg-path() {
  local group_version="$1"

  # Special cases first.
  # TODO(lavalamp): Simplify this by moving pkg/api/v1 and splitting pkg/api,
  # moving the results to pkg/apis/api.
  case "${group_version}" in
    # both group and version are "", this occurs when we generate deep copies for internal objects of the legacy v1 API.
    __internal)
      echo "pkg/apis/core"
      ;;
    #core/v1)
      #echo "${PROJ_ROOT_DIR}/pkg/apis/core/v1"
      #;;
    apps/v1beta1)
      echo "${PROJ_ROOT_DIR}/pkg/apis/apps/v1beta1"
      ;;
    batch/v1beta1)
      echo "${PROJ_ROOT_DIR}/pkg/apis/batch/v1beta1"
      ;;
    *)
      echo "pkg/apis/${group_version%__internal}"
      ;;
  esac
}

# special case for v1: v1 -> v1
function onex::util::gv-to-swagger-name() {
  local group_version="$1"
  case "${group_version}" in
    v1)
      echo "v1"
      ;;
    *)
      echo "${group_version%/*}_${group_version#*/}"
      ;;
  esac
}

# repo, e.g. "upstream" or "origin".
function onex::util::git_upstream_remote_name() {
  git remote -v | grep fetch |\
    grep -E 'github.com[/:]onexstack/onex|onexstack.io/onex' |\
    head -n 1 | awk '{print $1}'
}

# the user can commit changes in a second terminal. This script will wait.
function onex::util::ensure_clean_working_dir() {
  while ! git diff HEAD --exit-code &>/dev/null; do
    echo -e "\nUnexpected dirty working directory:\n"
    if tty -s; then
        git status -s
    else
        git diff -a # be more verbose in log files without tty
        exit 1
    fi | sed 's/^/  /'
    echo -e "\nCommit your changes in another terminal and then continue here by pressing enter."
    read -r
  done 1>&2
}

# current ref from the remote upstream branch
function onex::util::base_ref() {
  local -r git_branch=$1

  if [[ -n ${PULL_BASE_SHA:-} ]]; then
    echo "${PULL_BASE_SHA}"
    return
  fi

  full_branch="$(onex::util::git_upstream_remote_name)/${git_branch}"

  # make sure the branch is valid, otherwise the check will pass erroneously.
  if ! git describe "${full_branch}" >/dev/null; then
    # abort!
    exit 1
  fi

  echo "${full_branch}"
}

# 0 (true) if there are changes detected.
function onex::util::has_changes() {
  local -r git_branch=$1
  local -r pattern=$2
  local -r not_pattern=${3:-totallyimpossiblepattern}

  local base_ref
  base_ref=$(onex::util::base_ref "${git_branch}")
  echo "Checking for '${pattern}' changes against '${base_ref}'"

  # notice this uses ... to find the first shared ancestor
  if git diff --name-only "${base_ref}...HEAD" | grep -v -E "${not_pattern}" | grep "${pattern}" > /dev/null; then
    return 0
  fi
  # also check for pending changes
  if git status --porcelain | grep -v -E "${not_pattern}" | grep "${pattern}" > /dev/null; then
    echo "Detected '${pattern}' uncommitted changes."
    return 0
  fi
  echo "No '${pattern}' changes detected."
  return 1
}

function onex::util::download_file() {
  local -r url=$1
  local -r destination_file=$2

  rm "${destination_file}" 2&> /dev/null || true

  for i in $(seq 5)
  do
    if ! curl -fsSL --retry 3 --keepalive-time 2 "${url}" -o "${destination_file}"; then
      echo "Downloading ${url} failed. $((5-i)) retries left."
      sleep 1
    else
      echo "Downloading ${url} succeed"
      return 0
    fi
  done
  return 1
}

# OPENSSL_BIN: The path to the openssl binary to use
function onex::util::test_openssl_installed {
    if ! openssl version >& /dev/null; then
      echo "Failed to run openssl. Please ensure openssl is installed"
      exit 1
    fi

    OPENSSL_BIN=$(command -v openssl)
}

# Some useful colors.
if [[ -z "${color_start-}" ]]; then
  declare -r color_start="\033["
  declare -r color_red="${color_start}0;31m"
  declare -r color_yellow="${color_start}0;33m"
  declare -r color_green="${color_start}0;32m"
  declare -r color_blue="${color_start}1;34m"
  declare -r color_cyan="${color_start}1;36m"
  declare -r color_norm="${color_start}0m"

  onex::util::sourced_variable "${color_start}"
  onex::util::sourced_variable "${color_red}"
  onex::util::sourced_variable "${color_yellow}"
  onex::util::sourced_variable "${color_green}"
  onex::util::sourced_variable "${color_blue}"
  onex::util::sourced_variable "${color_cyan}"
  onex::util::sourced_variable "${color_norm}"
fi

# 2. 兼容 Linux/macOS 获取相对路径函数
function onex::util::get_relpath() {
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
