package main

import (
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== 简化OTLP测试 ===")
	fmt.Println("测试修复的字段映射和数据格式")
	fmt.Println()

	// 简化的OTLP配置
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4327", // Agent
			Protocol: "grpc",
			Timeout:  5 * time.Second,
		},
	}

	logger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("❌ Logger创建失败: %v\n", err)
		return
	}

	fmt.Println("✅ Logger创建成功，开始发送简化测试日志...")

	testID := fmt.Sprintf("simple_test_%d", time.Now().Unix())

	// 发送简单的INFO日志
	logger.Infow("简化测试消息",
		"test_id", testID,
		"test_type", "simple",
		"timestamp", time.Now(),
		"status", "success",
	)

	// 发送ERROR日志测试不同级别
	logger.Errorw("简化错误测试",
		"test_id", testID,
		"test_type", "simple_error",
		"error_code", "SIMPLE_001",
	)

	fmt.Printf("📤 已发送简化测试日志，test_id: %s\n", testID)

	// 等待传输
	fmt.Println("等待3秒钟让日志传输...")
	time.Sleep(3 * time.Second)

	fmt.Println("✅ 简化测试完成")
}
