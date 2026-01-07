# W3C Trace Context 客户端传播设计方案

## 文档元数据
- **创建时间**：2026-01-07
- **作者**：Claude Code
- **版本**：v1.0
- **状态**：设计完成，待实施

---

## 1. 需求背景

### 1.1 现状分析

**已完成的追踪能力**：
- ✅ **服务端追踪**：`pkg/infra/middleware/observability/tracing.go` 实现了入站请求的 W3C Trace Context 提取
- ✅ **全局传播器**：`pkg/infra/tracing/provider.go` 配置了 W3C Trace Context + Baggage 传播器
- ✅ **Span 创建**：服务端请求自动创建 Span 并记录到 OTLP

**缺失的追踪能力**：
- ❌ **客户端追踪**：出站 HTTP 请求未注入 W3C Trace Context 头
- ❌ **调用链断裂**：LLM provider 调用（OpenAI、DeepSeek 等）无法关联到上游 Span
- ❌ **端到端追踪**：无法在 Jaeger/Zipkin 中看到完整的服务间调用链

### 1.2 业务场景

**典型调用链**：
```
客户端请求 → API Server (Span 1)
              ↓
              调用 LLM Provider (Span 2 - 缺失追踪)
              ↓
              OpenAI API (外部服务)
```

**期望效果**：
```
客户端请求 → API Server (Span 1, traceparent: xxx)
              ↓
              调用 LLM Provider (Span 2, traceparent: xxx → yyy)
              ↓
              OpenAI API (收到 traceparent: yyy, 可关联上游)
```

### 1.3 技术目标

| 目标 | 描述 | 优先级 |
|------|------|--------|
| **自动注入** | 所有 HTTP 客户端请求自动注入 W3C Trace Context | P0 |
| **零侵入** | 无需修改 LLM provider 代码 | P0 |
| **向后兼容** | 无 Span 时自动跳过，不影响现有功能 | P0 |
| **性能优化** | 注入开销 < 100ns | P1 |
| **可观测性** | 支持日志记录注入失败（可选） | P2 |

---

## 2. 技术方案

### 2.1 架构设计

#### 核心思路
在 `pkg/utils/httpclient.Client.DoRequest` 方法中，执行请求前自动注入 W3C Trace Context 头。

#### 设计原则
1. **集中控制**：所有使用 `httpclient.Client` 的请求统一处理
2. **优雅降级**：传播器或 Span 不可用时静默跳过
3. **最小修改**：仅修改 `client.go`，无需改动 provider 代码
4. **标准遵循**：使用 OpenTelemetry 官方 API

#### 调用流程

**修改前**：
```
LLM Provider
  → http.NewRequestWithContext(ctx, ...)
  → provider.setHeaders(req)  // 设置业务头
  → client.DoRequest(req)
  → client.httpClient.Do(req)  // 发送请求（无追踪头）
```

**修改后**：
```
LLM Provider
  → http.NewRequestWithContext(ctx, ...)
  → provider.setHeaders(req)  // 设置业务头
  → client.DoRequest(req)
      → client.injectTraceContext(req)  // [新增] 注入追踪头
      → client.httpClient.Do(req)  // 发送请求（含追踪头）
```

### 2.2 核心实现

#### 文件：`pkg/utils/httpclient/client.go`

```go
package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/kart-io/sentinel-x/pkg/utils/json"
)

// Client is a wrapper around http.Client with additional functionality.
type Client struct {
	httpClient *http.Client
	maxRetries int
}

// NewClient creates a new HTTP client wrapper.
func NewClient(timeout time.Duration, maxRetries int) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
	}
}

// DoRequest executes an HTTP request with retry logic and automatic trace context injection.
// The caller is responsible for providing a way to reset the request body if retries are needed.
// This is a low-level method.
func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	// 注入 W3C Trace Context（如果 Context 中有 Span）
	c.injectTraceContext(req)

	var lastErr error

	// If the request has a body, we need to be able to get it again for retries
	var bodyGetter func() (io.ReadCloser, error)
	if req.Body != nil {
		// We assume the body is already read into memory if we want to support retries here
		// or the caller should handle it. For LLM providers, bodies are small.
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		_ = req.Body.Close()
		bodyGetter = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	for i := 0; i <= c.maxRetries; i++ {
		if bodyGetter != nil {
			var err error
			req.Body, err = bodyGetter()
			if err != nil {
				return nil, err
			}
		}

		resp, err := c.httpClient.Do(req)
		if err == nil {
			if resp.StatusCode < 500 {
				return resp, nil
			}
			// It's a server error, we can retry. Close the body first.
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("server error, status code %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if i < c.maxRetries {
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(time.Duration(i+1) * 500 * time.Millisecond):
				// Continue to next retry
			}
		}
	}
	return nil, lastErr
}

// DoJSON executes a JSON request, decodes the response, and ensures the body is closed.
func (c *Client) DoJSON(req *http.Request, v interface{}) error {
	resp, err := c.DoRequest(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// injectTraceContext 注入 W3C Trace Context 到 HTTP 请求头。
//
// 该方法从请求的 Context 中提取当前 Span，并使用全局传播器将追踪上下文
// 注入到 HTTP 请求头中（主要是 traceparent 和 tracestate）。
//
// 如果以下任一条件不满足，方法将静默返回：
//   - 请求对象为 nil
//   - 请求的 Context 为 nil
//   - 全局传播器未配置
//   - Context 中不包含有效的 Span
//
// 这种设计确保了在追踪未启用或配置不完整时，HTTP 请求仍能正常发送。
func (c *Client) injectTraceContext(req *http.Request) {
	// 边界检查：请求和 Context 必须存在
	if req == nil || req.Context() == nil {
		return
	}

	// 获取全局传播器
	propagator := otel.GetTextMapPropagator()
	if propagator == nil {
		// 传播器未配置，静默跳过（正常情况，表示追踪未启用）
		return
	}

	// 注入 Trace Context 到 HTTP Header
	// 如果 Context 中没有 Span，Inject 操作是 no-op（无副作用）
	propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
}
```

### 2.3 关键技术点

#### 2.3.1 传播器注入机制

**OpenTelemetry 标准流程**：
```go
// 1. 获取全局传播器（在 Provider 初始化时设置）
propagator := otel.GetTextMapPropagator()

// 2. 使用 HeaderCarrier 适配 HTTP Header
carrier := propagation.HeaderCarrier(req.Header)

// 3. 注入 Trace Context（自动处理 traceparent 和 tracestate）
propagator.Inject(ctx, carrier)
```

**生成的 HTTP 头示例**：
```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
tracestate: rojo=00f067aa0ba902b7,congo=t61rcWkgMzE
```

#### 2.3.2 优雅降级策略

| 场景 | 行为 | 理由 |
|------|------|------|
| 请求为 nil | 静默返回 | 防御性编程，避免 panic |
| Context 为 nil | 静默返回 | 不应发生（`NewRequestWithContext` 保证），但仍需检查 |
| 传播器为 nil | 静默返回 | 追踪未启用，正常场景 |
| Context 无 Span | 静默返回（Inject 自动跳过） | 请求不在追踪上下文中 |

#### 2.3.3 性能考虑

**开销分析**：
- `otel.GetTextMapPropagator()`：读取全局变量，~1ns
- `propagation.HeaderCarrier(req.Header)`：类型转换，~1ns
- `propagator.Inject(ctx, carrier)`：
  - 有 Span：提取 traceparent/tracestate 并写入 Header，~50-100ns
  - 无 Span：快速返回，~5ns

**总开销**：~60-110ns（相比 HTTP 请求的 ms 级延迟，可忽略）

---

## 3. 实现细节

### 3.1 修改点清单

| 文件路径 | 修改类型 | 描述 |
|----------|----------|------|
| `pkg/utils/httpclient/client.go` | 修改 | 添加 `injectTraceContext` 方法，修改 `DoRequest` 方法 |
| `pkg/utils/httpclient/tracing_test.go` | 新增 | 单元测试 |

### 3.2 测试计划

#### 3.2.1 单元测试

**文件**：`pkg/utils/httpclient/tracing_test.go`

**测试用例**：

```go
package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TestInjectTraceContext_WithSpan 测试 Context 中有 Span 时的注入行为
func TestInjectTraceContext_WithSpan(t *testing.T) {
	// 1. 配置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 2. 创建 TracerProvider 和 Span
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// 3. 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 4. 执行注入
	client := NewClient(10*time.Second, 3)
	client.injectTraceContext(req)

	// 5. 验证 traceparent 头存在
	traceparent := req.Header.Get("traceparent")
	if traceparent == "" {
		t.Error("traceparent 头未注入")
	}

	// 6. 验证 traceparent 格式（00-{trace-id}-{span-id}-{flags}）
	if len(traceparent) < 55 {
		t.Errorf("traceparent 格式错误: %s", traceparent)
	}
}

// TestInjectTraceContext_WithoutSpan 测试 Context 中无 Span 时的降级行为
func TestInjectTraceContext_WithoutSpan(t *testing.T) {
	// 1. 配置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	))

	// 2. 创建不含 Span 的 Context
	ctx := context.Background()

	// 3. 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 4. 执行注入
	client := NewClient(10*time.Second, 3)
	client.injectTraceContext(req)

	// 5. 验证 traceparent 头不存在（正常，无 Span）
	traceparent := req.Header.Get("traceparent")
	if traceparent != "" {
		t.Errorf("无 Span 时不应注入 traceparent: %s", traceparent)
	}
}

// TestInjectTraceContext_NilRequest 测试 nil 请求的防御性处理
func TestInjectTraceContext_NilRequest(t *testing.T) {
	client := NewClient(10*time.Second, 3)

	// 不应 panic
	client.injectTraceContext(nil)
}

// TestInjectTraceContext_NoPropagator 测试传播器未配置时的降级行为
func TestInjectTraceContext_NoPropagator(t *testing.T) {
	// 1. 清空全局传播器
	otel.SetTextMapPropagator(nil)

	// 2. 创建带 Span 的 Context
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// 3. 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 4. 执行注入（应静默跳过）
	client := NewClient(10*time.Second, 3)
	client.injectTraceContext(req)

	// 5. 验证 traceparent 头不存在
	traceparent := req.Header.Get("traceparent")
	if traceparent != "" {
		t.Errorf("传播器未配置时不应注入 traceparent: %s", traceparent)
	}
}

// TestDoRequest_Integration 测试 DoRequest 的集成行为
func TestDoRequest_Integration(t *testing.T) {
	// 1. 配置全局传播器和 TracerProvider
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	))
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// 2. 创建模拟 HTTP 服务器
	receivedHeaders := make(http.Header)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 记录接收到的 Header
		for k, v := range r.Header {
			receivedHeaders[k] = v
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 3. 创建带 Span 的 Context
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-request")
	defer span.End()

	// 4. 发送 HTTP 请求
	client := NewClient(10*time.Second, 3)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 5. 验证服务器收到 traceparent 头
	traceparent := receivedHeaders.Get("traceparent")
	if traceparent == "" {
		t.Error("服务器未收到 traceparent 头")
	}
	t.Logf("服务器收到的 traceparent: %s", traceparent)
}
```

#### 3.2.2 集成测试

**场景**：LLM Provider 调用

**验证步骤**：
1. 启动 OTLP Collector（使用 `otlp-docker/`）
2. 启动 API Server（启用追踪）
3. 发送请求触发 LLM provider 调用
4. 在 Jaeger UI 中验证调用链：
   - API Server Span（父 Span）
   - LLM Provider HTTP 请求 Span（子 Span，关联正确）

---

## 4. 验收标准

### 4.1 功能验收

| 验收项 | 验收标准 | 验收方法 |
|--------|----------|----------|
| **自动注入** | 所有 `httpclient.Client` 发起的请求自动注入 traceparent 头 | 单元测试 + 网络抓包 |
| **零侵入** | LLM provider 代码无需修改 | 代码审查 |
| **向后兼容** | 无 Span 时不注入，不影响现有功能 | 单元测试（无 Span 场景） |
| **传播器复用** | 使用全局传播器，与服务端一致 | 代码审查 |

### 4.2 性能验收

| 验收项 | 验收标准 | 验收方法 |
|--------|----------|----------|
| **注入开销** | < 100ns | Benchmark 测试 |
| **内存分配** | 0 次额外分配 | Benchmark 测试（-benchmem） |
| **HTTP 延迟** | < 0.1% 增加 | 压测对比 |

### 4.3 测试验收

| 验收项 | 验收标准 | 验收方法 |
|--------|----------|----------|
| **单元测试** | 覆盖所有边界条件 | `go test -cover` |
| **集成测试** | 验证端到端追踪链路 | Jaeger UI 验证 |
| **Race 检测** | 无数据竞争 | `go test -race` |

---

## 5. 部署与迁移

### 5.1 部署步骤

1. **合并代码**：
   ```bash
   git add pkg/utils/httpclient/client.go
   git add pkg/utils/httpclient/tracing_test.go
   git commit -m "feat(httpclient): 实现 W3C Trace Context 客户端传播"
   ```

2. **运行测试**：
   ```bash
   go test -v ./pkg/utils/httpclient/
   go test -race ./pkg/utils/httpclient/
   ```

3. **部署到生产**：
   - 无需配置变更
   - 无需重启服务（滚动更新即可）

### 5.2 回滚计划

**触发条件**：
- 单元测试失败
- 集成测试显示追踪链路错误
- 性能测试显示显著延迟增加

**回滚步骤**：
1. 还原 `pkg/utils/httpclient/client.go` 到修改前版本
2. 删除 `pkg/utils/httpclient/tracing_test.go`
3. 重新运行测试验证回滚成功

---

## 6. 风险评估

### 6.1 技术风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| **Header 覆盖** | LLM provider 的 `setHeaders` 覆盖追踪头 | 低 | 已验证无冲突（业务头与追踪头不重叠） |
| **性能下降** | HTTP 请求延迟增加 | 极低 | Benchmark 验证，开销 < 0.1% |
| **并发问题** | Header 并发读写 | 极低 | `propagation.HeaderCarrier` 线程安全 |

### 6.2 业务风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| **外部服务不支持** | 外部 API 拒绝带追踪头的请求 | 极低 | 追踪头是标准 HTTP 头，不影响 API 兼容性 |
| **隐私泄露** | Trace ID 泄露敏感信息 | 无 | Trace ID 是随机 UUID，不包含业务数据 |

---

## 7. 未来扩展

### 7.1 可选功能

| 功能 | 描述 | 优先级 |
|------|------|--------|
| **注入日志** | 记录注入成功/失败的日志 | P3 |
| **选择性注入** | 支持配置黑名单跳过某些域名 | P4 |
| **自定义传播器** | 支持每个 Client 使用不同传播器 | P4 |

### 7.2 长期优化

- **性能监控**：采集注入耗时指标，优化热路径
- **Span 丰富**：自动为 HTTP 请求创建子 Span（需修改 provider 代码）
- **错误追踪**：关联 HTTP 错误到 Span 事件

---

## 8. 参考资料

### 8.1 规范文档

- [W3C Trace Context 规范](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry Go SDK 文档](https://opentelemetry.io/docs/languages/go/)
- [Context7 OpenTelemetry 示例](https://context7.com/open-telemetry/opentelemetry-go/)

### 8.2 相关代码

- **服务端提取**：`pkg/infra/middleware/observability/tracing.go:176-177`
- **传播器配置**：`pkg/infra/tracing/provider.go:117-120`
- **LLM Provider**：`pkg/llm/openai/provider.go`、`pkg/llm/deepseek/provider.go` 等

---

## 9. 总结

### 9.1 核心优势

- ✅ **最小侵入**：仅修改 1 个文件，无需改动 5 个 LLM provider
- ✅ **标准遵循**：使用 OpenTelemetry 官方 API，符合 W3C 规范
- ✅ **向后兼容**：无 Span 时自动降级，不影响现有功能
- ✅ **性能优化**：开销可忽略（< 100ns）
- ✅ **易于测试**：独立函数，边界清晰

### 9.2 设计亮点

1. **集中控制点**：在 `httpclient.Client` 统一处理，避免代码分散
2. **优雅降级**：多层检查（nil 请求、nil Context、nil 传播器），确保健壮性
3. **标准化实现**：复用全局传播器，与服务端中间件保持一致
4. **零配置**：无需额外配置，自动启用追踪

### 9.3 预期效果

实施后，系统将具备完整的端到端追踪能力：
- 入站请求：提取上游 Trace Context（已有）
- 出站请求：注入本地 Trace Context（本次新增）
- 调用链路：在 Jaeger/Zipkin 中可视化完整的服务间调用链
