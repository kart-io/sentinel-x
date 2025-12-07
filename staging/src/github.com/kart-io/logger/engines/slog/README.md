# Slog Engine

基于 Go 1.21+ 标准库 `log/slog` 的日志引擎实现。提供结构化日志记录，具有良好的兼容性和标准化支持。

## 📋 特性

- ✅ 基于 Go 标准库 `log/slog` 
- ✅ 完整的结构化日志支持
- ✅ 统一字段名称标准化  
- ✅ 多种输出格式（JSON、文本、控制台）
- ✅ 上下文感知日志记录
- ✅ 调用者信息和堆栈跟踪
- ✅ 自定义级别格式化（小写输出）
- ✅ 零配置默认设置

## 🚀 快速使用

### 基础创建

```go
package main

import (
    "github.com/kart-io/logger/engines/slog"
    "github.com/kart-io/logger/option"
    "github.com/kart-io/logger/core"
)

func main() {
    // 创建配置
    opt := &option.LogOption{
        Engine:      "slog",
        Level:       "info",
        Format:      "json",
        OutputPaths: []string{"stdout"},
    }
    
    // 创建 slog 引擎
    logger, err := slog.NewSlogLogger(opt)
    if err != nil {
        panic(err)
    }
    
    // 使用日志器
    logger.Info("Hello from slog engine!")
}
```

### 三种日志风格

```go
// 1. 基础方法 - 类似 fmt.Print
logger.Info("用户登录", "操作完成")
logger.Error("数据库连接失败", err)

// 2. 格式化方法 - 类似 fmt.Printf  
logger.Infof("用户 %s 在 %s 登录", username, time.Now().Format("15:04:05"))
logger.Errorf("处理请求失败: %v", err)

// 3. 结构化方法 - 键值对
logger.Infow("用户操作", 
    "action", "login",
    "user_id", "12345", 
    "ip", "192.168.1.1",
    "timestamp", time.Now(),
)
```

## 📊 输出格式

### JSON 格式

```go
opt := &option.LogOption{
    Engine: "slog",
    Format: "json",
    Level:  "debug",
}
```

输出示例：
```json
{"time":"2025-08-30T13:45:30.123456789Z","level":"info","msg":"用户登录成功","engine":"slog","user_id":"12345","action":"login"}
```

### 文本格式

```go
opt := &option.LogOption{
    Engine: "slog", 
    Format: "text", // 或 "console"
    Level:  "debug",
}
```

输出示例：
```
time=2025-08-30T13:45:30.123+08:00 level=info msg="用户登录成功" engine=slog user_id=12345 action=login
```

## 🔧 配置选项

### 基础配置

```go
opt := &option.LogOption{
    Engine:      "slog",           // 引擎名称
    Level:       "info",           // 日志级别
    Format:      "json",           // 输出格式: json, text, console
    OutputPaths: []string{"stdout"}, // 输出路径
    Development: false,            // 开发模式
    
    // 可选配置
    DisableCaller:     false,      // 禁用调用者信息
    DisableStacktrace: false,      // 禁用堆栈跟踪
}
```

### 输出路径

```go
// 标准输出/错误
OutputPaths: []string{"stdout"}
OutputPaths: []string{"stderr"}

// 文件输出
OutputPaths: []string{"/var/log/app.log"}

// 多个输出
OutputPaths: []string{"stdout", "/var/log/app.log"}
```

## 🎯 特色功能

### 字段标准化

slog 引擎通过 `standardizedHandler` 确保字段名称一致：

```go
// 自动标准化字段映射
ts -> timestamp
msg -> message  
trace.id -> trace_id
span_id -> span_id
```

### 上下文感知

```go
ctx := context.WithValue(context.Background(), "request_id", "req-123")

// 带上下文的日志
contextLogger := logger.WithCtx(ctx)
contextLogger.Info("处理请求开始")

// 添加持久化字段
persistentLogger := logger.With("service", "user-api", "version", "1.0.0")
persistentLogger.Info("服务启动")
```

### 调用者信息

```go
// 启用调用者信息（默认启用）
opt.DisableCaller = false

// 自定义调用者跳过层数
skipLogger := logger.WithCallerSkip(1)
skipLogger.Info("这将显示调用此函数的位置")
```

### 动态级别调整

```go
// 运行时修改级别
logger.SetLevel(core.DebugLevel)
logger.Debug("现在可以看到调试信息了")

logger.SetLevel(core.ErrorLevel) 
logger.Info("这条信息不会输出")
```

## 🧪 高级用法

### 错误处理和堆栈跟踪

```go
func handleRequest() {
    defer func() {
        if r := recover(); r != nil {
            // Fatal 级别自动包含堆栈跟踪
            logger.Fatal("请求处理发生致命错误", "error", r)
        }
    }()
    
    if err := processData(); err != nil {
        // Error 级别自动包含堆栈跟踪
        logger.Error("数据处理失败", "error", err)
        return
    }
    
    logger.Info("请求处理成功")
}
```

### 结构化数据记录

```go
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

user := &User{ID: "123", Name: "张三", Email: "zhangsan@example.com"}

logger.Infow("用户创建",
    "user", user,
    "operation", "create_user",
    "metadata", map[string]interface{}{
        "source": "api",
        "version": "v1",
    },
)
```

## 🔍 内部实现

### 标准化处理器

slog 引擎使用自定义的 `standardizedHandler` 包装标准 slog 处理器：

```go
type standardizedHandler struct {
    handler            slog.Handler
    mapper             *fields.FieldMapper
    disableCaller      bool
    disableStacktrace  bool
}
```

### 级别映射

```go
func mapToSlogLevel(level core.Level) slog.Level {
    switch level {
    case core.DebugLevel: return slog.LevelDebug
    case core.InfoLevel:  return slog.LevelInfo
    case core.WarnLevel:  return slog.LevelWarn
    case core.ErrorLevel: return slog.LevelError
    case core.FatalLevel: return slog.LevelError // Fatal映射为Error
    default:              return slog.LevelInfo
    }
}
```

## ⚡ 性能特征

### 优势

- ✅ **标准库稳定性**: 基于 Go 官方标准库
- ✅ **内存效率**: 合理的内存分配策略
- ✅ **兼容性好**: 与标准库生态系统完美集成
- ✅ **维护成本低**: 跟随 Go 版本更新

### 适用场景

- 🎯 需要标准库兼容性的项目
- 🎯 对性能要求适中的应用
- 🎯 希望使用官方标准的团队
- 🎯 需要长期稳定支持的项目

## 📋 测试

```bash
# 运行 slog 引擎测试
go test github.com/kart-io/logger/engines/slog -v

# 运行基准测试
go test github.com/kart-io/logger/engines/slog -bench=.

# 测试覆盖率
go test github.com/kart-io/logger/engines/slog -cover
```

## 🔗 相关资源

- [Go log/slog 官方文档](https://pkg.go.dev/log/slog)
- [Slog 最佳实践指南](https://golang.org/doc/tutorial/slog)
- [`engines/zap`](../zap/) - 高性能 Zap 引擎对比
- [`core`](../../core/) - 核心接口定义
- [`option`](../../option/) - 配置选项详解

## ⚠️ 注意事项

1. **Go 版本要求**: 需要 Go 1.21+ 支持
2. **Fatal 行为**: `Fatal` 级别会调用 `os.Exit(1)` 
3. **级别映射**: `Fatal` 级别在 slog 中映射为 `Error` 级别
4. **字段标准化**: 某些字段名会被自动标准化以保持一致性
5. **上下文传递**: `WithCtx` 方法主要用于追踪信息，不会自动提取上下文值

## 🚀 最佳实践

1. **结构化日志**: 优先使用 `*w` 方法进行结构化日志记录
2. **级别控制**: 生产环境建议使用 `info` 或更高级别
3. **字段命名**: 使用下划线命名法，如 `user_id`, `request_id`
4. **错误记录**: 使用 `Error` 级别记录可恢复错误，`Fatal` 仅用于不可恢复的致命错误
5. **性能考虑**: 对于高并发场景，考虑使用 [zap 引擎](../zap/) 获得更好性能