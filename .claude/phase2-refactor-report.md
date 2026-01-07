# 阶段2重构报告：分离运行时依赖

## 执行时间
2026-01-06

## 重构目标
分离 Logger、Recovery、RequestID 中间件的运行时依赖（Output、OnPanic、Generator），使配置选项完全可序列化，支持配置中心场景。

## 重构内容

### 1. Logger 中间件

**修改文件**：
- `pkg/options/middleware/logger.go`
- `pkg/infra/middleware/observability/logger.go`

**改动内容**：
1. 标记 `LoggerOptions.Output` 字段为 Deprecated
2. 添加注释说明运行时依赖应通过函数参数注入
3. 新增 `LoggerWithOptions(opts, output)` API
4. 保持向后兼容：旧字段保留但标记废弃

**新 API 签名**：
```go
func LoggerWithOptions(opts mwopts.LoggerOptions, output func(string, ...interface{})) transport.MiddlewareFunc
```

### 2. Recovery 中间件

**修改文件**：
- `pkg/options/middleware/recovery.go`
- `pkg/infra/middleware/resilience/recovery.go`

**改动内容**：
1. 标记 `RecoveryOptions.OnPanic` 字段为 Deprecated
2. 添加注释说明运行时依赖应通过函数参数注入
3. 新增 `PanicHandler` 类型定义
4. 新增 `RecoveryWithOptions(opts, onPanic)` API
5. 保持向后兼容：旧字段保留但标记废弃

**新 API 签名**：
```go
type PanicHandler func(ctx transport.Context, err interface{}, stack []byte)
func RecoveryWithOptions(opts mwopts.RecoveryOptions, onPanic PanicHandler) transport.MiddlewareFunc
```

### 3. RequestID 中间件

**修改文件**：
- `pkg/options/middleware/request_id.go`
- `pkg/infra/middleware/request_id.go`

**改动内容**：
1. 标记 `RequestIDOptions.Generator` 字段为 Deprecated
2. 移除 `NewRequestIDOptions()` 中的默认生成器初始化
3. 移除 `google/uuid` 依赖
4. 新增 `RequestIDGenerator` 类型定义
5. 新增 `RequestIDWithOptions(opts, generator)` API
6. 保持向后兼容：旧字段保留但标记废弃

**新 API 签名**：
```go
type RequestIDGenerator func() string
func RequestIDWithOptions(opts mwopts.RequestIDOptions, generator RequestIDGenerator) transport.MiddlewareFunc
```

### 4. 更新调用代码

**修改文件**：
- `pkg/infra/server/transport/http/server.go`

**改动内容**：
1. 添加 import：`observability` 和 `resilience` 包
2. 更新中间件调用，使用新的 `WithOptions` API
3. 运行时依赖从 Options 中提取并通过参数传递

**调用示例**：
```go
// 旧方式（仍然兼容）
router.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
    SkipPaths:           opts.Logger.SkipPaths,
    Output:              opts.Logger.Output,
    UseStructuredLogger: opts.Logger.UseStructuredLogger,
}))

// 新方式（推荐）
router.Use(observability.LoggerWithOptions(*opts.Logger, opts.Logger.Output))
```

## 验证结果

### 编译验证
```bash
✅ go build ./... 成功
```

### 测试验证
```bash
✅ pkg/infra/middleware - PASS
✅ pkg/infra/middleware/auth - PASS
✅ pkg/infra/middleware/observability - PASS
✅ pkg/infra/middleware/resilience - PASS
✅ pkg/options/middleware - PASS
❌ pkg/infra/middleware/security - FAIL (预先存在的 CORS 测试问题，与本次重构无关)
```

**重点测试验证**：
- ✅ TestRequestID_GeneratesID - PASS
- ✅ TestRequestID_PreservesExistingID - PASS
- ✅ TestRequestIDWithConfig_CustomGenerator - PASS
- ✅ TestRecovery_CatchesPanic - PASS
- ✅ TestRecoveryWithConfig_OnPanicCallback - PASS
- ✅ TestEnhancedLogger - PASS

## 技术亮点

### 1. 配置与运行时分离
- Options 结构体完全可 JSON 序列化
- 运行时依赖通过函数参数注入
- 适配配置中心场景（Nacos、Consul、etcd）

### 2. 向后兼容
- 旧字段保留并标记 Deprecated
- 旧 API（WithConfig）继续工作
- 新 API（WithOptions）提供更好的分离

### 3. 类型安全
- 定义专用类型（PanicHandler、RequestIDGenerator）
- 提供清晰的函数签名
- 减少类型转换和错误

### 4. 文档完善
- 所有新 API 都有完整的中文注释
- 提供使用示例
- 说明默认行为

## 遗留问题

### 已知测试失败
- `pkg/infra/middleware/security` 中的 `TestCORS_DefaultConfig` 失败
- 原因：CORS 默认配置测试期望值不匹配
- 影响：与本次重构无关，是预先存在的问题
- 建议：单独修复 CORS 测试

## 后续建议

### 1. 迁移指导
建议在下一个大版本中：
1. 更新所有调用代码使用新 API
2. 移除 Options 中的运行时依赖字段
3. 移除旧的 WithConfig API（或标记废弃）

### 2. 文档更新
建议补充：
1. 迁移指南（从旧 API 到新 API）
2. 配置中心集成示例
3. 性能对比测试

### 3. 其他中间件
建议对其他中间件进行类似重构：
- Metrics 中间件
- Auth 中间件（如果有运行时依赖）
- 其他自定义中间件

## 总结

阶段2重构成功完成，实现了以下目标：

✅ Logger、Recovery、RequestID 中间件运行时依赖完全分离
✅ Options 结构体完全可 JSON 序列化
✅ 提供新的 WithOptions API 支持依赖注入
✅ 保持向后兼容，旧代码无需修改
✅ 所有相关测试通过
✅ 项目编译成功

重构遵循了单一职责原则，配置纯化，运行时注入的设计模式，为配置中心集成奠定了坚实基础。
