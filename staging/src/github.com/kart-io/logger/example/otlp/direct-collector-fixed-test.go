package main

import (
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== 修复版直连Collector测试 ===")
	fmt.Println("直接发送到Collector，跳过Agent")
	fmt.Println()

	// 直接连接到Collector HTTP端口（HTTP/Protobuf）
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4318", // 直接连接到Collector的HTTP端口
			Protocol: "http",
			Timeout:  5 * time.Second,
		},
	}

	logger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("❌ 日志器创建失败: %v\n", err)
		return
	}

	fmt.Println("✅ 修复版直连Collector日志器创建成功")

	testID := fmt.Sprintf("fixed_direct_test_%d", time.Now().Unix())

	// 发送修复版测试日志
	logger.Infow("修复版直连Collector测试日志",
		"test_id", testID,
		"timestamp", time.Now(),
		"connection", "direct_fixed",
		"endpoint", "127.0.0.1:4318",
		"message", "这是修复版直接发送到Collector的测试日志",
		"environment", "debug",
		"protocol", "http",
	)

	logger.Errorw("修复版直连Collector错误日志",
		"test_id", testID,
		"level", "error",
		"details", "测试修复版直连到Collector是否工作",
		"error_code", "FIXED_DIRECT_TEST_001",
	)

	fmt.Printf("📤 已直接发送到Collector（修复版），test_id: %s\n", testID)

	// 等待数据传输
	fmt.Println("等待5秒钟让数据传输和处理...")
	time.Sleep(5 * time.Second)

	fmt.Println("✅ 修复版直连Collector测试完成")
}
