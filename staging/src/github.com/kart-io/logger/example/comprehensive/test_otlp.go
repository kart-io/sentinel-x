package main

import (
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func testOTLPIntegration() {
	fmt.Println("=== OTLP Integration Test ===")

	// 测试 OTLP gRPC 直连收集器
	grpcOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(true),    // 启用 OTLP 发送
			Endpoint: "127.0.0.1:4317", // gRPC 不需要 http:// 前缀
			Protocol: "grpc",
			Timeout:  10 * time.Second,
			Headers: map[string]string{
				"service.name":    "logger-test",
				"service.version": "1.0.0",
				"environment":     "test",
				"test.name":       "otlp-integration",
			},
		},
	}

	grpcLogger, err := logger.New(grpcOpt)
	if err != nil {
		fmt.Printf("Failed to create gRPC logger: %v\n", err)
		return
	}

	fmt.Println("Testing OTLP gRPC integration...")

	// 发送多条不同级别的日志
	grpcLogger.Infow("OTLP integration test started",
		"test_id", "otlp-grpc-001",
		"timestamp", time.Now().Unix(),
		"protocol", "grpc",
		"endpoint", "127.0.0.1:4317",
	)

	grpcLogger.Warnw("OTLP test warning message",
		"test_id", "otlp-grpc-002",
		"warning_type", "integration_test",
		"should_appear_in_vlogs", true,
	)

	grpcLogger.Errorw("OTLP test error message",
		"test_id", "otlp-grpc-003",
		"error_type", "simulated_error",
		"trace_id", "test-trace-123",
		"span_id", "test-span-456",
	)

	// 测试 HTTP 协议到 Agent
	fmt.Println("\nTesting OTLP HTTP to Agent...")
	httpOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "DEBUG",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(true),                   // 启用 OTLP 发送
			Endpoint: "http://127.0.0.1:4328/v1/logs", // Agent HTTP 端点
			Protocol: "http/protobuf",
			Timeout:  5 * time.Second,
			Headers: map[string]string{
				"service.name":    "logger-test-http",
				"service.version": "1.0.0",
				"environment":     "test",
				"protocol":        "http",
			},
		},
	}

	httpLogger, err := logger.New(httpOpt)
	if err != nil {
		fmt.Printf("Failed to create HTTP logger: %v\n", err)
		return
	}

	httpLogger.Debugw("OTLP HTTP test debug message",
		"test_id", "otlp-http-001",
		"protocol", "http/protobuf",
		"via", "otel-agent",
	)

	httpLogger.Infow("OTLP HTTP test info message",
		"test_id", "otlp-http-002",
		"message_level", "info",
		"expected_in_vlogs", true,
	)

	// 等待消息发送完成
	fmt.Println("\nWaiting for OTLP messages to be sent...")
	time.Sleep(3 * time.Second)

	fmt.Println("OTLP integration test completed!")
	fmt.Println("Check VictoriaLogs at http://127.0.0.1:9428 for logs")
	fmt.Println("Search for service.name=\"logger-test\" or test_id=\"otlp-*\"")
}

// Helper function to create boolean pointers
func boolPtr(b bool) *bool {
	return &b
}

func main() {
	testOTLPIntegration()
}
