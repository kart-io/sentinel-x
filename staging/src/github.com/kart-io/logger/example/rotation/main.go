package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/fields"
)

// RotationLogger 实现了支持日志轮转的 logger
type RotationLogger struct {
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	level  core.Level
	mapper *fields.FieldMapper
}

// NewRotationLogger 创建支持轮转的日志器
func NewRotationLogger(writer io.Writer, level core.Level, format string) *RotationLogger {
	// 创建编码器配置
	var encoderConfig zapcore.EncoderConfig
	if format == "console" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = fields.TimestampField
		encoderConfig.LevelKey = fields.LevelField
		encoderConfig.MessageKey = fields.MessageField
		encoderConfig.CallerKey = fields.CallerField
		encoderConfig.StacktraceKey = fields.StacktraceField
	}

	// 创建编码器
	var encoder zapcore.Encoder
	if format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建写同步器
	writeSyncer := zapcore.AddSync(writer)

	// 映射日志级别
	var zapLevel zapcore.Level
	switch level {
	case core.DebugLevel:
		zapLevel = zapcore.DebugLevel
	case core.InfoLevel:
		zapLevel = zapcore.InfoLevel
	case core.WarnLevel:
		zapLevel = zapcore.WarnLevel
	case core.ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	case core.FatalLevel:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 创建 zap core
	core := zapcore.NewCore(encoder, writeSyncer, zapLevel)

	// 创建 logger
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	logger = logger.With(zap.String("engine", "zap-rotation"))

	return &RotationLogger{
		zap:    logger,
		sugar:  logger.Sugar(),
		level:  level,
		mapper: fields.NewFieldMapper(),
	}
}

// 实现 core.Logger 接口的所有方法

func (r *RotationLogger) Debug(args ...interface{}) {
	r.sugar.Debug(args...)
}

func (r *RotationLogger) Info(args ...interface{}) {
	r.sugar.Info(args...)
}

func (r *RotationLogger) Warn(args ...interface{}) {
	r.sugar.Warn(args...)
}

func (r *RotationLogger) Error(args ...interface{}) {
	r.sugar.Error(args...)
}

func (r *RotationLogger) Fatal(args ...interface{}) {
	r.sugar.Fatal(args...)
}

func (r *RotationLogger) Debugf(template string, args ...interface{}) {
	r.sugar.Debugf(template, args...)
}

func (r *RotationLogger) Infof(template string, args ...interface{}) {
	r.sugar.Infof(template, args...)
}

func (r *RotationLogger) Warnf(template string, args ...interface{}) {
	r.sugar.Warnf(template, args...)
}

func (r *RotationLogger) Errorf(template string, args ...interface{}) {
	r.sugar.Errorf(template, args...)
}

func (r *RotationLogger) Fatalf(template string, args ...interface{}) {
	r.sugar.Fatalf(template, args...)
}

func (r *RotationLogger) Debugw(msg string, keysAndValues ...interface{}) {
	r.sugar.Debugw(msg, keysAndValues...)
}

func (r *RotationLogger) Infow(msg string, keysAndValues ...interface{}) {
	r.sugar.Infow(msg, keysAndValues...)
}

func (r *RotationLogger) Warnw(msg string, keysAndValues ...interface{}) {
	r.sugar.Warnw(msg, keysAndValues...)
}

func (r *RotationLogger) Errorw(msg string, keysAndValues ...interface{}) {
	r.sugar.Errorw(msg, keysAndValues...)
}

func (r *RotationLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	r.sugar.Fatalw(msg, keysAndValues...)
}

func (r *RotationLogger) With(keysAndValues ...interface{}) core.Logger {
	return &RotationLogger{
		zap:    r.zap.With(convertToZapFields(keysAndValues...)...),
		sugar:  r.sugar.With(keysAndValues...),
		level:  r.level,
		mapper: r.mapper,
	}
}

func (r *RotationLogger) WithCtx(ctx context.Context, keysAndValues ...interface{}) core.Logger {
	return r.With(keysAndValues...)
}

func (r *RotationLogger) WithCallerSkip(skip int) core.Logger {
	newLogger := r.zap.WithOptions(zap.AddCallerSkip(skip))
	return &RotationLogger{
		zap:    newLogger,
		sugar:  newLogger.Sugar(),
		level:  r.level,
		mapper: r.mapper,
	}
}

func (r *RotationLogger) SetLevel(level core.Level) {
	r.level = level
}

func (r *RotationLogger) Flush() error {
	return r.zap.Sync()
}

// 辅助函数
func convertToZapFields(keysAndValues ...interface{}) []zap.Field {
	var fields []zap.Field
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			value := keysAndValues[i+1]
			fields = append(fields, zap.Any(key, value))
		}
	}
	return fields
}

func main() {
	fmt.Println("=== 日志轮转集成示例 ===")
	fmt.Println()

	// 确保日志目录存在
	os.MkdirAll("./logs", 0755)

	// 示例 1：基本的 lumberjack 轮转
	fmt.Println("1. 基本轮转示例")
	basicRotationExample()

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// 示例 2：高级轮转配置
	fmt.Println("2. 高级轮转配置")
	advancedRotationExample()

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// 示例 3：多输出轮转
	fmt.Println("3. 多输出轮转")
	multiOutputExample()

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// 示例 4：生产环境配置
	fmt.Println("4. 生产环境配置")
	productionExample()

	fmt.Println("\n示例运行完成！请查看 ./logs 目录中的日志文件。")
}

// 基本轮转示例
func basicRotationExample() {
	// 创建基本的轮转写入器
	rotateWriter := &lumberjack.Logger{
		Filename:   "./logs/basic.log",
		MaxSize:    1,     // 1MB（测试用小值）
		MaxBackups: 3,     // 保留 3 个备份
		MaxAge:     7,     // 7 天后删除
		Compress:   false, // 不压缩（便于查看）
		LocalTime:  true,
	}

	// 创建 logger
	logger := NewRotationLogger(rotateWriter, core.InfoLevel, "json")

	// 写入一些日志
	for i := 0; i < 50; i++ {
		logger.Infow("基本轮转测试",
			"iteration", i,
			"timestamp", time.Now().Unix(),
			"message_size", "这是一条用于测试日志轮转功能的较长消息，包含中文字符以增加文件大小")

		if i%10 == 0 {
			logger.Warnw("检查点",
				"checkpoint", i,
				"status", "running")
		}
	}

	// 刷新缓冲区
	logger.Flush()

	fmt.Printf("基本轮转完成，生成文件：%s\n", rotateWriter.Filename)
}

// 高级轮转配置示例
func advancedRotationExample() {
	// 按时间和大小双重策略轮转
	rotateWriter := &lumberjack.Logger{
		Filename:   "./logs/advanced.log",
		MaxSize:    2,    // 2MB
		MaxBackups: 5,    // 保留 5 个备份
		MaxAge:     3,    // 3 天后删除
		Compress:   true, // 压缩旧文件
		LocalTime:  true,
	}

	logger := NewRotationLogger(rotateWriter, core.DebugLevel, "json")

	// 模拟不同级别的日志
	scenarios := []struct {
		level string
		count int
	}{
		{"debug", 20},
		{"info", 30},
		{"warn", 10},
		{"error", 5},
	}

	for _, scenario := range scenarios {
		for i := 0; i < scenario.count; i++ {
			switch scenario.level {
			case "debug":
				logger.Debugw("调试信息",
					"scenario", scenario.level,
					"iteration", i,
					"details", "这是调试级别的详细信息，通常包含程序运行状态")
			case "info":
				logger.Infow("信息记录",
					"scenario", scenario.level,
					"iteration", i,
					"action", "用户操作记录",
					"user_id", fmt.Sprintf("user_%d", i))
			case "warn":
				logger.Warnw("警告信息",
					"scenario", scenario.level,
					"iteration", i,
					"warning_type", "性能警告",
					"threshold_exceeded", true)
			case "error":
				logger.Errorw("错误信息",
					"scenario", scenario.level,
					"iteration", i,
					"error_type", "业务异常",
					"error_code", "E001")
			}
		}
	}

	logger.Flush()
	fmt.Printf("高级轮转完成，生成文件：%s\n", rotateWriter.Filename)
}

// 多输出示例
func multiOutputExample() {
	// 创建文件轮转写入器
	fileWriter := &lumberjack.Logger{
		Filename:   "./logs/multi.log",
		MaxSize:    1,
		MaxBackups: 2,
		MaxAge:     1,
		Compress:   false,
		LocalTime:  true,
	}

	// 组合控制台和文件输出
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	logger := NewRotationLogger(multiWriter, core.InfoLevel, "console")

	// 写入日志（同时输出到控制台和文件）
	logger.Info("多输出示例开始")

	for i := 0; i < 20; i++ {
		logger.Infow("多输出测试",
			"output_targets", []string{"console", "file"},
			"iteration", i,
			"timestamp", time.Now().Format(time.RFC3339))

		time.Sleep(100 * time.Millisecond) // 短暂延迟以便观察
	}

	logger.Info("多输出示例结束")
	logger.Flush()
}

// 生产环境配置示例
func productionExample() {
	// 生产环境的轮转配置
	prodWriter := &lumberjack.Logger{
		Filename:   "./logs/production.log",
		MaxSize:    100,  // 100MB
		MaxBackups: 15,   // 保留 15 个文件（约 15 天）
		MaxAge:     15,   // 15 天后删除
		Compress:   true, // 压缩节省空间
		LocalTime:  true,
	}

	logger := NewRotationLogger(prodWriter, core.InfoLevel, "json")

	// 添加生产环境的通用字段
	prodLogger := logger.With(
		"service.name", "example-service",
		"service.version", "1.0.0",
		"environment", "production",
		"datacenter", "dc1",
	)

	// 模拟生产环境的各种日志
	scenarios := []struct {
		name     string
		action   func(core.Logger)
		interval time.Duration
	}{
		{
			name: "应用启动",
			action: func(l core.Logger) {
				l.Infow("服务启动",
					"startup_time", time.Now().Unix(),
					"pid", os.Getpid(),
					"config_loaded", true)
			},
		},
		{
			name: "用户请求",
			action: func(l core.Logger) {
				l.Infow("HTTP请求处理",
					"method", "POST",
					"path", "/api/users",
					"status_code", 200,
					"duration_ms", 45,
					"user_id", "user_12345")
			},
		},
		{
			name: "数据库操作",
			action: func(l core.Logger) {
				l.Infow("数据库查询",
					"operation", "SELECT",
					"table", "users",
					"duration_ms", 12,
					"rows_affected", 1)
			},
		},
		{
			name: "缓存命中",
			action: func(l core.Logger) {
				l.Debugw("缓存操作",
					"operation", "GET",
					"key", "user:12345",
					"hit", true,
					"ttl", 3600)
			},
		},
		{
			name: "业务警告",
			action: func(l core.Logger) {
				l.Warnw("性能警告",
					"metric", "response_time",
					"value", 2500,
					"threshold", 2000,
					"endpoint", "/api/heavy-operation")
			},
		},
	}

	// 执行生产环境日志记录
	for round := 0; round < 10; round++ {
		for _, scenario := range scenarios {
			scenario.action(prodLogger)
		}
	}

	prodLogger.Flush()
	fmt.Printf("生产环境示例完成，生成文件：%s\n", prodWriter.Filename)
}
