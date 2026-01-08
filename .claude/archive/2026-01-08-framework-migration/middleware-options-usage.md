# 中间件 Options API 使用指南

## 概述

本指南说明如何使用新的 `WithOptions` API 来配置中间件，实现配置与运行时依赖的分离。

## 背景

在配置中心场景（Nacos、Consul、etcd）中，配置数据必须可序列化（JSON/YAML）。传统的中间件配置包含运行时依赖（如函数指针），无法序列化。新的 `WithOptions` API 解决了这个问题。

## 对比：旧 API vs 新 API

### Logger 中间件

**旧方式（Config API）**：
```go
import "github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"

mw := observability.LoggerWithConfig(observability.LoggerConfig{
    SkipPaths:           []string{"/health"},
    UseStructuredLogger: true,
    Output:              log.Printf, // 函数指针，无法序列化
})
```

**新方式（Options API）**：
```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
    mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// 配置（可序列化）
opts := mwopts.LoggerOptions{
    SkipPaths:           []string{"/health"},
    UseStructuredLogger: true,
    // 不包含 Output 字段
}

// 运行时依赖注入
mw := observability.LoggerWithOptions(opts, log.Printf)

// 或使用默认值
mw := observability.LoggerWithOptions(opts, nil) // 使用 log.Printf
```

### Recovery 中间件

**旧方式（Config API）**：
```go
import "github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"

mw := resilience.RecoveryWithConfig(resilience.RecoveryConfig{
    EnableStackTrace: false,
    OnPanic: func(ctx transport.Context, err interface{}, stack []byte) {
        // 自定义 panic 处理
        alerting.SendAlert(err, stack)
    },
})
```

**新方式（Options API）**：
```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
    mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// 配置（可序列化）
opts := mwopts.RecoveryOptions{
    EnableStackTrace: false,
    // 不包含 OnPanic 字段
}

// 运行时依赖注入
panicHandler := func(ctx transport.Context, err interface{}, stack []byte) {
    // 自定义 panic 处理
    alerting.SendAlert(err, stack)
}

mw := resilience.RecoveryWithOptions(opts, panicHandler)

// 或使用默认值
mw := resilience.RecoveryWithOptions(opts, nil) // 仅记录日志
```

### RequestID 中间件

**旧方式（Config API）**：
```go
import "github.com/kart-io/sentinel-x/pkg/infra/middleware"

mw := middleware.RequestIDWithConfig(middleware.RequestIDConfig{
    Header: "X-Request-ID",
    Generator: func() string {
        return uuid.New().String()
    },
})
```

**新方式（Options API）**：
```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
    mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// 配置（可序列化）
opts := mwopts.RequestIDOptions{
    Header: "X-Request-ID",
    // 不包含 Generator 字段
}

// 运行时依赖注入
generator := func() string {
    return uuid.New().String()
}

mw := middleware.RequestIDWithOptions(opts, generator)

// 或使用默认值
mw := middleware.RequestIDWithOptions(opts, nil) // 使用 16 字节随机十六进制
```

## 配置中心集成示例

### Nacos 配置示例

**配置文件（nacos-config.yaml）**：
```yaml
middleware:
  logger:
    skip-paths:
      - /health
      - /metrics
    use-structured-logger: true

  recovery:
    enable-stack-trace: false

  request-id:
    header: X-Request-ID
```

**代码实现**：
```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
    "github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
    "github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
    mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// 从 Nacos 读取配置
type MiddlewareConfig struct {
    Logger    mwopts.LoggerOptions    `yaml:"logger"`
    Recovery  mwopts.RecoveryOptions  `yaml:"recovery"`
    RequestID mwopts.RequestIDOptions `yaml:"request-id"`
}

func setupMiddleware(cfg MiddlewareConfig) {
    // 1. Logger 中间件
    loggerMW := observability.LoggerWithOptions(cfg.Logger, nil)
    router.Use(loggerMW)

    // 2. Recovery 中间件（可选：自定义 panic 处理器）
    panicHandler := func(ctx transport.Context, err interface{}, stack []byte) {
        // 发送告警
        alerting.SendPanicAlert(err, stack)
    }
    recoveryMW := resilience.RecoveryWithOptions(cfg.Recovery, panicHandler)
    router.Use(recoveryMW)

    // 3. RequestID 中间件（可选：使用 UUID）
    generator := func() string {
        return uuid.New().String()
    }
    requestIDMW := middleware.RequestIDWithOptions(cfg.RequestID, generator)
    router.Use(requestIDMW)
}
```

### 动态配置更新

**监听配置变更**：
```go
import "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"

func watchConfig(client config_client.IConfigClient) {
    // 监听配置变更
    err := client.ListenConfig(vo.ConfigParam{
        DataId: "middleware-config",
        Group:  "DEFAULT_GROUP",
        OnChange: func(namespace, group, dataId, data string) {
            // 解析新配置
            var cfg MiddlewareConfig
            yaml.Unmarshal([]byte(data), &cfg)

            // 重新应用中间件
            setupMiddleware(cfg)
        },
    })
}
```

## 最佳实践

### 1. 配置存储
- ✅ **推荐**：使用 Options 结构体存储配置
- ❌ **避免**：在 Options 中存储运行时依赖

```go
// ✅ 好的做法
type AppConfig struct {
    Middleware mwopts.Options `yaml:"middleware"`
}

// ❌ 坏的做法
type AppConfig struct {
    Middleware struct {
        Logger struct {
            Output func(string, ...interface{}) // 无法序列化
        }
    }
}
```

### 2. 运行时依赖注入
- ✅ **推荐**：在应用启动时注入运行时依赖
- ❌ **避免**：在配置中硬编码运行时依赖

```go
// ✅ 好的做法
func setupMiddleware(opts mwopts.LoggerOptions) {
    // 根据环境选择不同的输出
    var output func(string, ...interface{})
    if isProd() {
        output = productionLogger
    } else {
        output = devLogger
    }

    mw := observability.LoggerWithOptions(opts, output)
    router.Use(mw)
}

// ❌ 坏的做法
func setupMiddleware() {
    opts := mwopts.LoggerOptions{
        Output: log.Printf, // 硬编码，无法序列化
    }
}
```

### 3. 默认值处理
- ✅ **推荐**：利用 `nil` 参数使用默认值
- ❌ **避免**：重复定义默认值

```go
// ✅ 好的做法
mw := middleware.RequestIDWithOptions(opts, nil) // 使用默认生成器

// ❌ 坏的做法
mw := middleware.RequestIDWithOptions(opts, func() string {
    return requestutil.GenerateRequestID() // 重复定义默认值
})
```

### 4. 类型安全
- ✅ **推荐**：使用定义的类型别名
- ❌ **避免**：直接使用匿名函数类型

```go
// ✅ 好的做法
var panicHandler resilience.PanicHandler = func(ctx transport.Context, err interface{}, stack []byte) {
    // ...
}

// ❌ 坏的做法
var panicHandler func(interface{}, interface{}, []byte) = func(a, b interface{}, c []byte) {
    // ...
}
```

## 迁移指南

### 步骤1：识别需要迁移的代码
搜索旧 API 的使用：
```bash
grep -r "LoggerWithConfig\|RecoveryWithConfig\|RequestIDWithConfig" .
```

### 步骤2：更新导入
```go
// 添加 options 包导入
import mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
```

### 步骤3：分离配置和运行时依赖
```go
// 旧代码
config := middleware.LoggerConfig{
    SkipPaths: []string{"/health"},
    Output:    customLogger,
}
mw := observability.LoggerWithConfig(config)

// 新代码
opts := mwopts.LoggerOptions{
    SkipPaths: []string{"/health"},
}
mw := observability.LoggerWithOptions(opts, customLogger)
```

### 步骤4：测试验证
```bash
go test ./... -v
```

## 常见问题

### Q1: 旧 API 还能用吗？
**A**: 是的，旧 API（`WithConfig`）仍然可用且向后兼容。但我们建议新代码使用 `WithOptions` API。

### Q2: 何时必须使用新 API？
**A**: 当你需要将配置存储在配置中心（Nacos、Consul、etcd）时，必须使用新 API，因为配置必须可序列化。

### Q3: 如何处理默认值？
**A**: 将运行时依赖参数设置为 `nil`，中间件会使用合理的默认值。

### Q4: 性能有影响吗？
**A**: 没有。新 API 在运行时的性能与旧 API 完全相同，因为它们最终调用相同的核心逻辑。

## 总结

新的 `WithOptions` API 提供了：
- ✅ 配置可序列化（支持配置中心）
- ✅ 配置与运行时分离（更清晰的职责）
- ✅ 向后兼容（旧代码无需修改）
- ✅ 类型安全（明确的类型定义）
- ✅ 文档完善（详细的注释和示例）

建议所有新项目使用 `WithOptions` API，旧项目可以渐进式迁移。
