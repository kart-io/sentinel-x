# 错误码简化迁移报告

## 执行日期
2025-12-10

## 目标
简化 sentinel-x 项目 goagent 子模块的错误码系统，从 105 个错误码简化为约 20 个核心错误码。

## 已完成的工作

### 1. 错误码定义简化 ✅

**文件**: `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/staging/src/github.com/kart-io/goagent/errors/errors.go`

**修改内容**:
- 将 105 个细分错误码合并为 20 个核心错误码
- 保留的核心错误码分类:
  - 通用错误 (1-5): CodeUnknown, CodeInvalidInput, CodeNotFound, CodeAlreadyExists, CodePermissionDenied
  - Agent 错误 (10-15): CodeAgentExecution, CodeAgentTimeout, CodeAgentConfig
  - 工具错误 (20-25): CodeToolExecution, CodeToolNotFound, CodeToolValidation
  - 检索/RAG 错误 (30-35): CodeRetrieval, CodeEmbedding, CodeVectorStore
  - 网络/外部服务错误 (40-45): CodeNetwork, CodeExternalService, CodeRateLimit
  - 资源错误 (50-55): CodeResource, CodeResourceLimit
  - 内部错误 (99): CodeInternal

### 2. Helper 函数简化 ✅

**文件**: `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/staging/src/github.com/kart-io/goagent/errors/helpers.go`

**修改内容**:
- 删除了约 60 个细分的工厂函数
- 保留通用工厂函数:
  - `NewError(code ErrorCode, message string) *AgentError`
  - `NewErrorWithCause(code ErrorCode, message string, cause error) *AgentError`
  - `NewErrorf(code ErrorCode, format string, args ...interface{}) *AgentError`
  - `ErrorWithRetry(err error, attempt, maxAttempts int) *AgentError`
  - `ErrorWithDuration(err error, durationMs int64) *AgentError`
  - `ErrorWithContext(err error, key string, value interface{}) *AgentError`
- 添加了辅助检查函数:
  - `IsContextCanceled(err error) bool`
  - `IsContextTimeout(err error) bool`
  - `IsNotFound(err error) bool`
  - `IsValidationError(err error) bool`

### 3. 错误码映射 ✅

创建了完整的旧错误码到新错误码的映射表:
- Agent 相关: CodeAgentValidation → CodeAgentExecution
- Tool 相关: CodeToolTimeout → CodeAgentTimeout
- LLM 相关: CodeLLMRequest → CodeExternalService
- State 相关: CodeStateLoad → CodeResource
- Stream 相关: CodeStreamWrite → CodeNetwork
- 等等...

### 4. 自动化迁移工具 ✅

创建了以下迁移工具:
1. **error-code-migration.sh**: Bash 脚本，自动替换错误码
2. **migrate_error_codes.py**: Python 脚本，智能替换错误码常量
3. **fix-helper-funcs.sh**: Perl 脚本，修复 helper 函数调用

### 5. 部分代码迁移 ⚠️

**已修复的文件**:
- `/staging/src/github.com/kart-io/goagent/store/langgraph_store.go` - 添加 fmt 导入并修复错误调用
- `/staging/src/github.com/kart-io/goagent/llm/factory.go` - 修复配置验证错误
- `/staging/src/github.com/kart-io/goagent/llm/common/base.go` - 修复 LLM 请求错误处理

**自动迁移完成的文件** (573 个文件中的 2 个):
- `core/checkpoint/redis.go`
- `examples/error_handling/main.go`

## 待完成的工作

### 1. 修复测试文件 ⚠️

**文件**: `/staging/src/github.com/kart-io/goagent/errors/errors_test.go`

**问题**: 测试文件中仍使用旧的函数签名和已删除的 helper 函数

**需要的修改**:
```go
// 旧代码 (错误)
err := NewErrorWithCause(CodeAgentExecution, "test-agent", "run", cause)

// 新代码 (正确)
err := NewErrorWithCause(CodeAgentExecution, "agent execution failed", cause).
    WithComponent("agent").
    WithOperation("run").
    WithContext("agent_name", "test-agent")
```

### 2. 修复项目中使用旧 helper 函数的代码 ⚠️

**受影响的文件** (需要手动检查和修复):
- `examples/error_handling/main.go` - 使用了 NewLLMRequestError, NewToolExecutionError 等
- `core/checkpoint/distributed.go` - 使用了已删除的函数
- `core/checkpoint/redis.go` - 使用了 NewStoreConnectionError
- `llm/common/base.go` - 使用了 NewInvalidInputError, NewInvalidConfigError
- `mcp/core/tool.go` - 使用了 ErrNotImplemented
- 其他 573 个 Go 文件需要扫描

**迁移模式**:
```go
// 旧模式
NewLLMRequestError(provider, model, cause)

// 新模式
NewErrorWithCause(CodeExternalService, "LLM request failed", cause).
    WithComponent("llm").
    WithOperation("request").
    WithContext("provider", provider).
    WithContext("model", model)
```

### 3. 完整编译验证 ⚠️

当前编译状态: **失败**

**主要错误类型**:
1. 未定义的 helper 函数 (如 NewLLMRequestError, NewToolExecutionError)
2. 函数参数数量不匹配
3. 未定义的常量 (已被删除的错误码)

**验证命令**:
```bash
cd /home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/staging/src/github.com/kart-io/goagent
go build ./...
go test ./...
```

## 错误码映射参考

| 旧错误码 | 新错误码 | 用途 |
|---------|---------|------|
| CodeAgentValidation | CodeAgentExecution | Agent 验证失败 |
| CodeAgentNotFound | CodeNotFound | Agent 未找到 |
| CodeAgentInitialization | CodeAgentConfig | Agent 初始化失败 |
| CodeToolTimeout | CodeAgentTimeout | 工具超时 |
| CodeToolRetryExhausted | CodeToolExecution | 工具重试耗尽 |
| CodeLLMRequest | CodeExternalService | LLM 请求失败 |
| CodeLLMResponse | CodeExternalService | LLM 响应错误 |
| CodeLLMTimeout | CodeAgentTimeout | LLM 超时 |
| CodeLLMRateLimit | CodeRateLimit | LLM 速率限制 |
| CodeStateLoad | CodeResource | 状态加载失败 |
| CodeStateSave | CodeResource | 状态保存失败 |
| CodeStreamRead | CodeNetwork | 流读取失败 |
| CodeStreamWrite | CodeNetwork | 流写入失败 |
| CodeDistributedConnection | CodeNetwork | 分布式连接失败 |
| CodeRetrievalSearch | CodeRetrieval | 检索搜索失败 |
| CodeRetrievalEmbedding | CodeEmbedding | 嵌入生成失败 |
| CodeDocumentNotFound | CodeNotFound | 文档未找到 |
| CodePlanningFailed | CodeAgentExecution | 计划失败 |
| CodeParserFailed | CodeInvalidInput | 解析失败 |
| CodeMultiAgentMessage | CodeNetwork | 多 Agent 消息错误 |
| CodeStoreConnection | CodeNetwork | 存储连接失败 |
| CodeRouterNoMatch | CodeNotFound | 路由未匹配 |
| CodeInvalidConfig | CodeAgentConfig | 配置无效 |
| CodeNotImplemented | CodeUnknown | 未实现 |

## 使用新 API 的示例

### 创建错误
```go
// 简单错误
err := errors.NewError(errors.CodeInvalidInput, "invalid parameter value")

// 带原因的错误
err := errors.NewErrorWithCause(errors.CodeNetwork, "connection failed", originalErr)

// 格式化消息
err := errors.NewErrorf(errors.CodeAgentTimeout, "operation timed out after %d seconds", timeout)
```

### 添加上下文
```go
err := errors.NewError(errors.CodeNotFound, "resource not found").
    WithComponent("database").
    WithOperation("query").
    WithContext("table", "users").
    WithContext("id", userID)
```

### 错误检查
```go
if errors.IsCode(err, errors.CodeNotFound) {
    // 处理未找到错误
}

if errors.IsNotFound(err) {
    // 使用辅助函数检查
}

if errors.IsValidationError(err) {
    // 处理验证错误
}
```

## 后续步骤

1. **立即** - 修复 `errors/errors_test.go` 中的测试用例
2. **高优先级** - 使用 `grep` 找到所有使用旧 helper 函数的地方并修复
3. **中优先级** - 批量替换项目中的错误码常量引用
4. **验证** - 确保所有包都能成功编译
5. **测试** - 运行完整的测试套件
6. **文档** - 更新 errors 包的 README.md

## 工具和脚本

所有迁移工具已保存在:
- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/.claude/error-code-migration.sh`
- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/.claude/fix-migration-errors.sh`
- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/.claude/migrate_error_codes.py`
- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/.claude/fix-helper-funcs.sh`

## 总结

### 已完成
✅ 错误码定义简化 (105 → 20)
✅ Helper 函数简化 (60 → 3 核心 + 6 辅助)
✅ 创建自动化迁移工具
✅ 部分代码文件修复

### 待完成
⚠️ 修复测试文件
⚠️ 完整的代码库迁移 (573 个文件)
⚠️ 编译验证和测试验证

### 风险
- 由于修改范围大，可能影响大量现有代码
- 需要仔细测试以确保错误处理逻辑正确
- 某些复杂的错误场景可能需要手动调整

### 建议
1. 分批次进行迁移,每次修复一个模块
2. 优先修复核心模块 (errors, core, llm)
3. 在修复后立即运行测试确保功能正常
4. 考虑编写迁移指南帮助团队成员理解新 API
