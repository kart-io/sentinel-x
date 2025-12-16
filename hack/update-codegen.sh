#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


export GOFLAGS="-mod=mod"

# Project root
PROJ_ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
BOILERPLATE_FILE="${PROJ_ROOT}/hack/boilerplate.go.txt"

# Go module name
GO_MODULE=$(go list -m)

# 默认设置
# 输出生成的 ClientSet, Lister, Informer 的目录 (相对于 pkg)
GENERATED_PKG="${GO_MODULE}/pkg/generated"
# CRD 定义所在的根目录
APIS_PKG="${GO_MODULE}/pkg/apis"

# 检查参数
if [ $# -eq 0 ]; then
  echo "Usage: $0 <group:version>..."
  echo "  <group:version>  e.g., 'sentinel:v1' implies pkg/apis/sentinel/v1"
  echo ""
  echo "Example:"
  echo "  $0 sentinel:v1"
  exit 1
fi

GROUPS_VERSIONS="$@"

# 允许通过 GENERATORS 环境变量指定要生成的类型，默认为 "all"
# 示例: GENERATORS="deepcopy,client" ./hack/update-codegen.sh sentinel:v1
GENERATORS="${GENERATORS:-all}"

function is_generator_enabled() {
  local gen=$1
  if [[ "${GENERATORS}" == "all" ]]; then
    return 0
  fi
  if [[ ",${GENERATORS}," == *",${gen},"* ]]; then
    return 0
  fi
  return 1
}

echo "MODULE: ${GO_MODULE}"
echo "APIS:   ${APIS_PKG}"
echo "GEN:    ${GENERATED_PKG}"
echo "TARGETS: ${GROUPS_VERSIONS}"
echo "GENERATORS: ${GENERATORS}"

# 1. 生成 DeepCopy
# input-dirs 格式: github.com/my/project/pkg/apis/group/v1
# 我们遍历参数，拼接出完整的包路径
INPUT_DIRS=""
for gv in ${GROUPS_VERSIONS}; do
  group=${gv%%:*}
  version=${gv##*:}
  if [ -z "${INPUT_DIRS}" ]; then
    INPUT_DIRS="${APIS_PKG}/${group}/${version}"
  else
    INPUT_DIRS="${INPUT_DIRS},${APIS_PKG}/${group}/${version}"
  fi
done

if is_generator_enabled "deepcopy"; then
  echo ">>> Generating deepcopy..."
  # Convert comma-separated to space-separated
  INPUT_DIRS_SPACE="${INPUT_DIRS//,/ }" 
  deepcopy-gen \
    --go-header-file "${BOILERPLATE_FILE}" \
    --output-file "zz_generated.deepcopy.go" \
    ${INPUT_DIRS_SPACE}
fi

# 2. 生成 Client, Lister, Informer
# client-gen 需要 --input-base 和 --input
# 这里我们将 INPUT_DIRS 拆解回相对路径或者直接用完整路径
# client-gen 的 --input 需要指定的只是包名（相对于 input-base）或者完整包名（如果 input-base为空）

if is_generator_enabled "client"; then
  echo ">>> Generating client..."
  # Transform fully qualified path to relative path for client-gen
  # e.g. github.com/kart-io/sentinel-x/pkg/apis/sentinel/v1 -> sentinel/v1
  CLIENT_INPUTS=""
  for dir in ${INPUT_DIRS//,/ }; do
    REL_DIR=${dir#${GO_MODULE}/pkg/apis/}
    if [ -z "${CLIENT_INPUTS}" ]; then
        CLIENT_INPUTS="${REL_DIR}"
    else
        CLIENT_INPUTS="${CLIENT_INPUTS},${REL_DIR}"
    fi
  done

  client-gen \
    --clientset-name "versioned" \
    --input-base "${GO_MODULE}/pkg/apis" \
    --output-pkg "${GENERATED_PKG}/clientset" \
    --output-dir "pkg/generated/clientset" \
    --go-header-file "${BOILERPLATE_FILE}" \
    --input "${CLIENT_INPUTS}"
fi

if is_generator_enabled "lister"; then
  echo ">>> Generating lister..."
   # Convert comma-separated to space-separated
  INPUT_DIRS_SPACE="${INPUT_DIRS//,/ }" 
  lister-gen \
    --output-pkg "${GENERATED_PKG}/listers" \
    --output-dir "pkg/generated/listers" \
    --go-header-file "${BOILERPLATE_FILE}" \
    ${INPUT_DIRS_SPACE}
fi

if is_generator_enabled "informer"; then
  echo ">>> Generating informer..."
   # Convert comma-separated to space-separated
  INPUT_DIRS_SPACE="${INPUT_DIRS//,/ }" 
  informer-gen \
    --versioned-clientset-package "${GENERATED_PKG}/clientset/versioned" \
    --listers-package "${GENERATED_PKG}/listers" \
    --output-pkg "${GENERATED_PKG}/informers" \
    --output-dir "pkg/generated/informers" \
    --go-header-file "${BOILERPLATE_FILE}" \
    ${INPUT_DIRS_SPACE}
fi

if is_generator_enabled "controller"; then
  echo ">>> Generating controller (CRD, RBAC, etc.)..."
  # Convert comma-separated to space-separated, ensure ./ prefix relative to module if needed, 
  # but controller-gen works well with package paths.
  INPUT_DIRS_SPACE="${INPUT_DIRS//,/ }"
  
  # Default output for CRDs
  CRD_OUTPUT_DIR="deploy/crds"
  mkdir -p "${CRD_OUTPUT_DIR}"

  # Run controller-gen
  # We generate CRDs. (rbac, webhook etc can be added if needed)
  controller-gen \
    crd \
    paths="${INPUT_DIRS//,/;}" \
    output:crd:dir="${CRD_OUTPUT_DIR}"
fi


echo ">>> Generation complete!"
