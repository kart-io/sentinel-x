#!/bin/bash

# 错误码迁移脚本
# 将 goagent 子模块中的旧错误码替换为新的简化错误码

set -e

GOAGENT_DIR="/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/staging/src/github.com/kart-io/goagent"

echo "开始迁移错误码..."

# 定义错误码映射
# 旧错误码 -> 新错误码

# Agent 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeAgentValidation/CodeAgentExecution/g' \
  -e 's/CodeAgentNotFound/CodeNotFound/g' \
  -e 's/CodeAgentInitialization/CodeAgentConfig/g' \
  {} +

# Tool 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeToolTimeout/CodeAgentTimeout/g' \
  -e 's/CodeToolRetryExhausted/CodeToolExecution/g' \
  {} +

# Middleware 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeMiddlewareExecution/CodeAgentExecution/g' \
  -e 's/CodeMiddlewareChain/CodeAgentExecution/g' \
  -e 's/CodeMiddlewareValidation/CodeInvalidInput/g' \
  {} +

# State 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeStateLoad/CodeResource/g' \
  -e 's/CodeStateSave/CodeResource/g' \
  -e 's/CodeStateValidation/CodeInvalidInput/g' \
  -e 's/CodeStateCheckpoint/CodeResource/g' \
  {} +

# Stream 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeStreamRead/CodeNetwork/g' \
  -e 's/CodeStreamWrite/CodeNetwork/g' \
  -e 's/CodeStreamTimeout/CodeAgentTimeout/g' \
  -e 's/CodeStreamClosed/CodeResource/g' \
  {} +

# LLM 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeLLMRequest/CodeExternalService/g' \
  -e 's/CodeLLMResponse/CodeExternalService/g' \
  -e 's/CodeLLMTimeout/CodeAgentTimeout/g' \
  -e 's/CodeLLMRateLimit/CodeRateLimit/g' \
  {} +

# Context 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeContextCanceled/CodeAgentTimeout/g' \
  -e 's/CodeContextTimeout/CodeAgentTimeout/g' \
  {} +

# Distributed 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeDistributedConnection/CodeNetwork/g' \
  -e 's/CodeDistributedSerialization/CodeInvalidInput/g' \
  -e 's/CodeDistributedCoordination/CodeNetwork/g' \
  -e 's/CodeDistributedScheduling/CodeAgentExecution/g' \
  -e 's/CodeDistributedHeartbeat/CodeNetwork/g' \
  -e 's/CodeDistributedRegistry/CodeResource/g' \
  {} +

# Retrieval/RAG 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeRetrievalSearch/CodeRetrieval/g' \
  -e 's/CodeRetrievalEmbedding/CodeEmbedding/g' \
  -e 's/CodeDocumentNotFound/CodeNotFound/g' \
  -e 's/CodeVectorDimMismatch/CodeInvalidInput/g' \
  {} +

# Planning 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodePlanningFailed/CodeAgentExecution/g' \
  -e 's/CodePlanValidation/CodeInvalidInput/g' \
  -e 's/CodePlanExecutionFailed/CodeAgentExecution/g' \
  -e 's/CodePlanNotFound/CodeNotFound/g' \
  {} +

# Parser 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeParserFailed/CodeInvalidInput/g' \
  -e 's/CodeParserInvalidJSON/CodeInvalidInput/g' \
  -e 's/CodeParserMissingField/CodeInvalidInput/g' \
  {} +

# MultiAgent 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeMultiAgentRegistration/CodeAgentConfig/g' \
  -e 's/CodeMultiAgentConsensus/CodeAgentExecution/g' \
  -e 's/CodeMultiAgentMessage/CodeNetwork/g' \
  {} +

# Store 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeStoreConnection/CodeNetwork/g' \
  -e 's/CodeStoreSerialization/CodeInvalidInput/g' \
  -e 's/CodeStoreNotFound/CodeNotFound/g' \
  {} +

# Router 相关错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeRouterNoMatch/CodeNotFound/g' \
  -e 's/CodeRouterFailed/CodeAgentExecution/g' \
  -e 's/CodeRouterOverload/CodeResourceLimit/g' \
  {} +

# 其他通用错误
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/CodeTypeMismatch/CodeInvalidInput/g' \
  -e 's/CodeInvalidConfig/CodeAgentConfig/g' \
  -e 's/CodeNotImplemented/CodeUnknown/g' \
  -e 's/CodeInvalidOutput/CodeInvalidInput/g' \
  {} +

# 删除旧的 helper 函数调用，替换为通用函数
find "$GOAGENT_DIR" -name "*.go" -type f -not -path "*/vendor/*" -exec sed -i \
  -e 's/NewAgentExecutionError(/NewErrorWithCause(CodeAgentExecution, /g' \
  -e 's/NewAgentValidationError(/NewError(CodeAgentExecution, /g' \
  -e 's/NewAgentNotFoundError(/NewError(CodeNotFound, /g' \
  -e 's/NewAgentInitializationError(/NewErrorWithCause(CodeAgentConfig, /g' \
  -e 's/NewToolExecutionError(/NewErrorWithCause(CodeToolExecution, /g' \
  -e 's/NewToolNotFoundError(/NewError(CodeToolNotFound, /g' \
  -e 's/NewToolValidationError(/NewError(CodeToolValidation, /g' \
  -e 's/NewToolTimeoutError(/NewError(CodeAgentTimeout, /g' \
  -e 's/NewToolRetryExhaustedError(/NewErrorWithCause(CodeToolExecution, /g' \
  -e 's/NewMiddlewareExecutionError(/NewErrorWithCause(CodeAgentExecution, /g' \
  -e 's/NewMiddlewareChainError(/NewErrorWithCause(CodeAgentExecution, /g' \
  -e 's/NewMiddlewareValidationError(/NewError(CodeInvalidInput, /g' \
  -e 's/NewStateLoadError(/NewErrorWithCause(CodeResource, /g' \
  -e 's/NewStateSaveError(/NewErrorWithCause(CodeResource, /g' \
  -e 's/NewStateValidationError(/NewError(CodeInvalidInput, /g' \
  -e 's/NewStateCheckpointError(/NewErrorWithCause(CodeResource, /g' \
  -e 's/NewStreamReadError(/NewErrorWithCause(CodeNetwork, /g' \
  -e 's/NewStreamWriteError(/NewErrorWithCause(CodeNetwork, /g' \
  -e 's/NewStreamTimeoutError(/NewError(CodeAgentTimeout, /g' \
  -e 's/NewStreamClosedError(/NewError(CodeResource, /g' \
  -e 's/NewLLMRequestError(/NewErrorWithCause(CodeExternalService, /g' \
  -e 's/NewLLMResponseError(/NewError(CodeExternalService, /g' \
  -e 's/NewLLMTimeoutError(/NewError(CodeAgentTimeout, /g' \
  -e 's/NewLLMRateLimitError(/NewError(CodeRateLimit, /g' \
  -e 's/NewContextCanceledError(/NewError(CodeAgentTimeout, /g' \
  -e 's/NewContextTimeoutError(/NewError(CodeAgentTimeout, /g' \
  -e 's/NewInvalidInputError(/NewError(CodeInvalidInput, /g' \
  -e 's/NewInvalidConfigError(/NewError(CodeAgentConfig, /g' \
  -e 's/NewNotImplementedError(/NewError(CodeUnknown, /g' \
  -e 's/NewInternalError(/NewErrorWithCause(CodeInternal, /g' \
  -e 's/NewDistributedConnectionError(/NewErrorWithCause(CodeNetwork, /g' \
  -e 's/NewDistributedSerializationError(/NewErrorWithCause(CodeInvalidInput, /g' \
  -e 's/NewDistributedCoordinationError(/NewErrorWithCause(CodeNetwork, /g' \
  -e 's/NewRetrievalSearchError(/NewErrorWithCause(CodeRetrieval, /g' \
  -e 's/NewRetrievalEmbeddingError(/NewErrorWithCause(CodeEmbedding, /g' \
  -e 's/NewDocumentNotFoundError(/NewError(CodeNotFound, /g' \
  -e 's/NewVectorDimMismatchError(/NewError(CodeInvalidInput, /g' \
  -e 's/NewPlanningError(/NewErrorWithCause(CodeAgentExecution, /g' \
  -e 's/NewPlanValidationError(/NewError(CodeInvalidInput, /g' \
  -e 's/NewPlanExecutionError(/NewErrorWithCause(CodeAgentExecution, /g' \
  -e 's/NewPlanNotFoundError(/NewError(CodeNotFound, /g' \
  -e 's/NewParserError(/NewErrorWithCause(CodeInvalidInput, /g' \
  -e 's/NewParserInvalidJSONError(/NewErrorWithCause(CodeInvalidInput, /g' \
  -e 's/NewParserMissingFieldError(/NewError(CodeInvalidInput, /g' \
  -e 's/NewMultiAgentRegistrationError(/NewErrorWithCause(CodeAgentConfig, /g' \
  -e 's/NewMultiAgentConsensusError(/NewError(CodeAgentExecution, /g' \
  -e 's/NewMultiAgentMessageError(/NewErrorWithCause(CodeNetwork, /g' \
  -e 's/NewStoreConnectionError(/NewErrorWithCause(CodeNetwork, /g' \
  -e 's/NewStoreSerializationError(/NewErrorWithCause(CodeInvalidInput, /g' \
  -e 's/NewStoreNotFoundError(/NewError(CodeNotFound, /g' \
  -e 's/NewRouterNoMatchError(/NewError(CodeNotFound, /g' \
  -e 's/NewRouterFailedError(/NewErrorWithCause(CodeAgentExecution, /g' \
  -e 's/NewRouterOverloadError(/NewError(CodeResourceLimit, /g' \
  {} +

echo "错误码迁移完成！"
echo ""
echo "请运行以下命令验证编译："
echo "cd $GOAGENT_DIR && go build ./..."
