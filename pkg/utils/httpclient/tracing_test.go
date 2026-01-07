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

// setupTracer 设置测试用的 OpenTelemetry Tracer。
func setupTracer() (trace.Tracer, *sdktrace.TracerProvider) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// 设置全局传播器（W3C Trace Context + Baggage）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Tracer("test"), tp
}

// TestInjectTraceContext_WithSpan 测试有 Span 时正确注入 traceparent 头。
func TestInjectTraceContext_WithSpan(t *testing.T) {
	tracer, tp := setupTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	client := NewClient(10*time.Second, 0)

	// 创建一个带 Span 的 Context
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// 创建请求
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req = req.WithContext(ctx)

	// 注入追踪上下文
	client.injectTraceContext(req)

	// 验证 traceparent 头已注入
	traceparent := req.Header.Get("traceparent")
	if traceparent == "" {
		t.Error("expected traceparent header to be set, got empty")
	}

	// 验证 traceparent 格式（W3C Trace Context 格式：version-trace_id-parent_id-trace_flags）
	// 示例：00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
	if len(traceparent) < 55 { // 最小长度：2 + 1 + 32 + 1 + 16 + 1 + 2 = 55
		t.Errorf("traceparent format invalid: %s", traceparent)
	}
}

// TestInjectTraceContext_WithoutSpan 测试无 Span 时不注入追踪头。
func TestInjectTraceContext_WithoutSpan(t *testing.T) {
	_, tp := setupTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	client := NewClient(10*time.Second, 0)

	// 创建没有 Span 的请求
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)

	// 注入追踪上下文
	client.injectTraceContext(req)

	// 验证 traceparent 头未注入
	traceparent := req.Header.Get("traceparent")
	if traceparent != "" {
		t.Errorf("expected no traceparent header, got: %s", traceparent)
	}
}

// TestInjectTraceContext_NilRequest 测试 nil 请求的防御性处理。
func TestInjectTraceContext_NilRequest(t *testing.T) {
	_, tp := setupTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	client := NewClient(10*time.Second, 0)

	// 不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("injectTraceContext panicked with nil request: %v", r)
		}
	}()

	client.injectTraceContext(nil)
}

// TestInjectTraceContext_NoPropagator 测试无全局传播器时的降级处理。
func TestInjectTraceContext_NoPropagator(t *testing.T) {
	// 保存原有传播器
	originalPropagator := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(originalPropagator)

	// 设置 nil 传播器
	otel.SetTextMapPropagator(nil)

	client := NewClient(10*time.Second, 0)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)

	// 不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("injectTraceContext panicked with nil propagator: %v", r)
		}
	}()

	client.injectTraceContext(req)

	// 验证未注入
	traceparent := req.Header.Get("traceparent")
	if traceparent != "" {
		t.Errorf("expected no traceparent header, got: %s", traceparent)
	}
}

// TestDoRequest_TracingIntegration 测试 DoRequest 端到端追踪集成。
func TestDoRequest_TracingIntegration(t *testing.T) {
	tracer, tp := setupTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// 创建测试服务器，捕获请求头
	var receivedTraceparent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient(10*time.Second, 0)

	// 创建带 Span 的 Context
	ctx, span := tracer.Start(context.Background(), "test-client-request")
	defer span.End()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// 执行请求（内部会自动调用 injectTraceContext）
	resp, err := client.DoRequest(req)
	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 验证 traceparent 头已传播到下游服务
	if receivedTraceparent == "" {
		t.Error("downstream service did not receive traceparent header")
	}

	// 验证格式
	if len(receivedTraceparent) < 55 {
		t.Errorf("invalid traceparent format received by downstream: %s", receivedTraceparent)
	}
}

// BenchmarkInjectTraceContext 测试追踪注入的性能开销。
func BenchmarkInjectTraceContext(b *testing.B) {
	tracer, tp := setupTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	client := NewClient(10*time.Second, 0)

	// 创建带 Span 的 Context
	ctx, span := tracer.Start(context.Background(), "benchmark-operation")
	defer span.End()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req = req.WithContext(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		client.injectTraceContext(req)
	}
}

// BenchmarkInjectTraceContext_NoSpan 测试无 Span 时的性能。
func BenchmarkInjectTraceContext_NoSpan(b *testing.B) {
	_, tp := setupTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	client := NewClient(10*time.Second, 0)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		client.injectTraceContext(req)
	}
}
