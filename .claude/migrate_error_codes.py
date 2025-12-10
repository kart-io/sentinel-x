#!/usr/bin/env python3
"""
错误码迁移工具 - 智能替换 goagent 中的旧错误码
"""

import os
import re
from pathlib import Path

GOAGENT_DIR = "/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/staging/src/github.com/kart-io/goagent"

# 错误码映射：旧码 -> 新码
ERROR_CODE_MAPPING = {
    # Agent 相关
    "CodeAgentValidation": "CodeAgentExecution",
    "CodeAgentNotFound": "CodeNotFound",
    "CodeAgentInitialization": "CodeAgentConfig",

    # Tool 相关
    "CodeToolTimeout": "CodeAgentTimeout",
    "CodeToolRetryExhausted": "CodeToolExecution",

    # Middleware 相关
    "CodeMiddlewareExecution": "CodeAgentExecution",
    "CodeMiddlewareChain": "CodeAgentExecution",
    "CodeMiddlewareValidation": "CodeInvalidInput",

    # State 相关
    "CodeStateLoad": "CodeResource",
    "CodeStateSave": "CodeResource",
    "CodeStateValidation": "CodeInvalidInput",
    "CodeStateCheckpoint": "CodeResource",

    # Stream 相关
    "CodeStreamRead": "CodeNetwork",
    "CodeStreamWrite": "CodeNetwork",
    "CodeStreamTimeout": "CodeAgentTimeout",
    "CodeStreamClosed": "CodeResource",

    # LLM 相关
    "CodeLLMRequest": "CodeExternalService",
    "CodeLLMResponse": "CodeExternalService",
    "CodeLLMTimeout": "CodeAgentTimeout",
    "CodeLLMRateLimit": "CodeRateLimit",

    # Context 相关
    "CodeContextCanceled": "CodeAgentTimeout",
    "CodeContextTimeout": "CodeAgentTimeout",

    # Distributed 相关
    "CodeDistributedConnection": "CodeNetwork",
    "CodeDistributedSerialization": "CodeInvalidInput",
    "CodeDistributedCoordination": "CodeNetwork",
    "CodeDistributedScheduling": "CodeAgentExecution",
    "CodeDistributedHeartbeat": "CodeNetwork",
    "CodeDistributedRegistry": "CodeResource",

    # Retrieval/RAG 相关
    "CodeRetrievalSearch": "CodeRetrieval",
    "CodeRetrievalEmbedding": "CodeEmbedding",
    "CodeDocumentNotFound": "CodeNotFound",
    "CodeVectorDimMismatch": "CodeInvalidInput",

    # Planning 相关
    "CodePlanningFailed": "CodeAgentExecution",
    "CodePlanValidation": "CodeInvalidInput",
    "CodePlanExecutionFailed": "CodeAgentExecution",
    "CodePlanNotFound": "CodeNotFound",

    # Parser 相关
    "CodeParserFailed": "CodeInvalidInput",
    "CodeParserInvalidJSON": "CodeInvalidInput",
    "CodeParserMissingField": "CodeInvalidInput",

    # MultiAgent 相关
    "CodeMultiAgentRegistration": "CodeAgentConfig",
    "CodeMultiAgentConsensus": "CodeAgentExecution",
    "CodeMultiAgentMessage": "CodeNetwork",

    # Store 相关
    "CodeStoreConnection": "CodeNetwork",
    "CodeStoreSerialization": "CodeInvalidInput",
    "CodeStoreNotFound": "CodeNotFound",

    # Router 相关
    "CodeRouterNoMatch": "CodeNotFound",
    "CodeRouterFailed": "CodeAgentExecution",
    "CodeRouterOverload": "CodeResourceLimit",

    # 其他通用错误
    "CodeTypeMismatch": "CodeInvalidInput",
    "CodeNotImplemented": "CodeUnknown",
    "CodeInvalidOutput": "CodeInvalidInput",
}

# 删除的 Helper 函数映射到通用函数
HELPER_FUNC_MAPPING = {
    "NewAgentExecutionError": ("NewErrorWithCause", "CodeAgentExecution"),
    "NewAgentValidationError": ("NewError", "CodeAgentExecution"),
    "NewAgentNotFoundError": ("NewError", "CodeNotFound"),
    "NewAgentInitializationError": ("NewErrorWithCause", "CodeAgentConfig"),
    "NewToolExecutionError": ("NewErrorWithCause", "CodeToolExecution"),
    "NewToolNotFoundError": ("NewError", "CodeToolNotFound"),
    "NewToolValidationError": ("NewError", "CodeToolValidation"),
    "NewToolTimeoutError": ("NewError", "CodeAgentTimeout"),
    "NewToolRetryExhaustedError": ("NewErrorWithCause", "CodeToolExecution"),
    "NewMiddlewareExecutionError": ("NewErrorWithCause", "CodeAgentExecution"),
    "NewMiddlewareChainError": ("NewErrorWithCause", "CodeAgentExecution"),
    "NewMiddlewareValidationError": ("NewError", "CodeInvalidInput"),
    "NewStateLoadError": ("NewErrorWithCause", "CodeResource"),
    "NewStateSaveError": ("NewErrorWithCause", "CodeResource"),
    "NewStateValidationError": ("NewError", "CodeInvalidInput"),
    "NewStateCheckpointError": ("NewErrorWithCause", "CodeResource"),
    "NewStreamReadError": ("NewErrorWithCause", "CodeNetwork"),
    "NewStreamWriteError": ("NewErrorWithCause", "CodeNetwork"),
    "NewStreamTimeoutError": ("NewError", "CodeAgentTimeout"),
    "NewStreamClosedError": ("NewError", "CodeResource"),
    "NewLLMRequestError": ("NewErrorWithCause", "CodeExternalService"),
    "NewLLMResponseError": ("NewError", "CodeExternalService"),
    "NewLLMTimeoutError": ("NewError", "CodeAgentTimeout"),
    "NewLLMRateLimitError": ("NewError", "CodeRateLimit"),
    "NewContextCanceledError": ("NewError", "CodeAgentTimeout"),
    "NewContextTimeoutError": ("NewError", "CodeAgentTimeout"),
    "NewInvalidInputError": ("NewError", "CodeInvalidInput"),
    "NewInvalidConfigError": ("NewError", "CodeAgentConfig"),
    "NewNotImplementedError": ("NewError", "CodeUnknown"),
    "NewInternalError": ("NewErrorWithCause", "CodeInternal"),
    "NewDistributedConnectionError": ("NewErrorWithCause", "CodeNetwork"),
    "NewDistributedSerializationError": ("NewErrorWithCause", "CodeInvalidInput"),
    "NewDistributedCoordinationError": ("NewErrorWithCause", "CodeNetwork"),
    "NewRetrievalSearchError": ("NewErrorWithCause", "CodeRetrieval"),
    "NewRetrievalEmbeddingError": ("NewErrorWithCause", "CodeEmbedding"),
    "NewDocumentNotFoundError": ("NewError", "CodeNotFound"),
    "NewVectorDimMismatchError": ("NewError", "CodeInvalidInput"),
    "NewPlanningError": ("NewErrorWithCause", "CodeAgentExecution"),
    "NewPlanValidationError": ("NewError", "CodeInvalidInput"),
    "NewPlanExecutionError": ("NewErrorWithCause", "CodeAgentExecution"),
    "NewPlanNotFoundError": ("NewError", "CodeNotFound"),
    "NewParserError": ("NewErrorWithCause", "CodeInvalidInput"),
    "NewParserInvalidJSONError": ("NewErrorWithCause", "CodeInvalidInput"),
    "NewParserMissingFieldError": ("NewError", "CodeInvalidInput"),
    "NewMultiAgentRegistrationError": ("NewErrorWithCause", "CodeAgentConfig"),
    "NewMultiAgentConsensusError": ("NewError", "CodeAgentExecution"),
    "NewMultiAgentMessageError": ("NewErrorWithCause", "CodeNetwork"),
    "NewStoreConnectionError": ("NewErrorWithCause", "CodeNetwork"),
    "NewStoreSerializationError": ("NewErrorWithCause", "CodeInvalidInput"),
    "NewStoreNotFoundError": ("NewError", "CodeNotFound"),
    "NewRouterNoMatchError": ("NewError", "CodeNotFound"),
    "NewRouterFailedError": ("NewErrorWithCause", "CodeAgentExecution"),
    "NewRouterOverloadError": ("NewError", "CodeResourceLimit"),
}

def process_file(filepath):
    """处理单个 Go 文件"""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()

        original_content = content

        # 1. 替换错误码常量
        for old_code, new_code in ERROR_CODE_MAPPING.items():
            # 处理 errors.CodeXXX 和 agentErrors.CodeXXX 形式
            content = re.sub(rf'\b(errors|agentErrors)\.{old_code}\b', rf'\1.{new_code}', content)

        # 2. 替换特定的 helper 函数调用（保留参数）
        # 这部分比较复杂，暂时跳过复杂函数调用的自动替换

        # 如果内容有变化，写回文件
        if content != original_content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(content)
            return True
        return False
    except Exception as e:
        print(f"处理文件 {filepath} 时出错: {e}")
        return False

def main():
    """主函数"""
    print("开始迁移错误码...")

    # 查找所有 Go 文件（排除 vendor 和 errors 目录）
    go_files = []
    for root, dirs, files in os.walk(GOAGENT_DIR):
        # 排除 vendor 和 errors 目录
        if 'vendor' in root or '/errors' in root:
            continue

        for file in files:
            if file.endswith('.go'):
                go_files.append(os.path.join(root, file))

    print(f"找到 {len(go_files)} 个 Go 文件")

    # 处理文件
    modified_count = 0
    for filepath in go_files:
        if process_file(filepath):
            modified_count += 1
            print(f"✓ 已修改: {filepath}")

    print(f"\n完成! 共修改了 {modified_count} 个文件")
    print("\n注意: 某些复杂的函数调用需要手动检查和修改")
    print("请运行以下命令验证编译:")
    print(f"cd {GOAGENT_DIR} && go build ./...")

if __name__ == "__main__":
    main()
