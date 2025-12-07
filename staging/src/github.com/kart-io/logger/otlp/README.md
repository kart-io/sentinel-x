# OTLP Package

OTLP (OpenTelemetry Protocol) 集成包，提供标准化的遥测日志数据传输功能，支持多种后端（Jaeger、VictoriaLogs、OpenTelemetry Collector 等）。

## 概述

`otlp` 包实现了完整的 OTLP 日志传输功能：

- **多协议支持**：gRPC 和 HTTP 两种传输协议
- **标准兼容**：完全符合 OpenTelemetry 规范
- **后端适配**：特别优化对 VictoriaLogs 等后端的兼容性
- **资源管理**：自动管理连接和资源清理
- **错误处理**：完善的错误处理和调试信息
- **类型转换**：智能的 Go 类型到 OTLP 类型转换

## 核心组件

### LoggerProvider

```go
type LoggerProvider struct {
    client   *OTLPClient
    resource *resourcev1.Resource
}
```

管理 OTLP 日志客户端和资源信息的提供者。

### OTLPClient

```go
type OTLPClient struct {
    endpoint   string
    protocol   string
    timeout    time.Duration
    headers    map[string]string
    insecure   bool
    
    // gRPC 客户端
    grpcConn   *grpc.ClientConn
    grpcClient v1.LogsServiceClient
    
    // HTTP 客户端
    httpClient *http.Client
}
```

实际处理 OTLP 数据传输的客户端。

## 使用方式

### 1. 基本集成（通过配置自动启用）

```go
package main

import (
    "github.com/kart-io/logger"
    "github.com/kart-io/logger/option"
)

func main() {
    opt := &option.LogOption{
        Engine: "zap",
        Level:  "info",
        Format: "json",
        // 设置 OTLP 端点会自动启用
        OTLPEndpoint: "http://localhost:4317",
    }
    
    logger, err := logger.New(opt)
    if err != nil {
        panic(err)
    }
    
    // 日志会同时输出到控制台和 OTLP 后端
    logger.Infow("用户登录", 
        "user_id", "12345",
        "ip", "192.168.1.100",
        "user_agent", "Mozilla/5.0...",
    )
}
```

### 2. 详细 OTLP 配置

```go
package main

import (
    "time"
    "github.com/kart-io/logger"
    "github.com/kart-io/logger/option"
)

func main() {
    opt := &option.LogOption{
        Engine: "slog",
        Level:  "debug",
        Format: "json",
        
        // 嵌套 OTLP 配置（高级控制）
        OTLP: &option.OTLPOption{
            Enabled:  true,
            Endpoint: "https://jaeger.example.com:4317",
            Protocol: "grpc",
            Timeout:  15 * time.Second,
            Headers: map[string]string{
                "Authorization": "Bearer your-token",
                "X-Tenant-ID":   "tenant-123",
            },
        },
    }
    
    logger, err := logger.New(opt)
    if err != nil {
        panic(err)
    }
    
    logger.Errorw("API 请求失败",
        "method", "POST",
        "url", "/api/users",
        "status_code", 500,
        "error", "database connection timeout",
        "duration_ms", 5000,
    )
}
```

### 3. 直接使用 OTLP 提供者

```go
package main

import (
    "context"
    "github.com/kart-io/logger/otlp"
    "github.com/kart-io/logger/option"
    "github.com/kart-io/logger/core"
)

func directOTLPUsage() {
    // 创建 OTLP 选项
    otlpOpt := &option.OTLPOption{
        Enabled:  true,
        Endpoint: "http://localhost:4317",
        Protocol: "grpc",
        Timeout:  10 * time.Second,
    }
    
    // 创建 OTLP 提供者
    provider, err := otlp.NewLoggerProvider(context.Background(), otlpOpt)
    if err != nil {
        panic(err)
    }
    defer provider.Shutdown(context.Background())
    
    // 直接发送日志记录
    attributes := map[string]interface{}{
        "service_name": "user-api",
        "version":      "1.2.3",
        "environment":  "production",
        "user_id":      12345,
        "request_id":   "req-abc123",
        "duration":     156.7,
        "success":      true,
    }
    
    err = provider.SendLogRecord(
        core.InfoLevel,
        "用户注册成功",
        attributes,
    )
    if err != nil {
        fmt.Printf("OTLP 发送失败: %v\n", err)
    }
}
```

## 后端集成示例

### 1. Jaeger 集成

```yaml
# docker-compose.yml
version: '3.8'
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "14268:14268"   # HTTP collector
      - "4317:4317"     # OTLP gRPC receiver
      - "4318:4318"     # OTLP HTTP receiver
      - "16686:16686"   # Web UI
    environment:
      - COLLECTOR_OTLP_ENABLED=true
```

```go
// 应用配置
opt := &option.LogOption{
    Engine:       "zap",
    Level:        "info",
    OTLPEndpoint: "http://localhost:4317",  // Jaeger OTLP gRPC
    // 或使用 HTTP: "http://localhost:4318/v1/logs"
}
```

### 2. VictoriaLogs 集成

```yaml
# docker-compose.yml
version: '3.8'
services:
  victorialogs:
    image: victoriametrics/victoria-logs:latest
    ports:
      - "9428:9428"   # HTTP API
      - "4317:4317"   # OTLP gRPC receiver
    command:
      - '-storageDataPath=/logs-data'
      - '-loggerLevel=INFO'
      - '-otlp.enabled=true'
      - '-otlp.listenAddr=:4317'
```

```go
// 应用配置
opt := &option.LogOption{
    Engine:       "slog",
    Level:        "debug",
    Format:       "json",
    OTLPEndpoint: "http://localhost:4317",
}

logger, _ := logger.New(opt)

// VictoriaLogs 特别优化的字段
logger.Infow("HTTP 请求",
    "method", "GET",           // 标准字段
    "path", "/api/health",     // 标准字段
    "status", 200,             // 标准字段
    "job", "web-server",       // VictoriaLogs 流字段
    "instance", "web-01",      // VictoriaLogs 流字段
)
```

### 3. OpenTelemetry Collector 集成

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:

exporters:
  logging:
    loglevel: debug
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, jaeger]
```

```go
// 应用配置
opt := &option.LogOption{
    Engine:       "zap",
    Level:        "info",
    OTLPEndpoint: "http://otel-collector:4317",
}
```

## 协议支持

### gRPC 协议

```go
opt := &option.LogOption{
    OTLP: &option.OTLPOption{
        Enabled:  true,
        Endpoint: "grpc://localhost:4317",  // 或 "localhost:4317"
        Protocol: "grpc",
        Timeout:  10 * time.Second,
        Headers: map[string]string{
            "authorization": "Bearer token",
        },
    },
}
```

**特点**：
- 高性能二进制协议
- 内置流式支持
- 更好的错误处理
- 推荐用于生产环境

### HTTP 协议

```go
opt := &option.LogOption{
    OTLP: &option.OTLPOption{
        Enabled:  true,
        Endpoint: "http://localhost:4318/v1/logs",
        Protocol: "http",
        Timeout:  15 * time.Second,
        Headers: map[string]string{
            "Content-Type": "application/x-protobuf",
            "X-API-Key":    "your-api-key",
        },
    },
}
```

**特点**：
- 标准 HTTP/1.1 或 HTTP/2
- 易于调试和监控
- 更好的防火墙兼容性
- 适合简单部署场景

## 数据格式和字段映射

### 标准字段映射

| 原始字段 | OTLP 字段 | VictoriaLogs 字段 | 说明 |
|----------|-----------|------------------|------|
| `level` | `level` | `level` | 日志级别（小写） |
| `timestamp` | `TimeUnixNano` | `@timestamp` | 时间戳 |
| `message` | `Body` | `_msg` | 日志消息 |
| `caller` | `attributes.caller` | `caller` | 调用位置 |
| `trace_id` | `attributes.trace_id` | `trace_id` | 追踪ID |

### 资源属性

```go
// 自动添加的资源属性
resource := &resourcev1.Resource{
    Attributes: []*commonv1.KeyValue{
        {Key: "service.name", Value: "kart-io-logger"},
        {Key: "service.version", Value: "1.0.0"},
        {Key: "job", Value: "kart-io-logger"},          // VictoriaLogs
        {Key: "instance", Value: "localhost"},          // VictoriaLogs
    },
}
```

### 类型转换支持

```go
// 支持的 Go 类型自动转换
attributes := map[string]interface{}{
    "string_field":    "text",              // → StringValue
    "int_field":       42,                  // → IntValue
    "int64_field":     int64(42),          // → IntValue
    "float_field":     3.14,               // → DoubleValue
    "bool_field":      true,               // → BoolValue
    "time_field":      time.Now(),         // → StringValue (RFC3339)
    "complex_field":   struct{Name string}{"test"}, // → StringValue (JSON)
}
```

## 监控和调试

### 调试信息

启用调试模式查看传输详情：

```go
// 设置开发模式查看调试信息
opt := &option.LogOption{
    Development:  true,                    // 启用调试信息
    OTLPEndpoint: "http://localhost:4317",
}

logger, _ := logger.New(opt)

// 输出示例：
// 🔍 OTLP Request Debug:
//   Resource attributes: 4
//     [0] service.name = kart-io-logger
//     [1] service.version = 1.0.0
//     [2] job = kart-io-logger
//     [3] instance = localhost
//   Log record:
//     Timestamp: 1693834567123456789
//     Severity: INFO (9)
//     Body: 用户登录成功
//     Attributes: 3
//       [0] level = info
//       [1] @timestamp = 2023-09-04T12:34:56.123456789Z
//       [2] _msg = 用户登录成功
// ✅ gRPC OTLP export successful: http://localhost:4317
```

### 错误处理

```go
// 设置错误回调监控传输失败
factory := factory.NewLoggerFactory(opt)
factory.SetErrorCallback(func(err *errors.LoggerError) {
    if err.Component == "otlp" {
        // OTLP 传输失败处理
        fmt.Printf("OTLP 错误: %v\n", err)
        
        // 可以触发告警或降级处理
        alerting.SendAlert("otlp_failure", err.Error())
    }
})
```

### 健康检查

```go
func otlpHealthCheck() error {
    // 创建测试提供者
    opt := &option.OTLPOption{
        Enabled:  true,
        Endpoint: "http://localhost:4317",
        Protocol: "grpc",
        Timeout:  5 * time.Second,
    }
    
    provider, err := otlp.NewLoggerProvider(context.Background(), opt)
    if err != nil {
        return fmt.Errorf("OTLP 不可用: %w", err)
    }
    defer provider.Shutdown(context.Background())
    
    // 发送测试日志
    testAttrs := map[string]interface{}{
        "health_check": true,
        "timestamp":    time.Now(),
    }
    
    return provider.SendLogRecord(core.InfoLevel, "健康检查", testAttrs)
}
```

## 性能优化

### 批量发送

```go
// 当前实现是同步单条发送
// 生产环境建议通过 OpenTelemetry Collector 进行缓冲
```

### 资源池化

```go
// gRPC 连接自动复用
// HTTP 客户端使用连接池
httpClient := &http.Client{
    Timeout: opt.Timeout,
    Transport: &http.Transport{
        MaxIdleConns:    100,
        IdleConnTimeout: 90 * time.Second,
    },
}
```

## 故障处理

### 连接失败处理

```go
// OTLP 传输失败不会影响本地日志输出
logger.Infow("重要消息", "key", "value")
// ✅ 本地日志正常输出
// ❌ OTLP 传输失败（静默失败，不影响应用）
```

### 超时配置

```go
opt := &option.LogOption{
    OTLP: &option.OTLPOption{
        Timeout: 5 * time.Second,  // 推荐短超时避免阻塞
        // 超时后自动失败，不影响应用性能
    },
}
```

### 降级策略

```go
// 配置 OTLP 失败时的降级行为
opt := &option.LogOption{
    Engine:       "zap",
    Level:        "info",
    OutputPaths:  []string{"stdout", "/var/log/app.log"}, // 本地备份
    OTLPEndpoint: "http://otlp-collector:4317",
}

// OTLP 不可用时，日志仍输出到 stdout 和文件
```

## API 参考

### 主要函数

| 函数 | 描述 |
|------|------|
| `NewLoggerProvider(ctx, opt)` | 创建 OTLP 日志提供者 |
| `NewOTLPClient(opt)` | 创建 OTLP 客户端 |

### 提供者方法

| 方法 | 描述 |
|------|------|
| `SendLogRecord(level, msg, attrs)` | 发送单条日志记录 |
| `Shutdown(ctx)` | 优雅关闭连接 |
| `ForceFlush(ctx)` | 强制刷新缓冲区 |

### 客户端方法

| 方法 | 描述 |
|------|------|
| `Export(ctx, req)` | 导出日志数据 |
| `exportGRPC(ctx, req)` | gRPC 协议导出 |
| `exportHTTP(ctx, req)` | HTTP 协议导出 |

## 注意事项

1. **同步发送**：当前实现为同步发送，可能影响性能，生产环境建议通过 Collector 缓冲
2. **错误静默**：OTLP 发送失败不会中断应用，但会输出调试信息
3. **资源清理**：使用完毕后调用 `Shutdown()` 清理 gRPC 连接
4. **类型支持**：复杂类型会序列化为 JSON 字符串
5. **时区处理**：所有时间字段统一转换为 UTC

## 相关包

- [`option`](../option/) - OTLP 配置选项
- [`core`](../core/) - 日志级别定义
- [`factory`](../factory/) - 日志器工厂集成