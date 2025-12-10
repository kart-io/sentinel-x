#!/bin/bash

# 修复 helper 函数调用 - 简化版本
# 将所有旧的 helper 函数替换为通用函数

set -e

GOAGENT_DIR="/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/staging/src/github.com/kart-io/goagent"

echo "修复 Helper 函数调用..."

# 对于每个需要三个参数的旧函数，将其替换为 NewError 并使用 WithComponent/WithContext
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -not -path "*/errors/*" -exec perl -i -pe '
s/agentErrors\.NewInvalidInputError\(([^,]+),\s*([^,]+),\s*([^)]+)\)/agentErrors.NewError(agentErrors.CodeInvalidInput, $3).WithComponent($1).WithContext("field", $2)/g;
s/agentErrors\.NewInvalidConfigError\(([^,]+),\s*([^,]+),\s*([^)]+)\)/agentErrors.NewError(agentErrors.CodeAgentConfig, $3).WithComponent($1).WithContext("field", $2)/g;
' {} +

echo "修复完成！"
