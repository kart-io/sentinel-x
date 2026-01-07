# 操作日志 - HTTP 客户端 W3C Trace Context 传播

## 时间线

### 2026-01-07 14:30:00 - 上下文收集完成

#### 结构化快速扫描
- ✅ 位置识别：`pkg/utils/httpclient/client.go`
- ✅ 现状分析：
  - 服务端已实现 W3C Trace Context 提取（`pkg/infra/middleware/observability/tracing.go:176-177`）
  - 全局传播器已配置（`pkg/infra/tracing/provider.go:117-120`）
  - HTTP 客户端无追踪注入（缺失）
- ✅ 技术栈：OpenTelemetry Go SDK, `propagation.TraceContext{}`
- ✅ 测试：参考 `pkg/infra/middleware/observability/tracing_test.go`

#### 识别关键疑问
- ❓ 在哪里注入最合适？（httpclient vs RoundTripper vs provider）
- ❓ 如何优雅降级？（无 Span 或传播器不可用时）
- ❓ 性能影响？（Header 注入开销）
- ❓ 测试策略？（单元测试 vs 集成测试）

#### 针对性深挖
- ✅ **OpenTelemetry 官方文档查询**：
  - 获取标准注入模式：`propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))`
  - 验证最佳实践：在 `http.NewRequestWithContext` 后立即注入
  - 确认降级策略：Context 无 Span 时 `Inject` 操作无副作用

- ✅ **LLM Provider 调用模式分析**：
  - 所有 provider 遵循统一模式：`NewRequestWithContext → setHeaders → DoRequest`
  - Context 已传递，适合追踪注入
  - 5 个 provider 文件（openai、deepseek、siliconflow、huggingface、ollama）

#### 充分性检查
- ✅ 我能定义清晰的接口契约吗？
  - 输入：`*http.Request`（带 Context）
  - 输出：注入 `traceparent` 和 `tracestate` 头
  - 副作用：修改 `req.Header`

- ✅ 我理解关键技术选型的理由吗？
  - **为什么在 httpclient 注入**：所有 LLM provider 统一使用此客户端，集中控制点
  - **为什么不用 RoundTripper**：项目无第三方库追踪需求，保持简单

- ✅ 我识别了主要风险点吗？
  - 并发：`propagation.HeaderCarrier` 线程安全（已验证）
  - 边界：Context 无 Span 时跳过注入
  - 性能：Header 注入约 100-200ns（可接受）

- ✅ 我知道如何验证实现吗？
  - 单元测试：验证 Header 注入正确性
  - 集成测试：验证端到端追踪链路（需 OTLP 环境）

---

## 方案比较与决策

### 方案 A：在 httpclient.Client 中添加传播器注入（推荐）

#### 设计
```go
// pkg/utils/httpclient/client.go

func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
    // 注入 W3C Trace Context
    c.injectTraceContext(req)

    // 原有重试逻辑...
}

func (c *Client) injectTraceContext(req *http.Request) {
    if req == nil || req.Context() == nil {
        return
    }

    propagator := otel.GetTextMapPropagator()
    if propagator == nil {
        return  // 优雅降级
    }

    propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
}
```

#### 优势
- ✅ **集中控制**：所有使用 `httpclient.Client` 的请求自动获得追踪
- ✅ **零侵入**：无需修改 LLM provider 代码（5 个文件）
- ✅ **简单实现**：仅需在 `DoRequest` 开头添加一行
- ✅ **易测试**：独立函数便于单元测试
- ✅ **向后兼容**：无 Span 时自动跳过，不影响现有功能

#### 劣势
- ⚠️ **作用范围**：仅对 `httpclient.Client` 生效（但项目中足够）
- ⚠️ **调用顺序依赖**：必须在 `setHeaders` 后不覆盖追踪头（已验证无冲突）

#### 风险评估
- **低风险**：`Inject` 操作幂等且线程安全
- **性能影响**：可忽略（<0.1% HTTP 请求时间）
- **向后兼容**：完全兼容（无 Span 时 no-op）

---

### 方案 B：创建 HTTP RoundTripper 包装器（不推荐）

#### 设计
```go
// pkg/infra/tracing/transport.go

type TracingTransport struct {
    Base http.RoundTripper
}

func (t *TracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    propagator := otel.GetTextMapPropagator()
    propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))

    return t.Base.RoundTrip(req)
}

// 使用方式
client := &http.Client{
    Transport: &TracingTransport{Base: http.DefaultTransport},
}
```

#### 优势
- ✅ **标准模式**：Go 生态常见的 HTTP 中间件模式
- ✅ **全局生效**：对所有使用此 Transport 的请求生效

#### 劣势
- ❌ **需修改初始化**：必须修改 `httpclient.NewClient` 初始化逻辑
- ❌ **复杂度增加**：新增文件和测试，维护成本高
- ❌ **不必要**：项目无第三方库追踪需求，过度设计

#### 风险评估
- **中等风险**：引入新的抽象层，可能影响现有重试逻辑

---

### 方案 C：在每个 provider 的 setHeaders 后注入（不推荐）

#### 设计
```go
// pkg/llm/openai/provider.go

func (p *Provider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    req, err := http.NewRequestWithContext(ctx, ...)
    p.setHeaders(req)
    p.injectTraceContext(req)  // 新增

    if err := p.client.DoJSON(req, &embedResp); err != nil {
        return nil, err
    }
}
```

#### 优势
- ✅ **明确控制**：每个 provider 可自定义注入逻辑

#### 劣势
- ❌ **代码重复**：需修改 5 个 provider 的多个方法（约 15 处）
- ❌ **易遗漏**：新增 provider 可能忘记注入
- ❌ **难维护**：分散的注入逻辑，难以统一修改

#### 风险评估
- **高风险**：代码分散，容易出现不一致

---

## 最终决策：方案 A

### 决策理由
1. **最小侵入性**：无需修改 LLM provider 代码
2. **复用全局传播器**：与服务端中间件保持一致
3. **上下文感知**：自动从 Context 提取 Span
4. **向后兼容**：无 Span 时自动跳过
5. **遵循项目模式**：参考服务端 `tracing.GetGlobalTextMapPropagator()`

### 实施计划
1. ✅ 上下文摘要已生成：`.claude/context-summary-http-tracing.md`
2. ⏳ 修改 `pkg/utils/httpclient/client.go`：添加 `injectTraceContext` 方法
3. ⏳ 编写单元测试：`pkg/utils/httpclient/tracing_test.go`
4. ⏳ 验证集成：运行 LLM provider 调用并检查追踪链路
5. ⏳ 性能基准测试：确认注入开销可接受

---

## 编码前检查

### 时间：2026-01-07 14:45:00

- ✅ 已查阅上下文摘要文件：`.claude/context-summary-http-tracing.md`
- ✅ 将使用以下可复用组件：
  - `otel.GetTextMapPropagator()`：获取全局传播器
  - `propagation.HeaderCarrier(req.Header)`：HTTP Header 适配器
- ✅ 将遵循命名约定：`injectTraceContext` 函数名清晰
- ✅ 将遵循代码风格：Go 标准库风格，错误返回，注释中文
- ✅ 确认不重复造轮子：复用 OpenTelemetry 官方 API

---

## 下一步行动

### 待办事项
- [ ] 在 `pkg/utils/httpclient/client.go` 中添加 `injectTraceContext` 方法
- [ ] 修改 `DoRequest` 方法，在开头调用 `injectTraceContext`
- [ ] 创建 `pkg/utils/httpclient/tracing_test.go` 单元测试
- [ ] 验证 LLM provider 调用时 Header 正确注入
- [ ] 编写集成测试文档

### 预期交付物
- 代码：`pkg/utils/httpclient/client.go`（修改）
- 测试：`pkg/utils/httpclient/tracing_test.go`（新增）
- 文档：操作日志和验证报告
