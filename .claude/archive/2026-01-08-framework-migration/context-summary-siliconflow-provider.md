## 项目上下文摘要（SiliconFlow 供应商实现）

生成时间：2026-01-04 11:40:00

### 1. 相似实现分析

#### 实现1: pkg/llm/openai/provider.go

- **模式**：完整 Provider 实现（同时支持 Embedding 和 Chat）
- **关键特性**：
  - 使用 `init()` 函数自动注册供应商到全局注册表（第20-22行）
  - Config 结构体包含：BaseURL、APIKey、EmbedModel、ChatModel、Timeout、MaxRetries
  - 使用工厂函数 `NewProvider(configMap map[string]any)` 从配置 map 创建实例
  - 使用 `httpclient.Client` 封装 HTTP 请求（第63行）
  - 请求/响应结构体定义（embeddingRequest、chatRequest等）
  - Bearer Token 认证方式（第297行）
- **可复用组件**：
  - `github.com/kart-io/sentinel-x/pkg/utils/httpclient` - HTTP 客户端封装
  - `github.com/kart-io/sentinel-x/pkg/utils/json` - JSON 序列化工具
- **需注意**：
  - 配置解析模式：从 map[string]any 提取并验证参数（第70-90行）
  - 必需参数验证：APIKey 为空时返回错误（第92-94行）
  - Embedding 响应按 index 排序确保顺序正确（第161-166行）

#### 实现2: pkg/llm/deepseek/provider.go

- **模式**：仅支持 Chat 的 Provider（Embedding 返回不支持错误）
- **关键特性**：
  - 与 OpenAI 类似的结构，但不支持 Embedding API（第98-106行）
  - 配置更简单：无 EmbedModel 字段
  - 同样使用 `llm.RegisterProvider` 注册（第21行）
  - API 兼容 OpenAI 格式（chat completions endpoint）
- **可复用组件**：同 OpenAI
- **需注意**：
  - DeepSeek 不支持 Embedding，明确返回错误而非 panic
  - BaseURL 默认为 `https://api.deepseek.com`（第45行）

#### 实现3: pkg/llm/resilience/resilience.go

- **模式**：弹性包装器（重试、熔断器）
- **关键特性**：
  - 提供 `RetryWithBackoff` 和 `CircuitBreaker` 机制
  - 可选集成，不是强制要求
- **可复用组件**：
  - `RetryConfig` 和 `CircuitBreakerConfig` 配置结构
  - `RetryWithCircuitBreaker` 组合函数
- **需注意**：
  - 当前 OpenAI/DeepSeek 实现中未集成 resilience
  - 可作为后续增强功能，不是核心功能

### 2. 项目约定

#### 命名约定

- **包名**：小写单词，如 `openai`、`deepseek`、`siliconflow`
- **常量**：驼峰命名，如 `ProviderName`
- **结构体**：驼峰命名，如 `Config`、`Provider`
- **接口**：定义在 `pkg/llm/provider.go`，使用 `Provider` 后缀
- **工厂函数**：`NewProvider`、`NewProviderWithConfig`

#### 文件组织

- 供应商实现位于独立子目录：`pkg/llm/{provider}/`
- 每个供应商一个 `provider.go` 文件
- 核心接口定义在 `pkg/llm/provider.go`

#### 导入顺序

```go
// 标准库
import (
    "bytes"
    "context"
    "fmt"
    "net/http"
    "time"

    // 项目库
    "github.com/kart-io/sentinel-x/pkg/llm"
    "github.com/kart-io/sentinel-x/pkg/utils/httpclient"
    "github.com/kart-io/sentinel-x/pkg/utils/json"
)
```

#### 代码风格

- 使用简体中文注释
- 结构体字段使用 `json` 和 `mapstructure` 标签
- 错误消息包含供应商前缀（如 "openai: api_key 是必需的"）
- 使用 `fmt.Errorf` 包装错误，提供上下文

### 3. 可复用组件清单

- `pkg/utils/httpclient.Client`: HTTP 客户端，支持超时和重试
  - `NewClient(timeout time.Duration, maxRetries int)`
  - `DoJSON(req *http.Request, resp interface{}) error`
- `pkg/utils/json`: JSON 序列化工具
  - `Marshal(v interface{}) ([]byte, error)`
  - `Unmarshal(data []byte, v interface{}) error`
- `pkg/llm.RegisterProvider`: 供应商注册函数
- `pkg/llm.Message`: 统一消息结构
- `pkg/llm.Role*` 常量: 角色定义（System/User/Assistant）

### 4. 测试策略

#### 测试框架

- Go 标准 testing 包
- 参考文件：`pkg/llm/provider_test.go`

#### 测试模式

- 单元测试：测试配置解析、接口实现
- Mock 测试：使用 httptest 模拟 API 响应
- 集成测试：需要真实 API Key（可能需要跳过）

#### 参考测试文件

- `pkg/llm/provider_test.go`：供应商注册和创建测试
- 建议创建 `pkg/llm/siliconflow/provider_test.go`

#### 覆盖要求

- 配置解析（正常/异常）
- 接口方法实现（Embed、EmbedSingle、Chat、Generate）
- 错误处理（网络错误、API 错误、空响应）

### 5. 依赖和集成点

#### 外部依赖

- Go 标准库：`context`、`net/http`、`time`、`bytes`
- 无第三方 HTTP 客户端依赖（使用项目内 httpclient 封装）

#### 内部依赖

- `pkg/llm`：核心接口定义和注册表
- `pkg/utils/httpclient`：HTTP 客户端
- `pkg/utils/json`：JSON 处理

#### 集成方式

- 使用 `init()` 函数自动注册到全局注册表
- 通过 `llm.NewProvider(name, config)` 创建实例
- 通过 `llm.NewEmbeddingProvider` 或 `llm.NewChatProvider` 创建专用实例

#### 配置来源

- 配置通过 `map[string]any` 传递
- 支持运行时动态配置

### 6. 技术选型理由

#### 为什么 SiliconFlow API 兼容 OpenAI 格式

- **理由**：根据搜索结果，SiliconFlow API 使用标准的 `/v1/chat/completions` 和 `/v1/embeddings` 端点
- **优势**：
  - 可复用大部分 OpenAI 实现逻辑
  - 仅需修改 BaseURL 和部分模型参数
  - 减少实现复杂度
- **劣势和风险**：
  - 可能有 SiliconFlow 特定的参数或限制
  - 需要确认响应格式完全兼容

#### API 端点

根据搜索结果：

- **Chat Completions**: `https://api.siliconflow.cn/v1/chat/completions`
- **Embeddings**: `https://api.siliconflow.cn/v1/embeddings`
- 备用域名：`https://api.siliconflow.com/v1/`

#### 默认模型选择

根据搜索结果：

- **Chat 模型**: 常见的如 `Qwen/QwQ`、`Pro/zai` 等
- **Embedding 模型**: `BAAI/bge-m3`（支持8192 tokens）
- 需要查阅官方文档确认推荐的默认模型

### 7. 关键风险点

#### 并发问题

- HTTP 客户端是否线程安全（已验证 `httpclient.Client` 基于 `http.Client`，线程安全）
- 无需额外的并发控制

#### 边界条件

- 空文本数组处理（参考 OpenAI 第135-137行）
- 空响应处理（参考 OpenAI 第245-247行）
- 配置参数缺失或无效

#### 性能瓶颈

- HTTP 超时设置（默认120秒）
- 最大重试次数（默认3次）
- 大批量 Embedding 请求可能需要分批处理

#### 安全考虑

- API Key 明文传输（通过 HTTPS）
- API Key 不应记录在日志中
- 配置验证：确保必需参数存在

### 8. SiliconFlow 特定信息

#### API 基础信息

- **Base URL**: `https://api.siliconflow.cn/v1` 或 `https://api.siliconflow.com/v1`
- **认证方式**: Bearer Token（与 OpenAI 一致）
- **支持功能**: Chat Completions、Embeddings、Audio、Rerank、Image

#### Chat Completions 关键参数

- `model`: 模型名称
- `stream`: 是否流式响应
- `max_tokens`: 最大生成 token 数
- `temperature`: 随机性控制
- `top_p`: 概率阈值
- `stop`: 停止序列（最多4个）
- `min_p`: 动态过滤阈值

#### Embeddings 关键参数

- `model`: Embedding 模型名称（如 `BAAI/bge-m3`）
- `encoding_format`: `float` 或 `base64`
- **Token 限制**:
  - `BAAI/bge-m3`: 8192 tokens
  - 其他模型: 512 tokens

#### 与 OpenAI 的差异

- 模型名称不同（SiliconFlow 特定模型）
- 可能有额外参数（如 `min_p`）
- BaseURL 不同

### 9. 实施计划（基于上下文）

#### 阶段1：基础结构（复用 OpenAI 模式）

1. 创建目录：`pkg/llm/siliconflow/`
2. 创建 `provider.go` 文件
3. 定义 `Config` 结构体（包含 BaseURL、APIKey、EmbedModel、ChatModel、Timeout、MaxRetries）
4. 实现 `NewProvider` 工厂函数
5. 实现 `Name()` 方法

#### 阶段2：实现核心接口

1. 实现 `Embed()` 方法（调用 `/v1/embeddings`）
2. 实现 `EmbedSingle()` 方法（复用 Embed）
3. 实现 `Chat()` 方法（调用 `/v1/chat/completions`）
4. 实现 `Generate()` 方法（复用 Chat）
5. 实现 `setHeaders()` 辅助方法

#### 阶段3：测试和验证

1. 创建 `provider_test.go`
2. 编写配置解析测试
3. 编写接口方法测试（使用 httptest）
4. 运行 `make test`
5. 运行 `make fmt` 和 `make lint`

#### 阶段4：集成和文档

1. 确认 `init()` 函数自动注册
2. 更新项目文档（如需要）
3. 添加使用示例（如需要）

### 10. 关键决策点

#### 决策1：是否支持 Embedding API？

- **建议**：支持（根据搜索结果，SiliconFlow 提供 Embeddings API）
- **理由**：完整实现 Provider 接口，与 OpenAI 保持一致

#### 决策2：默认模型选择

- **建议**：
  - Chat 模型：需要查询 SiliconFlow 官方文档确认推荐模型
  - Embedding 模型：`BAAI/bge-m3`（高 token 限制）
- **理由**：平衡性能和功能

#### 决策3：是否集成 resilience？

- **建议**：暂不集成，保持与现有供应商一致
- **理由**：
  - OpenAI 和 DeepSeek 实现中未使用
  - 可作为后续增强功能
  - `httpclient.Client` 已提供基础重试机制

#### 决策4：错误处理策略

- **建议**：遵循 OpenAI 模式
- **错误前缀**：`siliconflow:`
- **必需参数验证**：APIKey 缺失时返回错误
- **API 错误**：通过 `httpclient.Client.DoJSON` 处理

### 11. 验收标准

- [ ] 代码编译通过（`make build`）
- [ ] 格式化符合规范（`make fmt`）
- [ ] 静态检查通过（`make lint`）
- [ ] 单元测试覆盖核心功能
- [ ] 配置解析正确处理必需/可选参数
- [ ] 接口方法正确实现
- [ ] 错误处理符合项目约定
- [ ] 自动注册到全局供应商列表
