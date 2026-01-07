# HTTP 客户端追踪集成文档

## 概述

实现了 HTTP 客户端的 W3C Trace Context 自动传播功能，确保微服务间调用的端到端追踪链路完整。

## 核心变更

### 修改的文件

#### 1. `pkg/utils/httpclient/client.go`
**变更内容**：
- 添加 OpenTelemetry 依赖导入
- 在 `DoRequest` 方法开头自动调用 `injectTraceContext`
- 新增 `injectTraceContext` 方法实现追踪上下文注入

**关键代码**：
```go
func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	// 自动注入 W3C Trace Context 头
	c.injectTraceContext(req)

	// 原有重试逻辑...
}

func (c *Client) injectTraceContext(req *http.Request) {
	if req == nil || req.Context() == nil {
		return
	}

	propagator := otel.GetTextMapPropagator()
	if propagator == nil {
		return
	}

	propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
}
```

**影响范围**：
所有使用 `httpclient.Client` 的调用点（约 15 个），包括：
- `pkg/llm/openai/provider.go`
- `pkg/llm/deepseek/provider.go`
- `pkg/llm/siliconflow/provider.go`
- `pkg/llm/huggingface/provider.go`
- `pkg/llm/ollama/provider.go`

### 新增的文件

#### 2. `pkg/utils/httpclient/tracing_test.go`
**测试覆盖**：
- ✅ `TestInjectTraceContext_WithSpan`：验证有 Span 时正确注入 traceparent 头
- ✅ `TestInjectTraceContext_WithoutSpan`：验证无 Span 时不注入
- ✅ `TestInjectTraceContext_NilRequest`：验证 nil 请求的防御性处理
- ✅ `TestInjectTraceContext_NoPropagator`：验证无传播器时的降级
- ✅ `TestDoRequest_TracingIntegration`：端到端集成测试
- ✅ `BenchmarkInjectTraceContext`：性能基准测试（有 Span）
- ✅ `BenchmarkInjectTraceContext_NoSpan`：性能基准测试（无 Span）

**测试结果**：
```
PASS: TestInjectTraceContext_WithSpan (0.00s)
PASS: TestInjectTraceContext_WithoutSpan (0.00s)
PASS: TestInjectTraceContext_NilRequest (0.00s)
PASS: TestInjectTraceContext_NoPropagator (0.00s)
PASS: TestDoRequest_TracingIntegration (0.00s)
```

**性能数据**：
```
BenchmarkInjectTraceContext-28           7874162    142.5 ns/op    96 B/op    3 allocs/op
BenchmarkInjectTraceContext_NoSpan-28   66177189     18.10 ns/op     0 B/op    0 allocs/op
```

#### 3. `pkg/utils/httpclient/example_test.go`
**示例文档**：
- `ExampleClient_basic`：基本使用示例
- `ExampleClient_withTracing`：带追踪的使用示例
- `ExampleClient_tracingMechanism`：追踪机制说明
- `ExampleClient_microservices`：微服务场景示例
- `ExampleClient_bestPractices`：最佳实践指南

## 工作原理

### 追踪链路流程

```
1. 客户端请求
   ↓ (携带 traceparent: A)
2. API Gateway (提取 → Span 1, traceparent: A)
   ↓ 调用下游服务
3. httpclient.Client (注入 → traceparent: A→B)
   ↓
4. 下游服务 (提取 → Span 2, traceparent: A→B)
   ↓ 继续调用
5. httpclient.Client (注入 → traceparent: A→B→C)
   ↓
6. 第三方服务 (收到 traceparent: C)
```

### 注入的 HTTP 头

```http
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
             ↑  ↑                               ↑                ↑
          版本  Trace ID (128 bit)              Parent ID (64 bit) Flags
```

## 集成方式

### 无需代码变更

所有使用 `httpclient.Client` 的代码**无需任何修改**，自动获得追踪传播能力：

```go
// 原有代码
client := httpclient.NewClient(30*time.Second, 3)
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, _ := client.DoRequest(req)

// 追踪头已自动注入，无需额外代码
```

### 全局配置（已存在）

项目已在 `pkg/infra/tracing/provider.go` 中配置全局传播器：

```go
// Line 117-120
otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
    propagation.TraceContext{},  // W3C Trace Context
    propagation.Baggage{},       // W3C Baggage
))
```

## 优雅降级

### 无 Span 场景

```go
// 无 Span 的请求不会注入追踪头
req, _ := http.NewRequest("GET", url, nil)  // 无 Context
client.DoRequest(req)  // 正常执行，不注入
```

### 无传播器场景

```go
// 如果全局传播器未设置
otel.SetTextMapPropagator(nil)
client.DoRequest(req)  // 跳过注入，不影响功能
```

### 防御性处理

```go
// nil 请求不会 panic
client.injectTraceContext(nil)  // 安全返回
```

## 性能影响

### 注入开销

| 场景 | 延迟 (ns/op) | 内存 (B/op) | 分配次数 |
|------|--------------|-------------|----------|
| **有 Span** | 142.5 | 96 | 3 |
| **无 Span** | 18.10 | 0 | 0 |

### HTTP 请求总延迟影响

假设 HTTP 请求平均延迟 100ms：
- 注入开销：142.5 ns = 0.0001425 ms
- 占比：0.0001425 / 100 ≈ **0.0001%**（可忽略）

## 验证方法

### 1. 单元测试

```bash
go test -v ./pkg/utils/httpclient/ -run=TestInjectTraceContext
```

### 2. 端到端测试

```bash
go test -v ./pkg/utils/httpclient/ -run=TestDoRequest_TracingIntegration
```

### 3. 性能验证

```bash
go test -bench=BenchmarkInjectTraceContext -benchmem ./pkg/utils/httpclient/
```

### 4. 实际环境验证

#### 方法 A：Jaeger UI
1. 启动 Jaeger：`docker run -d -p16686:16686 -p4318:4318 jaegertracing/all-in-one:latest`
2. 配置应用导出到 Jaeger（已支持）
3. 访问 http://localhost:16686
4. 搜索 Service 或 Trace ID

#### 方法 B：tcpdump 抓包
```bash
sudo tcpdump -A -s 0 'tcp port 8080 and (((ip[2:2] - ((ip[0]&0xf)<<2)) - ((tcp[12]&0xf0)>>2)) != 0)' | grep -A 5 "traceparent"
```

查找输出中的：
```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

## 与现有系统集成

### 服务端中间件（已实现）

`pkg/infra/middleware/observability/tracing.go:176-177`：
```go
// 提取入站请求的追踪上下文
requestCtx = propagator.Extract(requestCtx, propagation.HeaderCarrier(req.Header))
```

### 客户端注入（本次新增）

`pkg/utils/httpclient/client.go:37`：
```go
// 注入出站请求的追踪上下文
c.injectTraceContext(req)
```

### 全局传播器（已实现）

`pkg/infra/tracing/provider.go:117-120`：
```go
// W3C Trace Context + Baggage
otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
    propagation.TraceContext{},
    propagation.Baggage{},
))
```

## 兼容性

### 向后兼容

- ✅ 无 Span 时不注入头，保持原有行为
- ✅ 无传播器时跳过注入，不影响功能
- ✅ 所有现有测试继续通过

### W3C 标准遵循

- ✅ 完全符合 W3C Trace Context 规范
- ✅ 支持 traceparent 和 tracestate 头
- ✅ 兼容所有支持 W3C 标准的追踪系统

### 外部 API 兼容性

- ✅ 追踪头是标准 HTTP 头，不影响 API 请求
- ✅ 外部服务可以忽略追踪头（向后兼容）
- ✅ 支持与第三方服务（OpenAI, Hugging Face 等）集成

## 故障排查

### 问题：追踪头未传播

**检查清单**：
1. 确认全局传播器已设置：`otel.GetTextMapPropagator() != nil`
2. 确认请求使用了带 Span 的 Context：`http.NewRequestWithContext(ctx, ...)`
3. 确认 Context 中有活跃 Span：`trace.SpanFromContext(ctx).IsRecording()`
4. 使用 tcpdump 验证实际 HTTP 头

### 问题：追踪链路断裂

**检查清单**：
1. 确认下游服务实现了服务端提取：`propagator.Extract(...)`
2. 确认下游服务使用了相同的传播器：`propagation.TraceContext{}`
3. 在 Jaeger UI 中检查每个服务的 Span

### 问题：性能下降

**检查清单**：
1. 运行 Benchmark 确认注入开销：`< 150 ns/op`
2. 检查是否误用了同步导出器：应使用 BatchSpanProcessor
3. 使用 pprof 分析实际瓶颈

## 最佳实践

### 1. 始终使用带 Context 的请求

```go
// ✓ 正确
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

// ✗ 错误（无法传播追踪）
req, _ := http.NewRequest("GET", url, nil)
```

### 2. 不手动设置 traceparent

```go
// ✓ 正确（自动注入）
client.DoRequest(req)

// ✗ 错误（多余且易错）
req.Header.Set("traceparent", "00-...")
client.DoRequest(req)
```

### 3. 在 main 函数初始化传播器

```go
func main() {
    // 设置全局传播器（仅一次）
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    // 应用逻辑...
}
```

## 未来改进

### 可选增强

1. **支持自定义传播器**：允许配置不同的传播格式（如 B3, Jaeger）
2. **追踪采样控制**：在客户端级别控制追踪采样率
3. **自动重试追踪**：记录重试次数和原因到 Span 属性
4. **请求元数据**：自动记录 HTTP 方法、URL、状态码到 Span

### 性能优化

1. **池化 HeaderCarrier**：减少内存分配
2. **延迟注入**：仅在首次发送时注入（减少重试开销）

## 参考资源

- **W3C Trace Context 规范**：https://www.w3.org/TR/trace-context/
- **OpenTelemetry Go SDK**：https://github.com/open-telemetry/opentelemetry-go
- **项目服务端中间件**：`pkg/infra/middleware/observability/tracing.go`
- **项目追踪提供者**：`pkg/infra/tracing/provider.go`

## 总结

本次改进实现了 HTTP 客户端的 W3C Trace Context 自动传播，具备以下特点：

- ✅ **零侵入**：无需修改业务代码
- ✅ **高性能**：注入开销 < 150 ns，可忽略
- ✅ **优雅降级**：无 Span 时自动跳过
- ✅ **标准遵循**：完全符合 W3C 规范
- ✅ **充分测试**：单元测试 + 集成测试 + 性能测试

通过此次改进，项目已具备完整的端到端分布式追踪能力，支持在 Jaeger、Zipkin 等可观测性平台中查看完整的微服务调用链路。
