# Middleware Performance Benchmarks

本文档说明中间件性能基准测试的用法、结果解读和性能优化建议。

## 概述

`benchmark_test.go` 提供了完整的中间件性能基准测试套件，涵盖所有核心中间件及其组合场景。

## 运行基准测试

### 基本用法

```bash
# 运行所有基准测试
go test -bench=. -benchmem ./pkg/infra/middleware/

# 运行特定中间件的基准测试
go test -bench=BenchmarkLoggerMiddleware -benchmem ./pkg/infra/middleware/

# 运行中间件链基准测试
go test -bench=BenchmarkMiddlewareChain -benchmem ./pkg/infra/middleware/

# 增加迭代次数以获得更准确的结果
go test -bench=. -benchmem -benchtime=1000x ./pkg/infra/middleware/

# 运行并发基准测试
go test -bench=BenchmarkMiddlewareChainConcurrent -benchmem ./pkg/infra/middleware/
```

### 性能分析

```bash
# 生成 CPU profile
go test -bench=BenchmarkMiddlewareChain -cpuprofile=cpu.prof ./pkg/infra/middleware/

# 生成内存 profile
go test -bench=BenchmarkMiddlewareChain -memprofile=mem.prof ./pkg/infra/middleware/

# 分析 profile
go tool pprof cpu.prof
go tool pprof mem.prof
```

## 基准测试说明

### Logger 中间件

- **BenchmarkLoggerMiddleware**: 测试结构化日志记录性能
- **BenchmarkLoggerMiddlewareWithSkip**: 测试路径跳过优化效果

**关键指标**:

- ns/op: 每次请求耗时
- B/op: 每次请求内存分配
- allocs/op: 每次请求分配次数

### Recovery 中间件

- **BenchmarkRecoveryMiddleware**: 测试正常操作性能（无 panic）
- **BenchmarkRecoveryMiddlewareWithPanic**: 测试 panic 恢复性能

**性能特点**:

- 正常操作开销极低（仅 defer 成本）
- Panic 恢复涉及栈追踪收集，性能开销较大

### RequestID 中间件

- **BenchmarkRequestIDMiddleware**: 测试随机 ID 生成性能
- **BenchmarkRequestIDMiddlewareWithExisting**: 测试已有 ID 的处理性能
- **BenchmarkGenerateRequestID**: 测试纯 ID 生成性能

**优化要点**:

- 使用加密随机数生成器确保唯一性
- 16 字节 hex 编码确保足够的唯一性

### RateLimit 中间件

- **BenchmarkRateLimitMiddleware**: 测试内存限流器性能
- **BenchmarkRateLimitMiddlewareWithSkip**: 测试路径跳过优化
- **BenchmarkMemoryRateLimiterAllow**: 测试限流器核心算法性能

**性能考虑**:

- 内存限流器使用滑动窗口算法
- 适合单实例部署
- 高流量场景建议使用 Redis 限流器

### SecurityHeaders 中间件

- **BenchmarkSecurityHeadersMiddleware**: 测试安全头设置性能
- **BenchmarkSecurityHeadersMiddlewareWithHSTS**: 测试 HSTS 启用时的性能

**性能特点**:

- 开销非常低（仅设置响应头）
- HSTS 检查 HTTPS 连接时略有额外开销

### Timeout 中间件

- **BenchmarkTimeoutMiddleware**: 测试超时控制正常操作性能
- **BenchmarkTimeoutMiddlewareWithSkip**: 测试路径跳过优化
- **BenchmarkTimeoutMiddlewareWithDelay**: 测试有实际延迟时的性能

**性能开销**:

- 需要创建 goroutine 和 context
- 使用 buffered channel 防止 goroutine 泄漏
- 建议为长时间运行的路径跳过超时检查

### 中间件链基准测试

- **BenchmarkMiddlewareChain**: 完整中间件链性能
- **BenchmarkMiddlewareChainMinimal**: 最小中间件链（RequestID + Logger + Recovery）
- **BenchmarkMiddlewareChainProduction**: 生产环境优化配置
- **BenchmarkMiddlewareChainConcurrent**: 并发场景性能

**中间件链顺序**（从外到内）:

1. RequestID - 生成请求追踪 ID
2. Logger - 记录请求日志
3. Recovery - 捕获 panic
4. SecurityHeaders - 设置安全响应头
5. RateLimit - 限流控制
6. Timeout - 超时控制

### 内存分配基准测试

- **BenchmarkMiddlewareMemoryAllocation**: 各中间件的内存分配模式

**优化目标**:

- 减少热路径上的内存分配
- 复用对象（如使用 sync.Pool）
- 避免不必要的字符串拼接

## 性能指标解读

### 指标说明

- **ns/op**: 每次操作的纳秒数（越低越好）
- **B/op**: 每次操作分配的字节数（越低越好）
- **allocs/op**: 每次操作的内存分配次数（越低越好）

### 性能基准参考

基于测试结果的性能分层：

**优秀性能**（单个中间件）:

- ns/op: < 2000 ns
- B/op: < 7000 B
- allocs/op: < 30

**可接受性能**:

- ns/op: 2000-5000 ns
- B/op: 7000-15000 B
- allocs/op: 30-60

**需要优化**:

- ns/op: > 5000 ns
- B/op: > 15000 B
- allocs/op: > 60

**中间件链性能**（完整链路）:

- 最小链（3个中间件）: ~10000 ns/op
- 标准链（6个中间件）: ~20000 ns/op
- 预期 QPS（单核）: > 50000

## 优化建议

### 通用优化

1. **使用路径跳过**: 为健康检查等频繁路径配置 SkipPaths
2. **优化日志配置**: 生产环境禁用不必要的日志字段
3. **合理配置超时**: 避免为所有路径启用严格超时
4. **选择合适的限流器**: 单实例用内存限流器，分布式用 Redis

### Logger 中间件优化

```go
// 优化：跳过健康检查路径
LoggerWithConfig(LoggerConfig{
    SkipPaths: []string{"/health", "/metrics", "/ready"},
    UseStructuredLogger: true,
})
```

### RateLimit 中间件优化

```go
// 优化：提高限流阈值，减少检查频率
RateLimitWithConfig(RateLimitConfig{
    Limit:     1000,
    Window:    1 * time.Minute,
    SkipPaths: []string{"/health"},
})
```

### Timeout 中间件优化

```go
// 优化：为长时间运行的 API 跳过超时
TimeoutWithConfig(TimeoutConfig{
    Timeout:   30 * time.Second,
    SkipPaths: []string{"/upload", "/export"},
})
```

## 性能监控建议

### 生产环境监控

1. **响应时间监控**: 监控 P50、P95、P99 延迟
2. **内存使用监控**: 监控中间件的内存分配趋势
3. **错误率监控**: 监控 panic 恢复、限流拒绝等错误
4. **吞吐量监控**: 监控 QPS 和并发连接数

### 关键性能指标 (KPI)

- **P99 响应时间**: < 100ms（包含完整中间件链）
- **内存分配率**: < 10MB/s per 1000 QPS
- **Panic 恢复率**: < 0.01%
- **限流拒绝率**: < 5%（正常流量下）

## 基准测试开发指南

### 添加新的基准测试

```go
// BenchmarkNewMiddleware 测试新中间件性能
func BenchmarkNewMiddleware(b *testing.B) {
    middleware := NewMiddleware()

    handler := middleware(func(c transport.Context) {
        c.JSON(200, map[string]string{"status": "ok"})
    })

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        req := httptest.NewRequest(http.MethodGet, "/test", nil)
        w := httptest.NewRecorder()
        ctx := newMockContext(req, w)
        handler(ctx)
    }
}
```

### 基准测试最佳实践

1. **使用 b.ResetTimer()**: 排除初始化时间
2. **使用 b.ReportAllocs()**: 报告内存分配
3. **独立测试**: 每个基准测试独立运行，避免相互影响
4. **真实场景**: 模拟真实的请求场景
5. **并发测试**: 使用 b.RunParallel() 测试并发性能

## 常见问题

### Q: 为什么 Logger 中间件性能较低？

A: Logger 中间件涉及结构化日志输出，包含时间计算、字段格式化等操作。优化方法：

- 使用 SkipPaths 跳过频繁路径
- 减少日志字段数量
- 使用异步日志（如果可用）

### Q: RateLimit 中间件的内存使用较高？

A: 内存限流器需要存储时间戳历史。优化方法：

- 调整窗口大小和限流阈值
- 使用 Redis 限流器分担内存压力
- 定期清理过期数据

### Q: Timeout 中间件的 goroutine 开销如何优化？

A: Timeout 中间件必须使用 goroutine 实现异步超时控制。优化方法：

- 为快速 API 跳过超时检查
- 使用 buffered channel 防止泄漏
- 合理设置超时时间

## 相关文档

- [中间件开发指南](./README.md)
- [性能优化最佳实践](../../docs/PERFORMANCE.md)
- [生产部署指南](../../docs/DEPLOYMENT.md)

## 贡献指南

如需添加新的基准测试或优化现有测试，请遵循以下原则：

1. 每个基准测试独立运行
2. 测试真实场景，避免过度优化
3. 提供清晰的测试说明和性能指标
4. 更新本文档说明新增的基准测试

## 版本历史

- v1.0.0 (2025-12-10): 初始版本，包含所有核心中间件基准测试
