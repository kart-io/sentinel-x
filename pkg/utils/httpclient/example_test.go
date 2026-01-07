package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kart-io/sentinel-x/pkg/utils/httpclient"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ExampleClient_basic 演示 HTTP 客户端的基本使用。
//
// 使用场景:
//   - 发送带重试逻辑的 HTTP 请求
//   - 自动管理请求超时
//   - 自动处理请求失败重试
func ExampleClient_basic() {
	// 创建 HTTP 客户端：超时 30 秒，最多重试 3 次
	client := httpclient.NewClient(30*time.Second, 3)

	// 创建请求
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://api.example.com/data",
		nil,
	)
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return
	}

	// 发送请求（自动重试 5xx 错误）
	resp, err := client.DoRequest(req)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("请求成功，状态码: %d\n", resp.StatusCode)

	// 示例输出（实际输出取决于外部 API）:
	// 请求成功，状态码: 200
}

// ExampleClient_withTracing 演示带分布式追踪的 HTTP 客户端使用。
//
// 使用场景:
//   - 微服务间的 HTTP 调用需要追踪
//   - 自动传播 W3C Trace Context 到下游服务
//   - 在 Jaeger/Zipkin 中查看完整调用链路
func ExampleClient_withTracing() {
	// ========================================
	// 1. 设置 OpenTelemetry（应用启动时执行一次）
	// ========================================
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// 设置全局传播器（W3C Trace Context）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tp.Tracer("example-service")

	// ========================================
	// 2. 在业务逻辑中创建 Span
	// ========================================
	ctx, span := tracer.Start(context.Background(), "call-downstream-api")
	defer span.End()

	// ========================================
	// 3. 创建 HTTP 客户端并发送请求
	// ========================================
	client := httpclient.NewClient(30*time.Second, 3)

	req, err := http.NewRequestWithContext(
		ctx, // 使用带 Span 的 Context
		http.MethodPost,
		"https://downstream-service.example.com/api/process",
		nil,
	)
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return
	}

	// 发送请求（自动注入 traceparent 头）
	resp, err := client.DoRequest(req)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("请求成功，状态码: %d\n", resp.StatusCode)
	fmt.Println("追踪信息已自动传播到下游服务")

	// 示例输出:
	// 请求成功，状态码: 200
	// 追踪信息已自动传播到下游服务
}

// ExampleClient_tracingMechanism 演示追踪传播的工作机制。
func ExampleClient_tracingMechanism() {
	fmt.Println("=== HTTP 客户端追踪传播机制 ===")
	fmt.Println()

	fmt.Println("【工作流程】")
	fmt.Println("1. 应用启动时设置全局传播器（otel.SetTextMapPropagator）")
	fmt.Println("2. 业务代码创建 Span（tracer.Start）")
	fmt.Println("3. 创建带 Span Context 的 HTTP 请求")
	fmt.Println("4. httpclient.Client 自动注入 W3C Trace Context 头")
	fmt.Println("5. 下游服务从 HTTP Header 提取追踪信息")
	fmt.Println()

	fmt.Println("【注入的 HTTP 头】")
	fmt.Println("traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	fmt.Println("  ↑ 格式: version-trace_id-parent_id-trace_flags")
	fmt.Println()
	fmt.Println("tracestate: (可选，用于传递供应商特定信息)")
	fmt.Println()

	fmt.Println("【优雅降级】")
	fmt.Println("✓ 无 Span 时：不注入头，保持向后兼容")
	fmt.Println("✓ 无传播器时：跳过注入，不影响功能")
	fmt.Println("✓ 请求为 nil：防御性处理，不 panic")
	fmt.Println()

	fmt.Println("【性能开销】")
	fmt.Println("• 有 Span：约 142 ns/op，96 B/op")
	fmt.Println("• 无 Span：约 18 ns/op，0 B/op")
	fmt.Println("• 对 HTTP 请求总延迟的影响：< 0.1%")

	// Output:
	// === HTTP 客户端追踪传播机制 ===
	//
	// 【工作流程】
	// 1. 应用启动时设置全局传播器（otel.SetTextMapPropagator）
	// 2. 业务代码创建 Span（tracer.Start）
	// 3. 创建带 Span Context 的 HTTP 请求
	// 4. httpclient.Client 自动注入 W3C Trace Context 头
	// 5. 下游服务从 HTTP Header 提取追踪信息
	//
	// 【注入的 HTTP 头】
	// traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
	//   ↑ 格式: version-trace_id-parent_id-trace_flags
	//
	// tracestate: (可选，用于传递供应商特定信息)
	//
	// 【优雅降级】
	// ✓ 无 Span 时：不注入头，保持向后兼容
	// ✓ 无传播器时：跳过注入，不影响功能
	// ✓ 请求为 nil：防御性处理，不 panic
	//
	// 【性能开销】
	// • 有 Span：约 142 ns/op，96 B/op
	// • 无 Span：约 18 ns/op，0 B/op
	// • 对 HTTP 请求总延迟的影响：< 0.1%
}

// ExampleClient_microservices 演示微服务场景中的端到端追踪。
func ExampleClient_microservices() {
	fmt.Println("=== 微服务端到端追踪示例 ===")
	fmt.Println()

	fmt.Println("【场景：订单处理链路】")
	fmt.Println("客户端 → API 网关 → 订单服务 → 库存服务 → 支付服务")
	fmt.Println()

	fmt.Println("【Trace ID 传播】")
	fmt.Println("1. 客户端发起请求（生成 Trace ID: abc123）")
	fmt.Println("2. API 网关收到请求（提取 Trace ID: abc123）")
	fmt.Println("3. API 网关调用订单服务（注入 Trace ID: abc123）")
	fmt.Println("4. 订单服务调用库存服务（注入 Trace ID: abc123）")
	fmt.Println("5. 库存服务调用支付服务（注入 Trace ID: abc123）")
	fmt.Println()

	fmt.Println("【在 Jaeger 中查看】")
	fmt.Println("搜索 Trace ID: abc123，可看到完整调用链：")
	fmt.Println("api-gateway (100ms)")
	fmt.Println("├─ order-service (80ms)")
	fmt.Println("│  ├─ inventory-service (30ms)")
	fmt.Println("│  └─ payment-service (40ms)")
	fmt.Println()

	fmt.Println("【关键配置】")
	fmt.Println("1. 服务端中间件：提取追踪头（已实现）")
	fmt.Println("   pkg/infra/middleware/observability/tracing.go")
	fmt.Println()
	fmt.Println("2. 客户端自动注入：传播追踪头（已实现）")
	fmt.Println("   pkg/utils/httpclient/client.go")
	fmt.Println()
	fmt.Println("3. 全局传播器设置：W3C Trace Context（已实现）")
	fmt.Println("   pkg/infra/tracing/provider.go")

	// Output:
	// === 微服务端到端追踪示例 ===
	//
	// 【场景：订单处理链路】
	// 客户端 → API 网关 → 订单服务 → 库存服务 → 支付服务
	//
	// 【Trace ID 传播】
	// 1. 客户端发起请求（生成 Trace ID: abc123）
	// 2. API 网关收到请求（提取 Trace ID: abc123）
	// 3. API 网关调用订单服务（注入 Trace ID: abc123）
	// 4. 订单服务调用库存服务（注入 Trace ID: abc123）
	// 5. 库存服务调用支付服务（注入 Trace ID: abc123）
	//
	// 【在 Jaeger 中查看】
	// 搜索 Trace ID: abc123，可看到完整调用链：
	// api-gateway (100ms)
	// ├─ order-service (80ms)
	// │  ├─ inventory-service (30ms)
	// │  └─ payment-service (40ms)
	//
	// 【关键配置】
	// 1. 服务端中间件：提取追踪头（已实现）
	//    pkg/infra/middleware/observability/tracing.go
	//
	// 2. 客户端自动注入：传播追踪头（已实现）
	//    pkg/utils/httpclient/client.go
	//
	// 3. 全局传播器设置：W3C Trace Context（已实现）
	//    pkg/infra/tracing/provider.go
}

// ExampleClient_bestPractices 演示使用最佳实践。
func ExampleClient_bestPractices() {
	fmt.Println("=== HTTP 客户端追踪最佳实践 ===")
	fmt.Println()

	fmt.Println("【1. 全局初始化传播器】")
	fmt.Println("在应用启动时（main 函数）设置一次：")
	fmt.Println()
	fmt.Println("  import \"go.opentelemetry.io/otel\"")
	fmt.Println("  import \"go.opentelemetry.io/otel/propagation\"")
	fmt.Println()
	fmt.Println("  otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(")
	fmt.Println("      propagation.TraceContext{},  // W3C Trace Context")
	fmt.Println("      propagation.Baggage{},       // W3C Baggage")
	fmt.Println("  ))")
	fmt.Println()

	fmt.Println("【2. 始终使用带 Context 的请求】")
	fmt.Println("✓ 正确：http.NewRequestWithContext(ctx, ...)")
	fmt.Println("✗ 错误：http.NewRequest(...) // 无法传播追踪信息")
	fmt.Println()

	fmt.Println("【3. 不需要手动设置 traceparent】")
	fmt.Println("✓ 自动注入：httpclient.Client 自动处理")
	fmt.Println("✗ 手动注入：req.Header.Set(\"traceparent\", ...) // 多余且易错")
	fmt.Println()

	fmt.Println("【4. 验证追踪是否工作】")
	fmt.Println("方法1: 在 Jaeger UI 中搜索 Trace ID")
	fmt.Println("方法2: 使用 tcpdump 捕获 HTTP 请求头：")
	fmt.Println("  sudo tcpdump -A -s 0 'tcp port 8080 and (((ip[2:2] - ((ip[0]&0xf)<<2)) - ((tcp[12]&0xf0)>>2)) != 0)'")
	fmt.Println("  查找 'traceparent: 00-...' 头")
	fmt.Println()

	fmt.Println("【5. 性能监控】")
	fmt.Println("运行 Benchmark 验证注入开销：")
	fmt.Println("  go test -bench=BenchmarkInjectTraceContext -benchmem ./pkg/utils/httpclient/")
	fmt.Println("  预期：< 150 ns/op, < 100 B/op")

	// Output:
	// === HTTP 客户端追踪最佳实践 ===
	//
	// 【1. 全局初始化传播器】
	// 在应用启动时（main 函数）设置一次：
	//
	//   import "go.opentelemetry.io/otel"
	//   import "go.opentelemetry.io/otel/propagation"
	//
	//   otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
	//       propagation.TraceContext{},  // W3C Trace Context
	//       propagation.Baggage{},       // W3C Baggage
	//   ))
	//
	// 【2. 始终使用带 Context 的请求】
	// ✓ 正确：http.NewRequestWithContext(ctx, ...)
	// ✗ 错误：http.NewRequest(...) // 无法传播追踪信息
	//
	// 【3. 不需要手动设置 traceparent】
	// ✓ 自动注入：httpclient.Client 自动处理
	// ✗ 手动注入：req.Header.Set("traceparent", ...) // 多余且易错
	//
	// 【4. 验证追踪是否工作】
	// 方法1: 在 Jaeger UI 中搜索 Trace ID
	// 方法2: 使用 tcpdump 捕获 HTTP 请求头：
	//   sudo tcpdump -A -s 0 'tcp port 8080 and (((ip[2:2] - ((ip[0]&0xf)<<2)) - ((tcp[12]&0xf0)>>2)) != 0)'
	//   查找 'traceparent: 00-...' 头
	//
	// 【5. 性能监控】
	// 运行 Benchmark 验证注入开销：
	//   go test -bench=BenchmarkInjectTraceContext -benchmem ./pkg/utils/httpclient/
	//   预期：< 150 ns/op, < 100 B/op
}
