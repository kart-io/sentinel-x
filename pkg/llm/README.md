# LLM Provider Package

`pkg/llm` 提供了一个统一的 LLM (Large Language Model) 供应商抽象层，旨在简化不同模型供应商（如 OpenAI、Ollama 等）的集成和切换。

## 核心特性

- **统一接口**：提供标准化的 `ChatProvider` 和 `EmbeddingProvider` 接口。
- **多供应商支持**：内置支持 OpenAI, Ollama, DeepSeek, Gemini, HuggingFace, SiliconFlow 等。
- **高可用性**：内置重试机制（Exponential Backoff）和熔断器（Circuit Breaker）模式。
- **易于扩展**：通过工厂模式轻松注册和添加新的供应商。

## 快速开始

### 1. 引入包

```go
import (
    "context"
    "fmt"
    "github.com/kart-io/sentinel-x/pkg/llm"
    _ "github.com/kart-io/sentinel-x/pkg/llm/openai" // 注册 OpenAI 供应商
)
```

### 2. 初始化供应商

```go
config := map[string]any{
    "api_key":    "sk-xxxxxxxx",
    "chat_model": "gpt-4o",
    "base_url":   "https://api.openai.com/v1", // 可选
}

provider, err := llm.NewProvider("openai", config)
if err != nil {
    panic(err)
}
```

### 3. 使用 Chat API

```go
response, err := provider.Chat(context.Background(), []llm.Message{
    {Role: llm.RoleSystem, Content: "你是一个助手"},
    {Role: llm.RoleUser, Content: "你好"},
})
fmt.Println(response)
```

### 4. 使用 Embedding API

```go
vectors, err := provider.Embed(context.Background(), []string{"文本1", "文本2"})
```

## 支持的供应商与配置

### OpenAI (`openai`)

支持官方 API 及兼容 API（如 Azure OpenAI, LocalAI）。

| 配置项 | 类型 | 说明 | 默认值 |
|--------|------|------|--------|
| `api_key` | string | **必填** API 密钥 | - |
| `base_url` | string | API 基础地址 | `https://api.openai.com/v1` |
| `chat_model` | string | 对话模型 | `gpt-4o-mini` |
| `embed_model` | string | 嵌入模型 | `text-embedding-3-small` |
| `temperature` | float | 随机性 (0.0-2.0) | API 默认 |
| `max_tokens` | int | 最大生成 Token | API 默认 |
| `organization`| string | 组织 ID | - |

### Ollama (`ollama`)

本地运行的开源模型服务。

| 配置项 | 类型 | 说明 | 默认值 |
|--------|------|------|--------|
| `base_url` | string | 服务地址 | `http://localhost:11434` |
| `chat_model` | string | 对话模型 | `deepseek-r1:7b` |
| `embed_model` | string | 嵌入模型 | `nomic-embed-text` |

### DeepSeek (`deepseek`)

DeepSeek 官方 API。配置参数与 OpenAI 基本一致。

### Gemini (`gemini`)

Google Gemini API。

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `api_key` | string | **必填** API 密钥 |
| `chat_model`| string | 模型名称 |

### HuggingFace (`huggingface`)

HuggingFace Inference API。

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `api_key` | string | **必填** Access Token |
| `endpoint` | string | 模型 API 端点 |

### SiliconFlow (`siliconflow`)

硅基流动 API。配置参数与 OpenAI 基本一致。

## 韧性模式 (Resilience)

`pkg/llm/resilience` 提供了增强稳定性的工具，用于处理网络抖动和 API 不稳定性。

### 重试机制 (Retry with Backoff)

```go
import "github.com/kart-io/sentinel-x/pkg/llm/resilience"

config := resilience.DefaultRetryConfig()
config.MaxAttempts = 5

err := resilience.RetryWithBackoff(ctx, config, func() error {
    return doNetworkCall()
})
```

### 熔断器 (Circuit Breaker)

防止在服务不可用时持续请求，保护系统。

```go
cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig())

err := cb.Execute(func() error {
    return doUnstableCall()
})
```

## 接口定义

### Provider
```go
type Provider interface {
    EmbeddingProvider
    ChatProvider
}
```

### ChatProvider
```go
type ChatProvider interface {
    // Chat 进行多轮对话
    Chat(ctx context.Context, messages []Message) (string, error)
    
    // Generate 根据提示生成文本（单轮）
    Generate(ctx context.Context, prompt string, systemPrompt string) (*GenerateResponse, error)
    
    // Name 返回供应商名称
    Name() string
}
```

### EmbeddingProvider
```go
type EmbeddingProvider interface {
    // Embed 为多个文本生成向量嵌入
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    
    // EmbedSingle 为单个文本生成向量嵌入
    EmbedSingle(ctx context.Context, text string) ([]float32, error)
    
    // Name 返回供应商名称
    Name() string
}
```
