# 测试修复计划

生成时间: 2026-01-08 17:50

## 📋 问题概述

Token 统计功能实现后，接口签名从 `(string, error)` 变更为 `(*GenerateResponse, error)`，导致所有测试中的 mock 对象和断言失败。

## 🔍 识别的问题

### 编译失败的测试包
1. `pkg/llm` - Mock Provider 接口不匹配
2. `pkg/llm/openai` - Mock 或测试断言问题
3. `pkg/llm/siliconflow` - Mock 或测试断言问题
4. `internal/pkg/rag/enhancer` - 调用方式问题
5. `internal/pkg/rag/evaluator` - 调用方式问题

## 📝 修复策略

### 1. pkg/llm (provider_test.go)
**问题**: mockProvider 的 Generate 方法返回 `string` 而不是 `*GenerateResponse`

**修复**:
```go
// 修改前
func (m *mockProvider) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
    return "test response", nil
}

// 修改后
func (m *mockProvider) Generate(ctx context.Context, prompt string, systemPrompt string) (*GenerateResponse, error) {
    return &GenerateResponse{
        Content: "test response",
        TokenUsage: &TokenUsage{
            PromptTokens:     10,
            CompletionTokens: 20,
            TotalTokens:      30,
        },
    }, nil
}
```

### 2. pkg/llm/openai (provider_test.go)
**问题**: 测试中期望 `string` 返回值

**修复**:
- 更新断言检查 `resp.Content` 而不是直接的字符串
- 添加 `resp.TokenUsage` 的验证

### 3. pkg/llm/siliconflow (provider_test.go)
**问题**: 同 openai

**修复**: 同 openai

### 4. internal/pkg/rag/enhancer (enhancer_test.go)
**问题**: 测试中的 mock ChatProvider 返回旧格式

**修复**:
- 更新 mock 返回 `*GenerateResponse`
- 更新测试断言

### 5. internal/pkg/rag/evaluator (evaluator_test.go)
**问题**: 同 enhancer

**修复**: 同 enhancer

## ✅ 修复检查清单

- [ ] 修复 pkg/llm/provider_test.go
- [ ] 修复 pkg/llm/openai/provider_test.go
- [ ] 修复 pkg/llm/siliconflow/provider_test.go
- [ ] 修复 internal/pkg/rag/enhancer/enhancer_test.go
- [ ] 修复 internal/pkg/rag/evaluator/evaluator_test.go
- [ ] 验证所有测试编译通过
- [ ] 验证所有测试运行通过

## 🎯 预期结果

- 所有测试编译通过
- 所有测试运行通过
- Mock 对象正确返回 `*GenerateResponse`
- 测试断言验证 `Content` 和 `TokenUsage` 字段
