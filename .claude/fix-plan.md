# 测试修复执行计划

生成时间: 2026-01-07 16:45

## 问题总结

基于代码审查报告和测试运行结果,发现以下问题:

### 严重问题(阻塞合并)
1. **60+ 测试编译失败** - 主要原因是测试代码仍使用旧的 `transport.Context` API
2. **端点注册功能缺失** - health/metrics/pprof/version 端点被注释掉
3. **类型安全隐患** - CircuitBreaker 的不安全类型断言

### 测试失败分类

#### 类别1: Handler测试(user-center)
文件:
- `internal/user-center/handler/api_test.go` (5个错误)
- `internal/user-center/handler/validation_test.go` (1个错误)

错误: `undefined: custom_http.NewRequestContext`

原因: `custom_http.NewRequestContext` 已随适配器层删除

修复策略: 使用 `gin` 的测试工具 `httptest` 替换

#### 类别2: 中间件测试
文件:
- `pkg/infra/middleware/benchmark_test.go` (多个错误)
- `pkg/infra/middleware/auth/auth_test.go` (1个错误)
- `pkg/infra/middleware/security/cors_test.go` (多个错误)
- `pkg/infra/middleware/observability/integration_test.go` (多个错误)
- `pkg/infra/middleware/observability/tracing_test.go` (多个错误)
- `pkg/infra/middleware/resilience/body_limit_test.go` (多个错误)

错误:
- `middleware(func(c transport.Context) {…}) (no value) used as value`
- `cannot use func(c transport.Context) {…} as *gin.Context value`

原因: 测试代码仍使用 `transport.Context` 和旧的中间件签名

修复策略:
1. 更新测试辅助函数使用 `gin.HandlerFunc`
2. 使用 `httptest.NewRecorder()` 和 `gin.CreateTestContext()`

#### 类别3: HTTP传输层测试
文件:
- `pkg/infra/server/transport/http/response_test.go` (9个错误)

错误:
- `undefined: bindForm`
- `undefined: NewRequestContext`

原因: 测试针对已删除的 `RequestContext` 和 `bindForm` 功能

修复策略: 删除此测试文件(测试已废弃功能)

#### 类别4: 响应工具测试
文件:
- `pkg/utils/errors/example_test.go` (9个错误)

错误: `cannot use c (variable of interface type transport.Context) as *gin.Context value`

原因: example 测试使用旧的 `transport.Context`

修复策略: 更新为使用 `*gin.Context`

## 修复优先级

### P0 - 立即修复(阻塞合并)
1. ✅ 删除废弃的测试文件 (15分钟)
   - `pkg/infra/server/transport/http/response_test.go`

2. ⚠️ 修复 Handler 测试 (2小时)
   - `internal/user-center/handler/api_test.go`
   - `internal/user-center/handler/validation_test.go`

3. ⚠️ 修复中间件测试 (4小时)
   - 所有中间件测试文件(7个文件)

4. ⚠️ 修复响应工具测试 (30分钟)
   - `pkg/utils/errors/example_test.go`

5. ⚠️ 修复 CircuitBreaker 类型断言 (1小时)
   - `pkg/infra/middleware/resilience/circuit_breaker.go`

6. ⚠️ 重新启用端点注册 (2小时)
   - 检查并恢复 health/metrics/pprof/version 端点

### P1 - 合并后立即处理
1. 清理废弃接口
2. 统一响应工具函数
3. 优化池容量配置
4. 运行性能基准测试

## 修复执行计划

### 阶段1: 快速清理(15分钟)
删除明确废弃的测试文件

### 阶段2: Handler测试修复(2小时)
更新 user-center handler 测试使用 gin 测试工具

### 阶段3: 中间件测试修复(4小时)
批量更新所有中间件测试

### 阶段4: 其他测试修复(30分钟)
修复 errors 包的 example 测试

### 阶段5: 代码质量修复(3小时)
修复 CircuitBreaker 类型断言和重新启用端点

## 预计总工时: 10.5小时

## 执行记录

### 2026-01-07 16:45 - 计划制定完成
- 识别4类测试问题
- 制定5阶段修复计划
- 估算总工时10.5小时
