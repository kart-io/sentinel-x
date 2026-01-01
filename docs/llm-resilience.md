# LLM 韧性层使用指南

本文档说明如何使用 LLM 韧性层（Resilience Layer）来提升 LLM 调用的可靠性和稳定性。

## 概述

LLM 韧性层提供以下功能：

1. **重试机制（Retry）**: 指数退避重试失败的 LLM 调用
2. **熔断器（Circuit Breaker）**: 快速失败，避免级联故障
3. **智能错误判断**: 自动识别可重试的错误（网络错误、超时、5xx 错误等）

## 架构

```
用户代码
   ↓
ResilientChatProvider / ResilientEmbeddingProvider
   ↓
RetryWithCircuitBreaker
   ├── RetryWithBackoff（重试逻辑 + 指数退避）
   └── CircuitBreaker（熔断器状态管理）
       ↓
   原始 LLM Provider（DeepSeek, OpenAI, Ollama 等）
```

## 快速开始

### 1. 包装现有 LLM Provider

```go
import (
    "github.com/kart-io/sentinel-x/pkg/llm"
    "github.com/kart-io/sentinel-x/pkg/llm/resilience"
)

// 创建原始 provider
provider, err := llm.NewChatProvider("deepseek", config)
if err != nil {
    return err
}

// 包装为韧性 provider
resilientProvider := resilience.NewResilientChatProvider(
    provider,
    nil, // 使用默认重试配置
    nil, // 使用默认熔断器配置
)

// 使用方式与普通 provider 完全相同
response, err := resilientProvider.Generate(ctx, prompt, systemPrompt)
```

### 2. 自定义重试配置

```go
retryConfig := &resilience.RetryConfig{
    MaxAttempts:  5,                      // 最多重试 5 次
    InitialDelay: 1 * time.Second,        // 初始延迟 1 秒
    MaxDelay:     30 * time.Second,       // 最大延迟 30 秒
    Multiplier:   2.0,                    // 延迟倍增因子
    RetryableErrors: func(err error) bool {
        // 自定义错误判断逻辑
        return resilience.IsRetryableError(err)
    },
}

resilientProvider := resilience.NewResilientChatProvider(
    provider,
    retryConfig,
    nil,
)
```

### 3. 自定义熔断器配置

```go
cbConfig := &resilience.CircuitBreakerConfig{
    MaxFailures:      10,              // 10 次失败后打开熔断器
    Timeout:          2 * time.Minute, // 2 分钟后尝试半开
    HalfOpenMaxCalls: 3,               // 半开状态允许 3 次调用
}

resilientProvider := resilience.NewResilientChatProvider(
    provider,
    nil,
    cbConfig,
)
```

## 配置参数详解

### RetryConfig 重试配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `MaxAttempts` | int | 3 | 最大尝试次数（包括首次调用） |
| `InitialDelay` | time.Duration | 500ms | 初始延迟时间 |
| `MaxDelay` | time.Duration | 10s | 最大延迟时间 |
| `Multiplier` | float64 | 2.0 | 延迟倍增因子（指数退避） |
| `RetryableErrors` | func(error) bool | `IsRetryableError` | 可重试错误判断函数 |

**延迟计算公式**:
```
delay(n) = min(InitialDelay * Multiplier^(n-1), MaxDelay)
```

示例（默认配置）:
- 第 1 次尝试：立即
- 第 2 次尝试：延迟 500ms
- 第 3 次尝试：延迟 1000ms (500 * 2)

### CircuitBreakerConfig 熔断器配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `MaxFailures` | int | 5 | 触发熔断的最大失败次数 |
| `Timeout` | time.Duration | 60s | 熔断器打开后的超时时间 |
| `HalfOpenMaxCalls` | int | 1 | 半开状态允许的最大调用次数 |

### 熔断器状态机

```
        成功 < MaxFailures
  ┌──────────────────────┐
  │                      │
  │    Closed（关闭）     │ ─── 失败 ≥ MaxFailures ──→  Open（打开）
  │    正常工作           │                               │
  └──────────────────────┘                               │
            ↑                                            │
            │                                            │
    所有调用成功                                  Timeout 后
            │                                            │
            │                                            ↓
  ┌──────────────────────┐                    ┌──────────────────────┐
  │  Half-Open（半开）    │ ←── 任意调用失败 ─  │   Open（打开）         │
  │  允许部分探测         │                    │   拒绝所有请求         │
  └──────────────────────┘                    └──────────────────────┘
```

## 可重试错误类型

`IsRetryableError` 函数自动识别以下可重试的错误：

### ✅ 可重试错误

1. **网络错误**
   - 连接超时（Timeout）
   - 临时性网络错误（Temporary）
   - DNS 解析错误
   - 连接重置（Connection Reset）
   - EOF 错误

2. **HTTP 状态码**
   - `408` Request Timeout
   - `429` Too Many Requests（速率限制）
   - `500` Internal Server Error
   - `502` Bad Gateway
   - `503` Service Unavailable
   - `504` Gateway Timeout

3. **其他临时性错误**
   - 服务不可用
   - Rate Limit 错误

### ❌ 不可重试错误

1. **上下文错误**
   - `context.Canceled`
   - `context.DeadlineExceeded`

2. **熔断器错误**
   - `ErrCircuitBreakerOpen`

3. **客户端错误（4xx）**
   - `400` Bad Request
   - `401` Unauthorized
   - `403` Forbidden
   - `404` Not Found

4. **业务逻辑错误**
   - 无效参数
   - 权限错误

## 监控和观测

### 获取熔断器状态

```go
// Chat Provider
stats := resilience.GetChatProviderStats(resilientProvider)
if stats != nil {
    fmt.Printf("熔断器状态: %s\n", stats.CircuitBreakerState)
    fmt.Printf("失败次数: %d\n", stats.CircuitBreakerFailures)
}

// Embedding Provider
stats := resilience.GetEmbeddingProviderStats(resilientProvider)
```

### 直接访问熔断器

```go
cb := resilientProvider.CircuitBreaker()

// 获取当前状态
state := cb.State() // StateClosed, StateOpen, StateHalfOpen

// 获取详细统计
stats := cb.Stats()
fmt.Printf("状态: %s\n", stats["state"])
fmt.Printf("失败次数: %d\n", stats["failures"])
fmt.Printf("上次失败时间: %v\n", stats["last_failure_time"])
```

### 重置熔断器

```go
cb := resilientProvider.CircuitBreaker()
cb.Reset() // 强制关闭熔断器
```

## 使用场景

### 场景 1: RAG 查询（高可用性要求）

```go
retryConfig := &resilience.RetryConfig{
    MaxAttempts:  5,              // 多次重试
    InitialDelay: 1 * time.Second,
    MaxDelay:     30 * time.Second,
    Multiplier:   2.0,
}

cbConfig := &resilience.CircuitBreakerConfig{
    MaxFailures:      10,             // 容忍更多失败
    Timeout:          2 * time.Minute,
    HalfOpenMaxCalls: 3,
}

resilientProvider := resilience.NewResilientChatProvider(
    provider,
    retryConfig,
    cbConfig,
)
```

### 场景 2: 批量处理（快速失败）

```go
retryConfig := &resilience.RetryConfig{
    MaxAttempts:  2,              // 少量重试
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     1 * time.Second,
    Multiplier:   2.0,
}

cbConfig := &resilience.CircuitBreakerConfig{
    MaxFailures:      3,              // 快速熔断
    Timeout:          30 * time.Second,
    HalfOpenMaxCalls: 1,
}

resilientProvider := resilience.NewResilientEmbeddingProvider(
    provider,
    retryConfig,
    cbConfig,
)
```

### 场景 3: 实时交互（低延迟优先）

```go
retryConfig := &resilience.RetryConfig{
    MaxAttempts:  2,              // 最小重试
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     500 * time.Millisecond,
    Multiplier:   1.5,
}

cbConfig := &resilience.CircuitBreakerConfig{
    MaxFailures:      5,
    Timeout:          30 * time.Second,
    HalfOpenMaxCalls: 1,
}

resilientProvider := resilience.NewResilientChatProvider(
    provider,
    retryConfig,
    cbConfig,
)
```

## 最佳实践

### ✅ 推荐做法

1. **始终使用默认配置开始**
   ```go
   resilientProvider := resilience.NewResilientChatProvider(provider, nil, nil)
   ```

2. **根据场景调整参数**
   - 批量任务：较少重试，快速熔断
   - 关键业务：较多重试，容忍更多失败
   - 实时交互：低延迟优先

3. **监控熔断器状态**
   - 在日志中记录熔断器打开事件
   - 将熔断器状态暴露给监控系统
   - 设置告警规则

4. **合理设置超时**
   - 重试的最大延迟应小于上下文超时
   - 考虑整体请求时间预算

5. **自定义可重试错误判断**
   - 根据业务需求调整错误分类
   - 记录不可重试错误以便分析

### ❌ 避免

1. **过度重试**
   - 避免 MaxAttempts 过大（建议 ≤5）
   - 避免 MaxDelay 过长（建议 ≤30s）

2. **过于激进的熔断**
   - MaxFailures 太小可能导致频繁熔断
   - Timeout 太短可能无法恢复

3. **忽略上下文取消**
   - 确保传递正确的 context
   - 设置合理的超时时间

4. **在循环中创建包装器**
   - 应在初始化时创建一次
   - 重用包装器实例

## 示例代码

### 完整示例：RAG 服务集成

```go
package main

import (
    "context"
    "time"

    "github.com/kart-io/logger"
    "github.com/kart-io/sentinel-x/pkg/llm"
    "github.com/kart-io/sentinel-x/pkg/llm/resilience"
)

func main() {
    // 创建原始 provider
    chatProvider, err := llm.NewChatProvider("deepseek", map[string]any{
        "api_key":    "your-api-key",
        "chat_model": "deepseek-chat",
        "base_url":   "https://api.deepseek.com",
    })
    if err != nil {
        logger.Fatalw("failed to create chat provider", "error", err)
    }

    // 配置韧性层
    retryConfig := &resilience.RetryConfig{
        MaxAttempts:  3,
        InitialDelay: 500 * time.Millisecond,
        MaxDelay:     10 * time.Second,
        Multiplier:   2.0,
    }

    cbConfig := &resilience.CircuitBreakerConfig{
        MaxFailures:      5,
        Timeout:          60 * time.Second,
        HalfOpenMaxCalls: 1,
    }

    // 创建韧性 provider
    resilientProvider := resilience.NewResilientChatProvider(
        chatProvider,
        retryConfig,
        cbConfig,
    )

    // 使用韧性 provider
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    response, err := resilientProvider.Generate(
        ctx,
        "什么是向量数据库？",
        "你是一个有帮助的助手。",
    )

    if err != nil {
        logger.Errorw("LLM call failed", "error", err)

        // 检查熔断器状态
        stats := resilience.GetChatProviderStats(resilientProvider)
        if stats != nil {
            logger.Warnw("circuit breaker stats",
                "state", stats.CircuitBreakerState,
                "failures", stats.CircuitBreakerFailures,
            )
        }
        return
    }

    logger.Infow("LLM response received", "response", response)
}
```

## 故障排查

### 问题：重试次数过多导致响应慢

**解决方案**:
- 减少 `MaxAttempts`
- 减少 `MaxDelay`
- 增加上下文超时以快速失败

### 问题：熔断器频繁打开

**解决方案**:
- 增加 `MaxFailures` 容忍度
- 增加 `Timeout` 恢复时间
- 检查 LLM 服务健康状况

### 问题：某些错误不应重试但被重试了

**解决方案**:
```go
retryConfig.RetryableErrors = func(err error) bool {
    // 先使用默认判断
    if !resilience.IsRetryableError(err) {
        return false
    }

    // 添加自定义逻辑
    if strings.Contains(err.Error(), "invalid api key") {
        return false // 认证错误不重试
    }

    return true
}
```

## 参考资料

- [熔断器模式](https://martinfowler.com/bliki/CircuitBreaker.html)
- [重试模式](https://docs.microsoft.com/en-us/azure/architecture/patterns/retry)
- [指数退避](https://en.wikipedia.org/wiki/Exponential_backoff)
