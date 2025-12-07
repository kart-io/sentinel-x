# Zap Engine

基于 Uber Zap 的高性能日志引擎实现。专为高并发、低延迟场景设计，提供卓越的性能表现。

## 📋 特性

- ⚡ **极致性能**: 基于 Uber Zap，业界顶级性能
- 🚀 **零内存分配**: 在热路径上避免内存分配
- ✅ **结构化日志**: 完整的结构化日志记录支持
- ✅ **统一字段标准化**: 与其他引擎保持字段名称一致
- ✅ **多种输出格式**: JSON、控制台、文本格式
- ✅ **生产就绪**: 经过大规模生产环境验证
- ✅ **丰富配置**: 灵活的配置选项

## 🚀 快速使用

### 基础创建

```go
package main

import (
    "github.com/kart-io/logger/engines/zap"
    "github.com/kart-io/logger/option"
    "github.com/kart-io/logger/core"
)

func main() {
    // 创建高性能配置
    opt := &option.LogOption{
        Engine:      "zap",
        Level:       "info", 
        Format:      "json",
        OutputPaths: []string{"stdout"},
        Development: false, // 生产模式获得最佳性能
    }
    
    // 创建 zap 引擎
    logger, err := zap.NewZapLogger(opt)
    if err != nil {
        panic(err)
    }
    
    // 高性能日志记录
    logger.Info("Hello from Zap engine!")
}
```

### 三种日志风格

```go
// 1. 基础方法 - 高性能简单日志
logger.Info("系统启动完成")
logger.Error("连接失败", err)

// 2. 格式化方法 - 字符串格式化
logger.Infof("处理请求耗时: %dms", duration)
logger.Errorf("用户 %s 认证失败: %v", username, err)

// 3. 结构化方法 - 零分配结构化日志（推荐）
logger.Infow("API请求处理",
    "method", "POST",
    "path", "/api/users",
    "status_code", 200,
    "duration_ms", 45,
    "user_id", "12345",
)
```

## 📊 输出格式

### JSON 格式（推荐生产环境）

```go
opt := &option.LogOption{
    Engine: "zap",
    Format: "json",
    Development: false, // 生产模式
}
```

输出示例：
```json
{"level":"info","timestamp":"2025-08-30T13:45:30.123456789Z","caller":"main.go:25","message":"API请求处理","engine":"zap","method":"POST","path":"/api/users","status_code":200,"duration_ms":45}
```

### 开发者友好格式

```go
opt := &option.LogOption{
    Engine: "zap",
    Format: "console",
    Development: true, // 开发模式
}
```

输出示例：
```
2025-08-30T13:45:30.123+08:00   INFO    main.go:25      API请求处理     {"engine": "zap", "method": "POST", "path": "/api/users", "status_code": 200}
```

## 🔧 配置选项

### 高性能生产配置

```go
opt := &option.LogOption{
    Engine:            "zap",
    Level:             "info",           // 生产建议 info 或更高
    Format:            "json",           // JSON格式性能最佳
    OutputPaths:       []string{"stdout", "/var/log/app.log"},
    Development:       false,            // 生产模式，最佳性能
    DisableCaller:     false,            // 调用者信息
    DisableStacktrace: false,            // 错误堆栈跟踪
}
```

### 开发调试配置

```go  
opt := &option.LogOption{
    Engine:            "zap",
    Level:             "debug",          // 显示所有日志
    Format:            "console",        // 易读格式
    OutputPaths:       []string{"stdout"},
    Development:       true,             // 开发模式，更多信息
    DisableCaller:     false,            // 显示文件位置
    DisableStacktrace: false,            // 显示完整堆栈
}
```

## ⚡ 性能优化

### 零分配结构化日志

```go
// 推荐：使用 *w 方法，避免内存分配
logger.Infow("高性能日志",
    "key1", "value1",
    "key2", 42,
    "key3", time.Now(),
)

// 避免：字符串拼接和格式化在高并发场景下的开销
logger.Infof("用户 %s 操作 %s", user, action) // 会产生内存分配
```

### 条件日志记录

```go
// 高频调试日志的性能优化
if logger.Core().Enabled(zapcore.DebugLevel) {
    // 只有启用 debug 级别才执行昂贵的计算
    expensiveData := computeExpensiveDebugInfo()
    logger.Debugw("调试信息", "data", expensiveData)
}
```

### 批量字段复用

```go
// 创建带有通用字段的子日志器
requestLogger := logger.With(
    "request_id", requestID,
    "user_id", userID,
    "session_id", sessionID,
)

// 复用通用字段，避免重复传递
requestLogger.Info("请求开始")
requestLogger.Info("验证通过") 
requestLogger.Info("请求完成")
```

## 🎯 高级特性

### 动态级别调整

```go
// 运行时调整日志级别
logger.SetLevel(core.DebugLevel) // 开启调试
logger.SetLevel(core.ErrorLevel) // 只记录错误
```

### 调用者信息定制

```go
// 跳过包装函数，显示真实调用位置
func logWrapper(msg string) {
    logger.WithCallerSkip(1).Info(msg) // 跳过当前函数栈帧
}
```

### 上下文集成

```go
ctx := context.WithValue(context.Background(), "trace_id", "trace-123")

// 上下文感知日志（用于分布式追踪）
ctxLogger := logger.WithCtx(ctx)
ctxLogger.Info("开始处理请求")
```

## 🔍 内部实现

### 标准化编码器配置

Zap 引擎使用定制的编码器配置确保字段一致性：

```go
func createStandardizedEncoderConfig() zapcore.EncoderConfig {
    config := zap.NewProductionEncoderConfig()
    
    // 统一字段名称
    config.TimeKey = fields.TimestampField      // "timestamp"
    config.LevelKey = fields.LevelField         // "level"  
    config.MessageKey = fields.MessageField     // "message"
    config.CallerKey = fields.CallerField       // "caller"
    config.StacktraceKey = fields.StacktraceField // "stacktrace"
    
    // 小写级别输出
    config.EncodeLevel = zapcore.LowercaseLevelEncoder
    config.EncodeTime = zapcore.RFC3339NanoTimeEncoder
    
    return config
}
```

### 级别映射

```go
func mapToZapLevel(level core.Level) zapcore.Level {
    switch level {
    case core.DebugLevel: return zapcore.DebugLevel
    case core.InfoLevel:  return zapcore.InfoLevel  
    case core.WarnLevel:  return zapcore.WarnLevel
    case core.ErrorLevel: return zapcore.ErrorLevel
    case core.FatalLevel: return zapcore.FatalLevel
    default:              return zapcore.InfoLevel
    }
}
```

## 📊 性能基准

### 与其他引擎对比

| 操作 | Zap | Slog | 性能提升 |
|------|-----|------|----------|
| 简单日志 | 150ns/op | 300ns/op | **2x 更快** |
| 结构化日志 | 200ns/op | 450ns/op | **2.25x 更快** |
| 内存分配 | 0 allocs/op | 2 allocs/op | **零分配** |
| 高并发吞吐 | 8M ops/sec | 4M ops/sec | **2x 更高** |

### 基准测试命令

```bash
# 运行性能基准测试
go test github.com/kart-io/logger/engines/zap -bench=. -benchmem

# 与 slog 引擎对比
go test github.com/kart-io/logger/engines/... -bench=BenchmarkLogger -benchmem
```

## 🎯 适用场景

### 最适合

- 🚀 **高并发服务**: 微服务、API网关、消息队列
- ⚡ **性能敏感应用**: 游戏服务器、实时系统、HFT系统
- 📊 **大规模日志**: 每秒百万级日志记录
- 🔥 **热路径日志**: 频繁调用的关键代码路径

### 考虑其他选择

- 📱 简单应用或工具：可考虑 [slog 引擎](../slog/)
- 🧪 原型开发：标准库 slog 可能更简单
- 📝 对性能不敏感的场景

## 🧪 测试和调试

### 单元测试

```bash
# 运行 zap 引擎测试
go test github.com/kart-io/logger/engines/zap -v

# 测试覆盖率
go test github.com/kart-io/logger/engines/zap -cover -coverprofile=coverage.out
```

### 性能调试

```go
// 在代码中添加性能监控
import _ "net/http/pprof"

// 查看内存分配
go tool pprof http://localhost:6060/debug/pprof/allocs

// 查看 CPU 使用
go tool pprof http://localhost:6060/debug/pprof/profile
```

## 🔗 相关资源

- [Uber Zap 官方文档](https://pkg.go.dev/go.uber.org/zap)
- [Zap 性能最佳实践](https://github.com/uber-go/zap/blob/master/FAQ.md)
- [高性能日志设计原理](https://github.com/uber-go/zap/blob/master/benchmarks/README.md)
- [`engines/slog`](../slog/) - 标准库 slog 引擎对比
- [`example/performance`](../../example/performance/) - 性能对比示例

## ⚠️ 注意事项

### 性能相关

1. **开发vs生产**: `Development: false` 在生产环境获得最佳性能
2. **日志级别**: 生产环境建议使用 `info` 或更高级别
3. **格式选择**: JSON 格式比 console 格式性能更好
4. **字段类型**: 原生类型比接口类型性能更好

### 使用注意

1. **Fatal 行为**: `Fatal` 级别会调用 `os.Exit(1)`
2. **堆栈跟踪**: Error 和 Fatal 级别自动包含堆栈跟踪
3. **字段标准化**: 某些字段名会被标准化以保持一致性
4. **上下文支持**: `WithCtx` 主要用于追踪信息传递

## 🚀 最佳实践

### 性能优化

```go
// ✅ 推荐：结构化日志，零内存分配
logger.Infow("用户登录", "user_id", userID, "ip", clientIP)

// ❌ 避免：频繁字符串格式化
logger.Infof("用户 %s 从 %s 登录", userID, clientIP)
```

### 错误处理

```go
// ✅ 推荐：结构化错误记录
logger.Errorw("数据库操作失败", 
    "operation", "user_create",
    "error", err.Error(),
    "user_data", userData,
)

// ❌ 避免：丢失上下文信息
logger.Error("操作失败", err)
```

### 字段命名

```go
// ✅ 推荐：使用下划线命名
logger.Infow("请求处理", "user_id", uid, "request_time", reqTime)

// ❌ 避免：驼峰命名不统一
logger.Infow("请求处理", "userId", uid, "requestTime", reqTime)
```

选择 Zap 引擎，为你的应用提供工业级的高性能日志记录能力！ 🚀