# 中间件优先级机制

## 概述

中间件优先级机制提供了一种自动化的方式来管理中间件的执行顺序，避免手动管理中间件顺序时可能出现的错误。

## 核心概念

### 优先级常量

系统预定义了一组优先级常量，数值越大优先级越高，越先执行：

```go
const (
    PriorityRecovery          Priority = 1000  // 最高优先级，必须第一个执行以捕获所有 panic
    PriorityRequestID         Priority = 900   // 为其他中间件提供唯一请求 ID
    PriorityLogger            Priority = 800   // 依赖 RequestID，记录请求日志
    PriorityMetrics           Priority = 700   // 观测性中间件，收集性能指标
    PriorityTracing           Priority = 650   // 分布式追踪中间件
    PriorityCORS              Priority = 600   // 跨域资源共享，安全相关
    PrioritySecurityHeaders   Priority = 550   // 安全响应头设置
    PriorityTimeout           Priority = 500   // 请求超时控制，弹性机制
    PriorityAuth              Priority = 400   // 身份认证，必须在业务逻辑前执行
    PriorityAuthz             Priority = 300   // 授权检查，在认证后执行
    PriorityCustom            Priority = 100   // 自定义中间件的默认优先级
)
```

### 中间件执行顺序

基于上述优先级，实际执行顺序为：

```
请求 → Recovery → RequestID → Logger → Metrics → Tracing → CORS → SecurityHeaders
     → Timeout → Auth → Authz → Custom → 业务逻辑
```

## 使用方法

### 基本用法

```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
    "github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// 创建注册器
registrar := middleware.NewRegistrar()

// 注册中间件（自动按优先级排序）
registrar.Register("recovery", middleware.PriorityRecovery, recoveryMiddleware)
registrar.Register("auth", middleware.PriorityAuth, authMiddleware)
registrar.Register("logger", middleware.PriorityLogger, loggerMiddleware)

// 应用到路由器（自动按优先级顺序应用）
registrar.Apply(router)
```

### 条件注册

使用 `RegisterIf` 方法进行条件注册：

```go
// 仅在启用认证时注册
registrar.RegisterIf(enableAuth, "auth", middleware.PriorityAuth, authMiddleware)

// 仅在开发环境注册调试中间件
registrar.RegisterIf(isDev, "debug", middleware.PriorityCustom, debugMiddleware)
```

### 自定义优先级

对于自定义中间件，可以使用预定义优先级或自定义数值：

```go
// 使用预定义的默认优先级
registrar.Register("custom", middleware.PriorityCustom, customMiddleware)

// 使用自定义优先级（在 Auth 和 Authz 之间）
registrar.Register("custom", middleware.Priority(350), customMiddleware)
```

### 同优先级中间件

当多个中间件具有相同优先级时，按注册顺序执行：

```go
registrar.Register("custom1", middleware.PriorityCustom, customMiddleware1)
registrar.Register("custom2", middleware.PriorityCustom, customMiddleware2)
registrar.Register("custom3", middleware.PriorityCustom, customMiddleware3)

// 执行顺序：custom1 → custom2 → custom3
```

## 实际应用示例

### HTTP 服务器集成

在 `pkg/infra/server/transport/http/server.go` 中的使用示例：

```go
func (s *Server) applyMiddleware(router transport.Router, opts *mwopts.Options) {
    // 创建中间件注册器
    registrar := middleware.NewRegistrar()

    // Recovery middleware (最高优先级)
    registrar.RegisterIf(
        opts.IsEnabled(mwopts.MiddlewareRecovery),
        "recovery",
        middleware.PriorityRecovery,
        resilience.RecoveryWithOptions(*opts.Recovery, opts.Recovery.OnPanic),
    )

    // RequestID middleware (为其他中间件提供 RequestID)
    registrar.RegisterIf(
        opts.IsEnabled(mwopts.MiddlewareRequestID),
        "request-id",
        middleware.PriorityRequestID,
        middleware.RequestIDWithOptions(*opts.RequestID, opts.RequestID.Generator),
    )

    // Logger middleware (依赖 RequestID)
    registrar.RegisterIf(
        opts.IsEnabled(mwopts.MiddlewareLogger),
        "logger",
        middleware.PriorityLogger,
        observability.LoggerWithOptions(*opts.Logger, opts.Logger.Output),
    )

    // Auth middleware (在业务逻辑前执行)
    registrar.RegisterIf(
        opts.IsEnabled(mwopts.MiddlewareAuth) && opts.Auth.Authenticator != nil,
        "auth",
        middleware.PriorityAuth,
        authMiddleware,
    )

    // 按优先级顺序应用所有中间件
    registrar.Apply(router)
}
```

## 调试和诊断

### 查看中间件列表

```go
registrar := middleware.NewRegistrar()
// ... 注册中间件 ...

// 获取按优先级排序的中间件名称列表
names := registrar.List()
for _, name := range names {
    fmt.Println(name)  // 输出：recovery[1000], logger[800], auth[400], ...
}
```

### 查看中间件数量

```go
count := registrar.Count()
fmt.Printf("已注册 %d 个中间件\n", count)
```

## 优势

1. **自动排序**：无需手动管理中间件顺序，减少人为错误
2. **明确依赖**：通过优先级清晰表达中间件之间的依赖关系
3. **易于扩展**：新增中间件只需指定优先级，无需调整其他中间件
4. **可读性强**：代码意图清晰，易于理解和维护
5. **类型安全**：编译时检查，避免运行时错误

## 最佳实践

1. **使用预定义优先级**：尽量使用系统预定义的优先级常量
2. **避免冲突**：自定义优先级时，避免与系统优先级冲突
3. **明确命名**：为中间件指定有意义的名称，便于调试
4. **条件注册**：使用 `RegisterIf` 简化条件判断逻辑
5. **集中管理**：在一个地方统一管理所有中间件注册

## 注意事项

1. **线程安全**：注册器内部使用 `sync.RWMutex` 保证线程安全
2. **不可变性**：`Apply` 后不应修改注册器，避免不一致
3. **性能开销**：排序操作在 `Apply` 时执行一次，启动时开销可接受
4. **优先级范围**：建议使用 0-10000 范围内的优先级值

## 迁移指南

### 从手动顺序迁移

**旧方式：**

```go
router.Use(recoveryMiddleware)
router.Use(requestIDMiddleware)
router.Use(loggerMiddleware)
router.Use(authMiddleware)
```

**新方式：**

```go
registrar := middleware.NewRegistrar()
registrar.Register("recovery", middleware.PriorityRecovery, recoveryMiddleware)
registrar.Register("request-id", middleware.PriorityRequestID, requestIDMiddleware)
registrar.Register("logger", middleware.PriorityLogger, loggerMiddleware)
registrar.Register("auth", middleware.PriorityAuth, authMiddleware)
registrar.Apply(router)
```

## 测试

项目包含完整的单元测试和集成测试：

```bash
# 运行中间件优先级测试
go test -v ./pkg/infra/middleware -run "Priority"

# 运行所有中间件测试
go test -v ./pkg/infra/middleware

# 运行 HTTP 服务器测试
go test -v ./pkg/infra/server/transport/http
```

## 相关文件

- `pkg/infra/middleware/priority.go`：优先级机制实现
- `pkg/infra/middleware/priority_test.go`：单元测试
- `pkg/infra/server/transport/http/server.go`：HTTP 服务器集成
- `.claude/context-summary-middleware-priority.md`：上下文摘要

## 参考

- [Middleware Documentation](../README.md)
- [HTTP Server Documentation](../../server/transport/http/README.md)
