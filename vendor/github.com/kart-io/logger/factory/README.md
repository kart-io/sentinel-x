# Factory Package

工厂包提供日志器的创建、管理和错误处理功能，是连接配置和具体引擎实现的核心组件。

## 概述

`factory` 包实现了以下关键功能：

- **工厂模式**：根据配置动态创建不同引擎的日志器实例
- **错误处理**：集成错误处理机制，支持重试和优雅降级
- **引擎抽象**：屏蔽具体引擎细节，提供统一的创建接口
- **动态配置**：支持运行时配置更新和重新创建
- **故障转移**：引擎创建失败时的自动降级机制

## 核心类型

### LoggerFactory

```go
type LoggerFactory struct {
    option       *option.LogOption
    errorHandler *errors.ErrorHandler
}
```

工厂管理配置和错误处理，负责创建和管理日志器实例。

## 创建工厂

### 基本创建

```go
func NewLoggerFactory(opt *option.LogOption) *LoggerFactory
```

### 带错误处理的创建

```go
func NewLoggerFactoryWithErrorHandler(opt *option.LogOption, errorHandler *errors.ErrorHandler) *LoggerFactory
```

## 使用方式

### 1. 最简单的使用（推荐）

```go
package main

import (
    "github.com/kart-io/logger"
    "github.com/kart-io/logger/option"
)

func main() {
    // 方式1：使用根包的便利函数（推荐）
    logger, err := logger.New(option.DefaultLogOption())
    if err != nil {
        panic(err)
    }
    
    logger.Info("Hello, World!")
    
    // 方式2：使用默认配置
    logger2, err := logger.NewWithDefaults()
    if err != nil {
        panic(err)
    }
    
    // 方式3：全局日志器
    logger.SetGlobal(logger2)
    logger.Info("使用全局日志器") // 直接调用包级函数
}
```

### 2. 使用工厂模式

```go
package main

import (
    "github.com/kart-io/logger/factory"
    "github.com/kart-io/logger/option"
)

func main() {
    // 创建配置
    opt := &option.LogOption{
        Engine: "zap",
        Level:  "debug",
        Format: "json",
    }
    
    // 创建工厂
    f := factory.NewLoggerFactory(opt)
    
    // 创建日志器
    logger, err := f.CreateLogger()
    if err != nil {
        panic(err)
    }
    
    // 使用日志器
    logger.Info("日志器创建成功")
    logger.Debugw("调试信息", "user_id", "12345", "action", "login")
}
```

### 3. 带错误处理的创建

```go
package main

import (
    "context"
    "github.com/kart-io/logger/factory"
    "github.com/kart-io/logger/option"
    "github.com/kart-io/logger/errors"
)

func main() {
    opt := option.DefaultLogOption()
    
    // 创建自定义错误处理器
    errorHandler := errors.NewErrorHandler(&errors.RetryPolicy{
        MaxRetries:    3,
        BackoffPolicy: errors.BackoffExponential,
    })
    
    // 创建工厂
    f := factory.NewLoggerFactoryWithErrorHandler(opt, errorHandler)
    
    // 带上下文创建日志器
    ctx := context.Background()
    logger, err := f.CreateLoggerWithContext(ctx)
    if err != nil {
        // 处理创建错误
        stats := f.GetErrorStats()
        fmt.Printf("创建失败，错误统计: %+v\n", stats)
        
        // 获取详细错误信息
        lastErrors := f.GetLastErrors()
        for component, err := range lastErrors {
            fmt.Printf("组件 %s 错误: %v\n", component, err)
        }
        return
    }
    
    logger.Info("日志器创建成功")
}
```

## 动态配置更新

### 运行时配置变更

```go
package main

import (
    "github.com/kart-io/logger/factory"
    "github.com/kart-io/logger/option"
)

func dynamicConfigExample() {
    // 初始配置
    opt := &option.LogOption{
        Engine: "slog",
        Level:  "info",
        Format: "json",
    }
    
    f := factory.NewLoggerFactory(opt)
    logger, _ := f.CreateLogger()
    
    logger.Info("初始配置日志")
    
    // 运行时更新配置
    newOpt := &option.LogOption{
        Engine: "zap",
        Level:  "debug",
        Format: "console",
    }
    
    // 更新工厂配置
    if err := f.UpdateOption(newOpt); err != nil {
        panic(err)
    }
    
    // 重新创建日志器（应用新配置）
    newLogger, _ := f.CreateLogger()
    newLogger.Debug("新配置的调试日志") // 现在可以显示调试日志
}
```

### 配置重载示例

```go
func configReloadExample() {
    f := factory.NewLoggerFactory(option.DefaultLogOption())
    
    // 设置配置变更回调
    f.SetErrorCallback(func(err *errors.LoggerError) {
        fmt.Printf("配置变更错误: %v\n", err)
    })
    
    // 模拟配置文件变更
    for {
        // 检测配置文件变化
        if configChanged() {
            newOpt := loadConfigFromFile()
            if err := f.UpdateOption(newOpt); err != nil {
                fmt.Printf("配置更新失败: %v\n", err)
            } else {
                fmt.Println("配置已更新")
            }
        }
        time.Sleep(5 * time.Second)
    }
}
```

## 错误处理和监控

### 错误统计

```go
func errorHandlingExample() {
    f := factory.NewLoggerFactory(option.DefaultLogOption())
    
    // 尝试创建日志器
    logger, err := f.CreateLogger()
    if err != nil {
        // 获取错误统计
        stats := f.GetErrorStats()
        fmt.Printf("错误统计: %+v\n", stats)
        
        // 输出示例: map[factory:2 zap:1 slog:1]
    }
    
    // 获取最近的错误详情
    lastErrors := f.GetLastErrors()
    for component, lastErr := range lastErrors {
        fmt.Printf("组件: %s, 最后错误: %v\n", component, lastErr)
    }
    
    // 重置错误统计
    f.ResetErrors()
}
```

### 设置回退日志器

```go
func fallbackLoggerExample() {
    f := factory.NewLoggerFactory(option.DefaultLogOption())
    
    // 创建一个简单的回退日志器
    fallbackLogger, _ := slog.NewSlogLogger(option.DefaultLogOption())
    f.SetFallbackLogger(fallbackLogger)
    
    // 即使主日志器创建失败，也能获得回退日志器
    logger, err := f.CreateLogger()
    if err != nil {
        fmt.Printf("使用回退日志器: %v\n", err)
    }
    
    // logger 现在是回退日志器，可以正常使用
    logger.Info("来自回退日志器的消息")
}
```

## 引擎选择和故障转移

### 自动故障转移

工厂支持引擎间的自动故障转移：

```go
// 配置为使用 zap 引擎
opt := &option.LogOption{
    Engine: "zap",
    // ... 其他配置
}

f := factory.NewLoggerFactory(opt)
logger, err := f.CreateLogger()

// 如果 zap 引擎创建失败，工厂会自动尝试 slog 引擎
// 确保应用在一种引擎不可用时仍能正常工作
```

### 引擎兼容性

| 引擎 | 性能 | 特性 | 适用场景 |
|------|------|------|----------|
| **Zap** | 高性能 | 丰富功能 | 生产环境，性能敏感场景 |
| **Slog** | 标准性能 | 标准接口 | Go 1.21+，标准化需求 |

## 生产环境最佳实践

### 1. 配置验证

```go
func productionSetup() {
    opt := &option.LogOption{
        Engine: "zap",
        Level:  "info",
        Format: "json",
        OutputPaths: []string{"stdout", "/var/log/app.log"},
        // OTLP 配置
        OTLPEndpoint: "http://jaeger:4317",
    }
    
    // 验证配置
    if err := opt.Validate(); err != nil {
        log.Fatalf("无效配置: %v", err)
    }
    
    f := factory.NewLoggerFactory(opt)
    logger, err := f.CreateLogger()
    if err != nil {
        log.Fatalf("日志器创建失败: %v", err)
    }
    
    // 设置为全局日志器
    logger.SetGlobal(logger)
}
```

### 2. 错误监控集成

```go
func monitoringIntegration() {
    f := factory.NewLoggerFactory(option.DefaultLogOption())
    
    // 设置错误监控回调
    f.SetErrorCallback(func(err *errors.LoggerError) {
        // 发送到监控系统（如 Prometheus、Sentry 等）
        metrics.IncrementCounter("logger.errors", 1, map[string]string{
            "component": err.Component,
            "type":      string(err.Type),
        })
        
        // 严重错误时发送告警
        if err.Type == errors.EngineError {
            alerting.SendAlert("logger_engine_failure", err.Error())
        }
    })
}
```

### 3. 健康检查

```go
func healthCheckExample() http.HandlerFunc {
    f := factory.NewLoggerFactory(option.DefaultLogOption())
    
    return func(w http.ResponseWriter, r *http.Request) {
        // 检查日志器是否正常工作
        logger, err := f.CreateLogger()
        if err != nil {
            http.Error(w, "日志器不可用", http.StatusServiceUnavailable)
            return
        }
        
        // 测试日志记录
        logger.Debug("健康检查")
        
        // 检查错误统计
        stats := f.GetErrorStats()
        if len(stats) > 0 {
            w.WriteHeader(http.StatusPartialContent)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "status": "degraded",
                "errors": stats,
            })
            return
        }
        
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    }
}
```

## API 参考

### 工厂方法

| 方法 | 描述 |
|------|------|
| `NewLoggerFactory(opt)` | 创建基本工厂实例 |
| `NewLoggerFactoryWithErrorHandler(opt, handler)` | 创建带错误处理的工厂实例 |

### 日志器创建

| 方法 | 描述 |
|------|------|
| `CreateLogger()` | 创建日志器实例 |
| `CreateLoggerWithContext(ctx)` | 带上下文创建日志器实例 |

### 配置管理

| 方法 | 描述 |
|------|------|
| `GetOption()` | 获取当前配置 |
| `UpdateOption(opt)` | 更新配置（支持动态变更） |

### 错误处理

| 方法 | 描述 |
|------|------|
| `GetErrorHandler()` | 获取错误处理器 |
| `GetErrorStats()` | 获取错误统计信息 |
| `GetLastErrors()` | 获取最近的错误详情 |
| `ResetErrors()` | 重置错误统计 |
| `SetErrorCallback(callback)` | 设置错误回调函数 |
| `SetFallbackLogger(logger)` | 设置回退日志器 |

## 注意事项

1. **线程安全**：工厂实例本身是线程安全的，但建议每个应用使用单一工厂实例
2. **配置验证**：创建日志器前会自动验证配置，无效配置会返回错误
3. **故障转移**：引擎创建失败会自动尝试另一个引擎，确保可用性
4. **内存管理**：工厂不会自动清理创建的日志器，需要应用层管理生命周期
5. **动态配置**：`UpdateOption` 只更新工厂配置，需要重新调用 `CreateLogger` 应用新配置

## 相关包

- [`core`](../core/) - 核心接口定义
- [`option`](../option/) - 配置选项管理
- [`engines/zap`](../engines/zap/) - Zap 引擎实现
- [`engines/slog`](../engines/slog/) - Slog 引擎实现
- [`errors`](../errors/) - 错误处理机制