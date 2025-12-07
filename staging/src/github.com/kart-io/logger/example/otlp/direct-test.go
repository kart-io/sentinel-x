package main

import (
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== 直接连接Collector测试 ===")

	// 直接连接到Collector (跳过Agent)
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4317", // 直接连接到Collector的gRPC端口
			Protocol: "grpc",
			Timeout:  5 * time.Second,
		},
	}

	logger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("❌ 日志器创建失败: %v\n", err)
		return
	}

	fmt.Println("✅ 直连Collector日志器创建成功")

	testID := fmt.Sprintf("direct_test_%d", time.Now().Unix())

	logger.Infow("直连Collector测试日志",
		"test_id", testID,
		"timestamp", time.Now(),
		"connection", "direct",
		"endpoint", "127.0.0.1:4317",
		"message", "这是一条直接发送到Collector的测试日志",
		"environment", "debug",
	)

	logger.Errorw("直连Collector错误日志",
		"test_id", testID,
		"level", "error",
		"details", "测试直连到Collector是否工作",
		"error_code", "DIRECT_TEST_001",
	)

	fmt.Printf("📤 已直接发送到Collector，test_id: %s\n", testID)

	// 等待数据传输
	time.Sleep(3 * time.Second)

	fmt.Println("✅ 直连Collector测试完成")
}
