package main

import (
	"fmt"
	"time"

	loggerPkg "github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== Agent连接测试 ===")
	fmt.Println("测试Agent是否能接收和转发日志")
	fmt.Println()

	// 测试gRPC连接到Agent
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4327", // Agent gRPC端口
			Protocol: "grpc",
			Timeout:  5 * time.Second,
		},
	}

	logger, err := loggerPkg.New(opt)
	if err != nil {
		fmt.Printf("❌ Agent gRPC Logger创建失败: %v\n", err)
		return
	}

	fmt.Println("✅ Agent gRPC Logger创建成功")

	testID := fmt.Sprintf("agent_test_%d", time.Now().Unix())

	// 发送简单测试
	logger.Infow("Agent gRPC测试",
		"test_id", testID,
		"agent_port", 4327,
		"protocol", "grpc",
	)

	fmt.Printf("📤 已发送到Agent gRPC，test_id: %s\n", testID)

	// 测试HTTP连接到Agent
	fmt.Println("\n--- 测试Agent HTTP ---")
	opt2 := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4328", // Agent HTTP端口
			Protocol: "http",
			Timeout:  5 * time.Second,
		},
	}

	logger2, err := loggerPkg.New(opt2)
	if err != nil {
		fmt.Printf("❌ Agent HTTP Logger创建失败: %v\n", err)
		return
	}

	fmt.Println("✅ Agent HTTP Logger创建成功")

	// 发送HTTP测试
	logger2.Infow("Agent HTTP测试",
		"test_id", testID,
		"agent_port", 4328,
		"protocol", "http",
	)

	fmt.Printf("📤 已发送到Agent HTTP，test_id: %s\n", testID)

	fmt.Println("\n等待10秒钟查看是否数据到达VictoriaLogs...")
	time.Sleep(10 * time.Second)

	fmt.Println("✅ Agent测试完成")
}
