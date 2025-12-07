package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// 简化的测试程序
func main() {
	fmt.Println("=== 日志轮转测试 ===")

	// 1. 首先确保目录存在
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}
	fmt.Printf("日志目录已创建: %s\n", logDir)

	// 2. 检查目录是否真的存在
	if info, err := os.Stat(logDir); err != nil {
		fmt.Printf("无法访问日志目录: %v\n", err)
		return
	} else {
		fmt.Printf("目录信息: %+v\n", info)
	}

	// 3. 创建 lumberjack 轮转器
	rotateWriter := &lumberjack.Logger{
		Filename:   "./logs/test.log",
		MaxSize:    1, // 1MB（测试用小值）
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		LocalTime:  true,
	}

	// 4. 测试写入
	fmt.Println("开始写入测试...")
	for i := 0; i < 10; i++ {
		message := fmt.Sprintf("测试日志条目 %d - %s\n", i, time.Now().Format(time.RFC3339))
		if _, err := rotateWriter.Write([]byte(message)); err != nil {
			fmt.Printf("写入失败: %v\n", err)
			return
		}
		fmt.Printf("已写入: %s", message)
		time.Sleep(100 * time.Millisecond)
	}

	// 5. 刷新并检查文件
	if err := rotateWriter.Close(); err != nil {
		fmt.Printf("关闭文件失败: %v\n", err)
	}

	// 6. 列出生成的文件
	fmt.Println("\n生成的日志文件:")
	if files, err := os.ReadDir(logDir); err != nil {
		fmt.Printf("读取目录失败: %v\n", err)
	} else {
		for _, file := range files {
			info, _ := file.Info()
			fmt.Printf("  %s (大小: %d 字节)\n", file.Name(), info.Size())
		}
	}

	fmt.Println("测试完成!")
}
