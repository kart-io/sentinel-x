# 错误处理示例

本示例演示了 GoAgent 中正确的错误处理模式，涵盖从基础到高级的各种场景。

## 主要变更

本示例已更新为使用新的错误处理 API，移除了所有旧的辅助函数。

### 已删除的旧函数

- `NewLLMRequestError`
- `NewToolExecutionError`
- `NewDocumentNotFoundError`
- `NewPlanExecutionError`
- `NewLLMRateLimitError`
- `NewToolTimeoutError`
- `NewToolRetryExhaustedError`
- `NewStoreConnectionError`
- `NewInvalidInputError`

### 新的错误 API

#### 核心错误创建函数

```go
// 创建简单错误
errors.NewError(code ErrorCode, message string) *AgentError

// 创建带原因的错误
errors.NewErrorWithCause(code ErrorCode, message string, cause error) *AgentError

// 创建格式化错误
errors.NewErrorf(code ErrorCode, format string, args ...interface{}) *AgentError

// 包装现有错误
errors.Wrap(err error, code ErrorCode, message string) *AgentError
```

#### 链式上下文添加

```go
err := errors.NewError(errors.CodeRateLimit, "rate limit exceeded").
    WithComponent("llm_client").
    WithOperation("request").
    WithContext("provider", "openai").
    WithContext("model", "gpt-4")
```

## 运行示例

```bash
cd examples/error_handling
go run main.go
```

## 包含的示例

### 示例 1: 创建不同类型的错误

演示如何使用新的错误 API 创建各种类型的错误：

- LLM 请求错误（使用 `CodeAgentExecution`）
- 工具执行错误（使用 `CodeToolExecution`）
- 文档未找到错误（使用 `CodeNotFound`）
- 计划执行错误（使用 `CodeAgentExecution`）

### 示例 2: 链式添加上下文

演示如何使用方法链为错误添加丰富的上下文信息。

### 示例 3: 错误包装

演示如何正确包装错误以保持错误链，以及如何遍历错误链和获取根因。

### 示例 4: 错误检查和分支处理

演示如何使用 `errors.IsCode()` 进行类型安全的错误检查，以及如何根据不同错误类型实现不同的处理逻辑。

### 示例 5: 重试逻辑

演示在实际场景中如何实现智能重试逻辑，特别是处理 LLM 速率限制错误。

### 示例 6: 降级处理

演示当主服务不可用时如何降级到备份服务，展示了基于错误类型的服务降级模式。

### 示例 7: 错误转换

演示如何将内部错误代码转换为 HTTP 状态码，适用于 API 层。

### 示例 8: 结构化日志

演示如何提取 AgentError 的所有字段用于结构化日志记录，包括堆栈跟踪。

### 示例 9: 批处理错误聚合

演示在批处理场景中如何收集和聚合多个错误，提供完整的批次处理报告。

### 示例 10: 错误链分析

演示如何分析和展示多层嵌套的错误链，对调试复杂错误场景很有帮助。

## 错误码映射表

| 旧函数                         | 新错误码                  | 说明           |
|-------------------------------|-------------------------|---------------|
| NewLLMRequestError            | CodeAgentExecution      | Agent 执行失败 |
| NewPlanExecutionError         | CodeAgentExecution      | Agent 执行失败 |
| NewToolExecutionError         | CodeToolExecution       | 工具执行失败   |
| NewDocumentNotFoundError      | CodeNotFound            | 资源未找到     |
| NewLLMRateLimitError          | CodeRateLimit           | 速率限制       |
| NewToolTimeoutError           | CodeAgentTimeout        | 超时错误       |
| NewToolRetryExhaustedError    | CodeToolExecution       | 工具执行失败   |
| NewStoreConnectionError       | CodeNetwork             | 网络错误       |
| NewInvalidInputError          | CodeInvalidInput        | 无效输入       |

## 迁移指南

### 旧代码

```go
// LLM 请求错误
err := errors.NewLLMRequestError("openai", "gpt-4", fmt.Errorf("API connection failed"))

// 文档未找到
err := errors.NewDocumentNotFoundError("doc-123")

// 工具超时
err := errors.NewToolTimeoutError("web_scraper", 30)
```

### 新代码

```go
// LLM 请求错误
err := errors.NewErrorWithCause(
    errors.CodeAgentExecution,
    "failed to call LLM API",
    fmt.Errorf("API connection failed"),
).WithComponent("openai").WithContext("model", "gpt-4")

// 文档未找到
err := errors.NewErrorf(errors.CodeNotFound, "document not found: doc-123")

// 工具超时
err := errors.NewError(errors.CodeAgentTimeout, "tool execution timeout").
    WithComponent("web_scraper").
    WithContext("timeout_seconds", 30)
```

## 关键要点

1. **使用新的错误 API**: 使用 `NewError()`, `NewErrorWithCause()`, `NewErrorf()` 代替旧的辅助函数
2. **类型检查**: 使用 `errors.IsCode()` 而非字符串比较
3. **保持错误链**: 使用 `errors.Wrap()` 或 `NewErrorWithCause()` 包装已有错误
4. **添加上下文**: 使用 `WithContext()`, `WithComponent()`, `WithOperation()` 添加调试所需的关键信息
5. **智能重试**: 只对可重试的错误（如速率限制、超时）进行重试
6. **降级策略**: 基于错误类型实现服务降级

## 相关文档

- [错误处理核心代码](../../errors/errors.go)
- [错误辅助函数](../../errors/helpers.go)
