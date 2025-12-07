package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== 简单日志轮转示例 ===")

	// 1. 确保日志目录存在
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}
	fmt.Printf("日志目录已创建: %s\n", logDir)

	// 2. 使用标准 kart-io/logger，但输出到 stdout（不会出错）
	fmt.Println("\n--- 使用标准输出的 kart-io/logger ---")

	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"}, // 使用标准输出避免文件问题
	}

	stdLogger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("创建标准 logger 失败: %v\n", err)
		return
	}

	// 写入一些日志到标准输出
	stdLogger.Info("应用程序启动")
	stdLogger.Infow("用户操作",
		"user_id", "12345",
		"action", "login",
		"ip", "192.168.1.100",
		"timestamp", time.Now().Unix())
	stdLogger.Warn("这是一条警告消息")
	stdLogger.Errorw("模拟错误",
		"error_code", "E001",
		"error_message", "数据库连接超时")

	// 3. 演示直接使用 lumberjack 进行轮转写入
	fmt.Println("\n--- 使用 lumberjack 直接写入 ---")

	rotateWriter := &lumberjack.Logger{
		Filename:   "./logs/direct.log",
		MaxSize:    1,     // 1MB（小值便于测试轮转）
		MaxBackups: 3,     // 保留 3 个备份
		MaxAge:     7,     // 7 天后删除
		Compress:   false, // 不压缩便于查看
		LocalTime:  true,
	}

	// 写入足够多的日志来触发轮转
	fmt.Println("写入日志以测试轮转...")
	for i := 0; i < 100; i++ {
		logLine := fmt.Sprintf(`{"timestamp":"%s","level":"info","message":"测试轮转消息 %d","iteration":%d,"data":"这是一条用于测试日志轮转功能的消息，包含一些额外数据以增加文件大小"}%s`,
			time.Now().Format(time.RFC3339),
			i,
			i,
			"\n")

		if _, err := rotateWriter.Write([]byte(logLine)); err != nil {
			fmt.Printf("写入失败: %v\n", err)
			break
		}

		if i%20 == 0 {
			fmt.Printf("已写入 %d 条日志\n", i+1)
		}
	}

	// 刷新并关闭
	rotateWriter.Close()

	// 4. 检查生成的文件
	fmt.Println("\n--- 检查生成的日志文件 ---")
	files, err := os.ReadDir(logDir)
	if err != nil {
		fmt.Printf("读取目录失败: %v\n", err)
		return
	}

	totalSize := int64(0)
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		totalSize += info.Size()
		fmt.Printf("  📄 %s (大小: %d 字节, 修改时间: %s)\n",
			file.Name(),
			info.Size(),
			info.ModTime().Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\n总计文件数: %d\n", len(files))
	fmt.Printf("总计大小: %d 字节 (%.2f KB)\n", totalSize, float64(totalSize)/1024)

	// 5. 显示其中一个日志文件的内容示例
	if len(files) > 0 {
		fmt.Printf("\n--- %s 文件内容示例（前5行）---\n", files[0].Name())
		filePath := fmt.Sprintf("%s/%s", logDir, files[0].Name())
		if content, err := os.ReadFile(filePath); err == nil {
			lines := 0
			for _, b := range content {
				if b == '\n' {
					lines++
					if lines >= 5 {
						fmt.Printf("... (文件还有更多内容)\n")
						break
					}
				}
				if lines < 5 {
					fmt.Printf("%c", b)
				}
			}
		}
	}

	fmt.Println("\n✅ 简单轮转示例完成！")
	fmt.Println("💡 提示：")
	fmt.Println("   - 查看 ./logs 目录中的文件")
	fmt.Println("   - 如果文件够大，你应该看到轮转的备份文件")
	fmt.Println("   - 可以多次运行程序观察轮转行为")
}
