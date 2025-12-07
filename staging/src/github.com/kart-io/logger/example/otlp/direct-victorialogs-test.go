package main

import (
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== 直连VictoriaLogs测试 ===")
	fmt.Println("直接发送到VictoriaLogs，跳过Agent和Collector")
	fmt.Println()

	// 直接连接到VictoriaLogs的OTLP端点
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "http://127.0.0.1:9428/insert/opentelemetry/v1/logs", // 直接连接到VictoriaLogs
			Protocol: "http",
			Timeout:  5 * time.Second,
		},
	}

	logger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("❌ 日志器创建失败: %v\n", err)
		return
	}

	fmt.Println("✅ 直连VictoriaLogs日志器创建成功")

	testID := fmt.Sprintf("direct_vl_test_%d", time.Now().Unix())

	// 发送直连VictoriaLogs测试日志
	logger.Infow("直连VictoriaLogs测试日志",
		"test_id", testID,
		"timestamp", time.Now(),
		"connection", "direct_victorialogs",
		"endpoint", "127.0.0.1:9428",
		"message", "这是直接发送到VictoriaLogs的测试日志",
		"environment", "debug",
		"protocol", "http",
	)

	logger.Errorw("直连VictoriaLogs错误日志",
		"test_id", testID,
		"level", "error",
		"details", "测试直连到VictoriaLogs是否工作",
		"error_code", "DIRECT_VL_TEST_001",
	)

	fmt.Printf("📤 已直接发送到VictoriaLogs，test_id: %s\n", testID)

	// 等待数据传输
	fmt.Println("等待5秒钟让数据传输和处理...")
	time.Sleep(5 * time.Second)

	fmt.Println("✅ 直连VictoriaLogs测试完成")
}
