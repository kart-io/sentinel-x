package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/option"
)

// SignalAwareLogger 支持信号处理的日志器包装
type SignalAwareLogger struct {
	core.Logger
	loggerFactory func() (core.Logger, error)
	cancel        context.CancelFunc
}

// NewSignalAwareLogger 创建支持信号处理的日志器
func NewSignalAwareLogger(loggerFactory func() (core.Logger, error)) (*SignalAwareLogger, error) {
	logger, err := loggerFactory()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	sal := &SignalAwareLogger{
		Logger:        logger,
		loggerFactory: loggerFactory,
		cancel:        cancel,
	}

	// 启动信号监听
	go sal.handleSignals(ctx)

	return sal, nil
}

// handleSignals 处理系统信号
func (sal *SignalAwareLogger) handleSignals(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)

	// 监听 USR1 信号（用于 logrotate）和其他常用信号
	signal.Notify(sigChan,
		syscall.SIGUSR1, // logrotate 轮转信号
		syscall.SIGHUP,  // 重新加载配置
		syscall.SIGTERM, // 优雅关闭
		syscall.SIGINT,  // 中断信号
	)

	for {
		select {
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGUSR1:
				sal.handleLogRotation()
			case syscall.SIGHUP:
				sal.handleConfigReload()
			case syscall.SIGTERM, syscall.SIGINT:
				sal.handleGracefulShutdown()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleLogRotation 处理日志轮转信号
func (sal *SignalAwareLogger) handleLogRotation() {
	sal.Info("收到日志轮转信号 (USR1)")

	// 刷新当前日志缓冲区
	if err := sal.Flush(); err != nil {
		sal.Errorf("刷新日志缓冲区失败: %v", err)
	}

	// 重新创建日志器（重新打开文件）
	newLogger, err := sal.loggerFactory()
	if err != nil {
		sal.Errorf("重新创建日志器失败: %v", err)
		return
	}

	// 原子替换日志器
	sal.Logger = newLogger
	sal.Info("日志轮转完成，日志文件已重新打开")
}

// handleConfigReload 处理配置重载信号
func (sal *SignalAwareLogger) handleConfigReload() {
	sal.Info("收到配置重载信号 (HUP)")

	// 这里可以重新读取配置文件并重建日志器
	newLogger, err := sal.loggerFactory()
	if err != nil {
		sal.Errorf("重新加载配置失败: %v", err)
		return
	}

	sal.Logger = newLogger
	sal.Info("配置重载完成")
}

// handleGracefulShutdown 处理优雅关闭
func (sal *SignalAwareLogger) handleGracefulShutdown() {
	sal.Info("收到关闭信号，准备优雅退出")

	// 刷新日志缓冲区
	if err := sal.Flush(); err != nil {
		sal.Errorf("最终刷新日志失败: %v", err)
	}

	sal.Info("日志器已安全关闭")
}

// Close 关闭信号处理
func (sal *SignalAwareLogger) Close() {
	if sal.cancel != nil {
		sal.cancel()
	}

	// 最后刷新一次
	sal.Flush()
}

// 使用示例
func main() {
	// 定义日志器工厂函数
	loggerFactory := func() (core.Logger, error) {
		// 确保日志目录存在
		logDir := "./logs"
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}

		opt := &option.LogOption{
			Engine:      "zap",
			Level:       "INFO",
			Format:      "json",
			OutputPaths: []string{"./logs/signal-demo.log"},
		}

		return logger.New(opt)
	}

	// 创建支持信号处理的日志器
	sigLogger, err := NewSignalAwareLogger(loggerFactory)
	if err != nil {
		panic(err)
	}
	defer sigLogger.Close()

	// 设置为全局日志器
	logger.SetGlobal(sigLogger)

	sigLogger.Info("应用程序启动，信号处理已就绪")
	sigLogger.Infow("信号处理器配置",
		"支持的信号", []string{"USR1", "HUP", "TERM", "INT"},
		"USR1", "日志轮转",
		"HUP", "配置重载",
		"TERM/INT", "优雅关闭")

	// 模拟应用程序运行
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ticker.C:
			counter++
			sigLogger.Infow("应用程序心跳",
				"counter", counter,
				"timestamp", time.Now().Unix())

			// 每10次心跳记录一些额外信息
			if counter%10 == 0 {
				sigLogger.Infow("应用状态检查",
					"heartbeat_count", counter,
					"uptime", time.Since(time.Now().Add(-time.Duration(counter)*5*time.Second)),
					"status", "healthy")
			}

			// 演示结束条件
			if counter >= 100 {
				sigLogger.Info("演示结束，准备退出")
				return
			}
		}
	}
}

// 测试脚本使用说明：
//
// 1. 编译程序：
//    go build -o signal-demo signal-handler.go
//
// 2. 在一个终端运行程序：
//    ./signal-demo
//
// 3. 在另一个终端测试信号：
//    # 测试日志轮转
//    kill -USR1 $(pidof signal-demo)
//
//    # 测试配置重载
//    kill -HUP $(pidof signal-demo)
//
//    # 优雅关闭
//    kill -TERM $(pidof signal-demo)
//
// 4. 结合 logrotate 测试：
//    # 创建测试的 logrotate 配置
//    echo "/tmp/test-app.log {
//        size 1k
//        rotate 3
//        missingok
//        notifempty
//        postrotate
//            kill -USR1 \$(pidof signal-demo) 2>/dev/null || true
//        endscript
//    }" | sudo tee /etc/logrotate.d/test-app
//
//    # 强制运行 logrotate
//    sudo logrotate -f /etc/logrotate.d/test-app
