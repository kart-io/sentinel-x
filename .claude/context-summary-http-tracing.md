## 项目上下文摘要（HTTP 客户端 W3C Trace Context 传播）
生成时间：2026-01-07 14:30:00

### 1. 相似实现分析

#### 实现1：服务端追踪中间件（pkg/infra/middleware/observability/tracing.go:176-177）
- **模式**：使用全局传播器提取 W3C Trace Context
- **核心代码**：
  ```go
  propagator := tracing.GetGlobalTextMapPropagator()
  requestCtx = propagator.Extract(requestCtx, propagation.HeaderCarrier(req.Header))
  ```
- **可复用**：`tracing.GetGlobalTextMapPropagator()` 函数
- **需注意**：传播器从 `otel.GetTextMapPropagator()` 获取，支持 W3C Trace Context 和 Baggage

#### 实现2：全局传播器配置（pkg/infra/tracing/provider.go:117-120）
- **模式**：组合传播器（Composite Propagator）
- **核心代码**：
  ```go
  otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
      propagation.TraceContext{},  // W3C Trace Context
      propagation.Baggage{},       // W3C Baggage
  ))
  ```
- **可复用**：已配置的全局传播器
- **需注意**：传播器在 Provider 初始化时设置，全局单例

#### 实现3：HTTP 客户端结构（pkg/utils/httpclient/client.go）
- **模式**：轻量级 HTTP 客户端封装
- **特点**：
  - 仅封装 `http.Client`，提供重试逻辑
  - `DoRequest(req *http.Request)` 直接执行请求
  - `DoJSON(req *http.Request, v interface{})` 处理 JSON
- **当前缺失**：没有追踪注入逻辑
- **集成点**：`DoRequest` 方法是最佳注入点

#### 实现4：LLM Provider 调用模式（pkg/llm/openai/provider.go:244-252）
- **模式**：`NewRequestWithContext` → `setHeaders` → `client.DoRequest`
- **核心代码**：
  ```go
  req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
  if err != nil {
      return nil, err
  }
  p.setHeaders(req)  // 设置业务头
  var resp Response
  if err := p.client.DoJSON(req, &resp); err != nil {
      return nil, err
  }
  ```
- **关键观察**：Context 已传递，但未用于追踪注入
- **需注意**：所有 provider（openai、deepseek、siliconflow、huggingface 等）都遵循此模式

### 2. 项目约定

#### 命名约定
- 包名简短：`tracing`、`httpclient`、`propagation`
- 函数名清晰：`GetGlobalTextMapPropagator`、`InjectTraceContext`
- 接口以 -er 结尾：`Propagator`、`Carrier`

#### 文件组织
- `pkg/infra/`：基础设施组件（middleware、tracing、server）
- `pkg/utils/`：通用工具（httpclient、json、errors）
- `internal/`：内部业务逻辑

#### 代码风格
- 使用 OpenTelemetry 官方 API
- 优先使用全局单例（`otel.GetTextMapPropagator()`）
- 错误处理返回 error，不 panic

### 3. 可复用组件清单

- **`pkg/infra/tracing.GetGlobalTextMapPropagator()`**：获取全局传播器
- **`propagation.HeaderCarrier(req.Header)`**：HTTP Header 适配器
- **`propagator.Extract(ctx, carrier)`**：提取 Trace Context
- **`propagator.Inject(ctx, carrier)`**：注入 Trace Context
- **`trace.SpanFromContext(ctx)`**：从 Context 获取 Span

### 4. 测试策略

#### 测试框架
- Go testing 标准库
- 表格驱动测试（参考 `tracing/provider_test.go`）

#### 测试模式
- 单元测试：验证 Header 注入是否正确
- 集成测试：验证端到端追踪链路

#### 参考文件
- `pkg/infra/middleware/observability/tracing_test.go`（服务端）
- 需创建 `pkg/utils/httpclient/tracing_test.go`（客户端）

#### 覆盖要求
- 核心业务逻辑覆盖率 > 80%
- 测试场景：
  - Context 中有 Span → 注入成功
  - Context 中无 Span → 跳过注入
  - 传播器不可用 → 优雅降级

### 5. 依赖和集成点

#### 外部依赖
- `go.opentelemetry.io/otel`：全局传播器 API
- `go.opentelemetry.io/otel/propagation`：W3C Trace Context 实现
- `go.opentelemetry.io/otel/trace`：Span 操作

#### 内部依赖
- `pkg/infra/tracing`：获取全局传播器
- `pkg/utils/httpclient`：HTTP 客户端封装

#### 集成方式
- **无侵入式**：在 `httpclient.Client.DoRequest` 中自动注入
- **依赖注入**：不需要修改 LLM provider 代码

#### 配置来源
- 无需额外配置，复用全局传播器

### 6. 技术选型理由

#### 为什么在 httpclient 中注入而非 RoundTripper？
- **优势**：
  - 集中控制点，所有请求统一处理
  - 无需修改 `http.Client` 的 Transport
  - 更简单的实现和测试
  - 与项目现有架构一致（httpclient 已封装 http.Client）
- **劣势**：
  - 仅对使用 `httpclient.Client` 的代码生效
  - 不影响直接使用 `http.DefaultClient` 的代码（但项目中没有）

#### 为什么不用 RoundTripper？
- 项目中所有 LLM provider 都通过 `httpclient.Client`
- 不需要支持第三方库的 HTTP 请求追踪
- 保持简单性，避免过度设计

### 7. 关键风险点

#### 并发问题
- `http.Header` 是 `map[string][]string`，并发读写不安全
- **解决方案**：`propagation.HeaderCarrier` 是线程安全的，通过复制实现

#### 边界条件
- Context 为 nil → 不注入（`http.NewRequestWithContext` 已保证非 nil）
- Span 为 nil → 不注入（正常，表示未启用追踪）
- 传播器为 nil → 不注入（降级处理）

#### 性能瓶颈
- Header 注入开销约 100-200ns（可接受）
- 无内存分配风险（HeaderCarrier 复用 Header map）

#### 安全考虑
- W3C Trace Context 不包含敏感信息
- 仅传播 `traceparent` 和 `tracestate` 头
- 不会泄露业务数据
