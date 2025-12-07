# Fields Package

字段标准化包，定义统一的日志字段名称和格式规范，确保不同引擎（Zap/Slog）输出完全一致的字段结构。

## 概述

`fields` 包是项目的核心组件之一，实现了字段标准化要求：

- **统一字段名**：定义标准的字段名称常量，避免引擎间差异
- **字段映射**：提供各种常见字段名到标准名称的映射
- **编码配置**：统一的日志编码格式配置
- **类型转换**：标准化的日志输出结构
- **验证机制**：字段名称规范验证

## 设计原则

### 字段标准化核心要求

**核心原则**：无论底层使用 Zap 还是 Slog，输出的日志字段必须完全一致：

- **时间字段**：统一使用 `timestamp` 字段名
- **级别字段**：统一使用 `level` 字段名，值格式一致（小写）
- **消息字段**：统一使用 `message` 字段名
- **调用者字段**：统一使用 `caller` 字段名和格式
- **跟踪字段**：统一使用 `trace_id`, `span_id` 等字段名
- **自定义字段**：用户添加的字段保持原样

## 标准字段定义

### 核心日志字段

```go
const (
    TimestampField = "timestamp"    // 时间戳
    LevelField     = "level"        // 日志级别
    MessageField   = "message"      // 日志消息
    CallerField    = "caller"       // 调用位置
)
```

### 追踪字段

```go
const (
    TraceIDField = "trace_id"       // 追踪ID
    SpanIDField  = "span_id"        // 跨度ID
)
```

### 错误字段

```go
const (
    ErrorField      = "error"       // 错误信息
    ErrorTypeField  = "error_type"  // 错误类型
    StacktraceField = "stacktrace"  // 堆栈跟踪
)
```

### 服务识别字段

```go
const (
    ServiceField     = "service"         // 服务名称
    ServiceVersion   = "service_version" // 服务版本
    EnvironmentField = "environment"     // 运行环境
)
```

### 请求上下文字段

```go
const (
    RequestIDField = "request_id"   // 请求ID
    UserIDField    = "user_id"      // 用户ID
    SessionIDField = "session_id"   // 会话ID
)
```

### 性能字段

```go
const (
    DurationField = "duration"      // 持续时间
    LatencyField  = "latency"       // 延迟时间
)
```

## 使用方式

### 1. 在引擎实现中使用标准字段

```go
package zap

import "github.com/kart-io/logger/fields"

func configureZapEncoder() zapcore.EncoderConfig {
    return zapcore.EncoderConfig{
        TimeKey:      fields.TimestampField,    // "timestamp"
        LevelKey:     fields.LevelField,        // "level"
        MessageKey:   fields.MessageField,      // "message"
        CallerKey:    fields.CallerField,       // "caller"
        EncodeLevel:  zapcore.LowercaseLevelEncoder,
        EncodeTime:   zapcore.RFC3339NanoTimeEncoder,
        EncodeCaller: zapcore.ShortCallerEncoder,
    }
}
```

### 2. 在应用代码中使用标准字段

```go
package main

import (
    "github.com/kart-io/logger"
    "github.com/kart-io/logger/fields"
)

func main() {
    logger, _ := logger.NewWithDefaults()
    
    // 使用标准字段常量
    logger.Infow("HTTP 请求处理完成",
        fields.RequestIDField, "req-12345",       // "request_id"
        fields.UserIDField, "user-67890",         // "user_id"
        fields.DurationField, 156,                // "duration"
        fields.ServiceField, "user-api",          // "service"
        fields.EnvironmentField, "production",    // "environment"
    )
    
    // 错误记录
    logger.Errorw("数据库连接失败",
        fields.ErrorField, err.Error(),           // "error"
        fields.ErrorTypeField, "connection",      // "error_type"
        fields.ServiceField, "user-api",          // "service"
    )
    
    // 追踪上下文
    logger.Infow("处理用户请求",
        fields.TraceIDField, "trace-abc123",      // "trace_id"
        fields.SpanIDField, "span-def456",        // "span_id"
        fields.RequestIDField, "req-789",         // "request_id"
    )
}
```

### 3. 使用字段映射器

```go
package main

import "github.com/kart-io/logger/fields"

func fieldMappingExample() {
    mapper := fields.NewFieldMapper()
    
    // 获取核心字段映射
    coreFields := mapper.MapCoreFields()
    /*
    map[string]string{
        "ts":        "timestamp",
        "time":      "timestamp",
        "timestamp": "timestamp",
        "level":     "level",
        "msg":       "message",
        "message":   "message",
        "caller":    "caller",
        "source":    "caller",
    }
    */
    
    // 获取追踪字段映射
    tracingFields := mapper.MapTracingFields()
    /*
    map[string]string{
        "trace.id":  "trace_id",
        "trace_id":  "trace_id",
        "traceId":   "trace_id",
        "span.id":   "span_id",
        "span_id":   "span_id",
        "spanId":    "span_id",
    }
    */
    
    // 验证字段名称
    isValid := mapper.ValidateFieldName("user_id")  // true
    isInvalid := mapper.ValidateFieldName("userId") // true（允许自定义字段）
}
```

## 编码器配置

### 默认编码配置

```go
package fields

func DefaultEncoderConfig() *EncoderConfig {
    return &EncoderConfig{
        TimeLayout:     time.RFC3339Nano,        // ISO 8601 格式
        LevelFormatter: LowercaseLevelFormatter, // 小写级别
        CallerFormat:   ShortCallerFormatter,    // 短路径格式
    }
}
```

### 自定义编码配置

```go
func customEncoderExample() {
    config := &fields.EncoderConfig{
        TimeLayout:     "2006-01-02 15:04:05.000", // 自定义时间格式
        LevelFormatter: fields.UppercaseLevelFormatter, // 大写级别
        CallerFormat:   fields.FullCallerFormatter,     // 完整路径
    }
    
    // 在引擎配置中使用
}
```

### 级别格式化选项

```go
const (
    UppercaseLevelFormatter LevelFormatter = iota  // "DEBUG", "INFO", "WARN"
    LowercaseLevelFormatter                        // "debug", "info", "warn" (推荐)
)
```

### 调用者格式化选项

```go
const (
    ShortCallerFormatter CallerFormatter = iota    // "file.go:123" (推荐)
    FullCallerFormatter                           // "/full/path/file.go:123"
)
```

## 标准化输出

### StandardizedOutput 结构

```go
type StandardizedOutput struct {
    Timestamp string                 `json:"timestamp"`
    Level     string                 `json:"level"`
    Message   string                 `json:"message"`
    Caller    string                 `json:"caller,omitempty"`
    Fields    map[string]interface{} `json:",inline"`
}
```

### JSON 输出示例

```go
func standardizedOutputExample() {
    output := &fields.StandardizedOutput{
        Timestamp: "2023-09-04T12:34:56.789Z",
        Level:     "info",
        Message:   "用户登录成功",
        Caller:    "auth.go:123",
        Fields: map[string]interface{}{
            fields.UserIDField:    "user-12345",
            fields.RequestIDField: "req-67890",
            fields.ServiceField:   "auth-service",
        },
    }
    
    jsonBytes, err := output.ToJSON()
    if err != nil {
        panic(err)
    }
    
    /*
    输出 JSON:
    {
        "timestamp": "2023-09-04T12:34:56.789Z",
        "level": "info",
        "message": "用户登录成功",
        "caller": "auth.go:123",
        "user_id": "user-12345",
        "request_id": "req-67890",
        "service": "auth-service"
    }
    */
}
```

## 实际应用场景

### 1. Web 应用日志记录

```go
func webRequestLogger(logger core.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        requestID := generateRequestID()
        
        // 请求开始日志
        logger.Infow("HTTP 请求开始",
            fields.RequestIDField, requestID,
            fields.ServiceField, "web-api",
            "method", r.Method,
            "path", r.URL.Path,
            "user_agent", r.UserAgent(),
            "remote_addr", r.RemoteAddr,
        )
        
        // 处理请求...
        
        // 请求完成日志
        duration := time.Since(start)
        logger.Infow("HTTP 请求完成",
            fields.RequestIDField, requestID,
            fields.DurationField, duration.Milliseconds(),
            fields.ServiceField, "web-api",
            "status_code", 200,
            "response_size", 1024,
        )
    }
}
```

### 2. 数据库操作日志

```go
func databaseLogger(logger core.Logger) {
    // 数据库查询开始
    logger.Debugw("数据库查询开始",
        fields.ServiceField, "database",
        "query", "SELECT * FROM users WHERE id = ?",
        "table", "users",
    )
    
    // 查询失败
    if err != nil {
        logger.Errorw("数据库查询失败",
            fields.ErrorField, err.Error(),
            fields.ErrorTypeField, "query_timeout",
            fields.ServiceField, "database",
            fields.DurationField, 5000,
            "query", "SELECT * FROM users WHERE id = ?",
        )
        return
    }
    
    // 查询成功
    logger.Infow("数据库查询完成",
        fields.ServiceField, "database",
        fields.DurationField, 156,
        "query", "SELECT * FROM users WHERE id = ?",
        "rows_affected", 1,
    )
}
```

### 3. 微服务追踪日志

```go
func microserviceHandler(ctx context.Context, logger core.Logger) {
    traceID := extractTraceID(ctx)
    spanID := extractSpanID(ctx)
    
    // 创建带追踪信息的日志器
    tracedLogger := logger.With(
        fields.TraceIDField, traceID,
        fields.SpanIDField, spanID,
        fields.ServiceField, "order-service",
    )
    
    tracedLogger.Info("开始处理订单")
    
    // 调用其他服务
    tracedLogger.Debugw("调用库存服务",
        "target_service", "inventory-service",
        "product_id", "prod-123",
        "quantity", 2,
    )
    
    // 处理完成
    tracedLogger.Infow("订单处理完成",
        "order_id", "order-456",
        fields.DurationField, 250,
        "status", "confirmed",
    )
}
```

## 字段命名规范

### 推荐命名约定

1. **使用下划线分隔**：`user_id`, `request_id`, `error_type`
2. **避免驼峰命名**：使用 `user_id` 而非 `userId`
3. **保持简洁明了**：使用 `duration` 而非 `execution_duration_milliseconds`
4. **统一时态**：使用 `created_at` 而非 `creation_time`
5. **避免缩写**：使用 `environment` 而非 `env`（除非是广泛认知的缩写）

### 常见字段名映射

| 不推荐 | 推荐 | 说明 |
|--------|------|------|
| `userId` | `user_id` | 下划线分隔 |
| `requestId` | `request_id` | 下划线分隔 |
| `traceId` | `trace_id` | 下划线分隔 |
| `msg` | `message` | 使用完整单词 |
| `ts` | `timestamp` | 使用完整单词 |
| `lvl` | `level` | 使用完整单词 |
| `err` | `error` | 使用完整单词 |

## 引擎适配示例

### Zap 引擎适配

```go
package zap

import (
    "go.uber.org/zap/zapcore"
    "github.com/kart-io/logger/fields"
)

func createZapConfig() zapcore.EncoderConfig {
    encoderConfig := fields.DefaultEncoderConfig()
    
    return zapcore.EncoderConfig{
        TimeKey:        fields.TimestampField,
        LevelKey:       fields.LevelField,
        NameKey:        "logger",
        CallerKey:      fields.CallerField,
        MessageKey:     fields.MessageField,
        StacktraceKey:  fields.StacktraceField,
        
        // 使用标准化配置
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
        EncodeDuration: zapcore.SecondsDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }
}
```

### Slog 引擎适配

```go
package slog

import (
    "log/slog"
    "github.com/kart-io/logger/fields"
)

func createSlogHandler() slog.Handler {
    opts := &slog.HandlerOptions{
        Level: slog.LevelDebug,
        ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
            // 标准化字段名映射
            switch attr.Key {
            case slog.TimeKey:
                return slog.Attr{Key: fields.TimestampField, Value: attr.Value}
            case slog.LevelKey:
                return slog.Attr{
                    Key:   fields.LevelField,
                    Value: slog.StringValue(strings.ToLower(attr.Value.String())),
                }
            case slog.MessageKey:
                return slog.Attr{Key: fields.MessageField, Value: attr.Value}
            case slog.SourceKey:
                return slog.Attr{Key: fields.CallerField, Value: attr.Value}
            }
            return attr
        },
    }
    
    return slog.NewJSONHandler(os.Stdout, opts)
}
```

## 验证和测试

### 字段名验证

```go
func validateFieldsExample() {
    mapper := fields.NewFieldMapper()
    
    testFields := []string{
        "user_id",      // ✅ 标准字段
        "request_id",   // ✅ 标准字段  
        "custom_field", // ✅ 自定义字段（允许）
        "level",        // ✅ 核心字段
    }
    
    for _, field := range testFields {
        if mapper.ValidateFieldName(field) {
            fmt.Printf("✅ 字段 %s 符合规范\n", field)
        } else {
            fmt.Printf("❌ 字段 %s 不符合规范\n", field)
        }
    }
}
```

### 一致性测试

```go
func consistencyTest() {
    // 测试两个引擎输出的字段是否一致
    zapLogger, _ := zap.NewZapLogger(option.DefaultLogOption())
    slogLogger, _ := slog.NewSlogLogger(option.DefaultLogOption())
    
    testData := map[string]interface{}{
        fields.UserIDField:    "user-123",
        fields.RequestIDField: "req-456",
        fields.ServiceField:   "test-service",
    }
    
    // 两个引擎应该输出相同的字段结构
    zapLogger.Infow("测试消息", testData)
    slogLogger.Infow("测试消息", testData)
}
```

## 最佳实践

1. **始终使用字段常量**：避免硬编码字段名
2. **保持字段名一致**：同一概念使用相同字段名
3. **遵循命名约定**：使用下划线分隔的小写命名
4. **验证字段规范**：使用 `ValidateFieldName` 检查自定义字段
5. **使用映射器**：通过 `FieldMapper` 处理不同来源的字段名

## 注意事项

1. **字段标准化是核心要求**：确保不同引擎输出完全一致
2. **级别格式统一**：推荐使用小写级别格式
3. **时间格式标准**：建议使用 RFC3339Nano 格式
4. **自定义字段支持**：允许用户添加自定义字段，但建议遵循命名约定
5. **映射器性能**：字段映射操作轻量级，可在热路径中使用

## 相关包

- [`core`](../core/) - 核心接口定义
- [`engines/zap`](../engines/zap/) - Zap 引擎实现
- [`engines/slog`](../engines/slog/) - Slog 引擎实现
- [`option`](../option/) - 配置选项管理