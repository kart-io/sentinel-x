# Sentinel-X 并发规范

## 概述

本文档定义了 Sentinel-X 项目中并发编程的规范和最佳实践。所有新的并发任务必须遵循本规范，使用统一的 ants 池管理，而非直接使用 `go` 关键字创建 goroutine。

## 核心原则

### 禁止直接使用 `go func()`

**以下场景禁止直接使用 `go` 关键字：**

1. HTTP 请求处理中的异步操作
2. 定时任务和后台清理
3. 并行健康检查
4. 回调执行
5. 任何可能高并发的场景

**唯一允许的例外：**

1. 服务启动时的监听 goroutine（如 HTTP/gRPC Server）
2. 需要长期运行的订阅 goroutine（如 Redis Pub/Sub）
3. 池不可用时的降级处理

### 为什么禁止直接创建 goroutine

1. **资源失控**：无限制创建 goroutine 会导致内存爆炸
2. **调度压力**：大量 goroutine 会增加 GC 和调度器压力
3. **难以监控**：散落的 goroutine 难以统计和追踪
4. **panic 处理**：缺乏统一的 panic 恢复机制

## 池类型说明

### 预定义池

| 池类型 | 用途 | 容量 | 模式 |
|--------|------|------|------|
| `DefaultPool` | 通用任务 | 1000 | 阻塞 |
| `HealthCheckPool` | 健康检查 | 100 | 阻塞 |
| `BackgroundPool` | 后台任务（清理、监控） | 50 | 阻塞 |
| `CallbackPool` | 回调执行 | 200 | 非阻塞 |
| `TimeoutPool` | 超时中间件 | 5000 | 阻塞 |

### 容量选择指南

```go
// 根据业务特性选择池容量
//
// 高并发低延迟（如 HTTP 请求处理）：
//   容量 = 预期 QPS × 平均处理时间(秒) × 2
//   例如：QPS=1000, 平均处理时间=50ms
//   容量 = 1000 × 0.05 × 2 = 100
//
// 低并发长任务（如后台清理）：
//   容量 = 预期并行任务数 × 1.5
//   例如：同时最多 20 个清理任务
//   容量 = 20 × 1.5 = 30
//
// 健康检查：
//   容量 = 存储实例数 × 2
//   例如：10 个存储实例
//   容量 = 10 × 2 = 20
```

## 使用方法

### 初始化

```go
package main

import (
    "github.com/kart-io/sentinel-x/pkg/infra/pool"
)

func main() {
    // 方式一：使用默认配置初始化
    if err := pool.InitGlobal(); err != nil {
        panic(err)
    }
    defer pool.CloseGlobal()

    // 方式二：使用自定义配置初始化
    config := pool.DefaultGlobalConfig()
    config.DefaultPool.Capacity = 2000
    config.CustomPools["my-pool"] = &pool.PoolConfig{
        Capacity:       500,
        ExpiryDuration: 30 * time.Second,
    }

    if err := pool.InitGlobalWithConfig(config); err != nil {
        panic(err)
    }
    defer pool.CloseGlobal()

    // 启动应用...
}
```

### 提交任务

```go
// 方式一：提交到默认池
err := pool.Submit(func() {
    // 业务逻辑
})

// 方式二：提交到指定类型的池
err := pool.SubmitToType(pool.HealthCheckPool, func() {
    // 健康检查逻辑
})

// 方式三：提交到自定义池
err := pool.SubmitTo("my-pool", func() {
    // 自定义任务
})

// 方式四：带上下文提交（支持取消）
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := pool.SubmitWithContext(ctx, func() {
    // 可被取消的任务
})
```

### 降级处理

当池不可用时，必须提供降级方案：

```go
// 推荐的降级模式
task := func() {
    // 业务逻辑
}

if err := pool.SubmitToType(pool.BackgroundPool, task); err != nil {
    // 降级：记录日志并使用 goroutine
    logger.Warnw("pool unavailable, fallback to goroutine",
        "error", err.Error(),
    )
    go task()
}
```

### 获取池统计

```go
// 获取所有池统计
stats := pool.Stats()
for name, info := range stats {
    fmt.Printf("池 %s: 运行中=%d, 空闲=%d, 等待=%d\n",
        name, info.Running, info.Free, info.Waiting)
}

// 动态调整池容量
pool.Tune("default", 2000)
```

## 配置选项详解

### PoolConfig 参数

| 参数 | 类型 | 说明 | 推荐值 |
|------|------|------|--------|
| `Capacity` | int | 池容量 | 根据业务计算 |
| `ExpiryDuration` | Duration | 空闲 worker 过期时间 | 5-60s |
| `PreAlloc` | bool | 是否预分配 | 高并发场景设为 true |
| `Nonblocking` | bool | 非阻塞模式 | 回调场景设为 true |
| `MaxBlockingTasks` | int | 最大阻塞等待数 | 容量的 10-50% |
| `DisablePurge` | bool | 禁用自动清理 | 一般为 false |
| `PanicHandler` | func | 自定义 panic 处理 | 可选 |

### 阻塞 vs 非阻塞模式

**阻塞模式（默认）：**

- 池满时等待空闲 worker
- 适合必须执行的任务
- 可设置 `MaxBlockingTasks` 限制等待队列

**非阻塞模式：**

- 池满时立即返回错误
- 适合可丢弃的任务（如回调通知）
- 需要调用方处理 `ErrPoolOverload`

```go
// 非阻塞模式示例
if err := pool.SubmitToType(pool.CallbackPool, task); err != nil {
    if errors.Is(err, ants.ErrPoolOverload) {
        // 池已满，任务被丢弃
        logger.Warnw("callback pool overloaded, task dropped")
        return
    }
    // 其他错误
}
```

## 迁移指南

### 从直接 goroutine 迁移

**原代码（禁止）：**

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("panic: %v", r)
        }
    }()
    doSomething()
}()
```

**新代码（推荐）：**

```go
task := func() {
    doSomething() // panic 会被池自动捕获
}

if err := pool.Submit(task); err != nil {
    logger.Warnw("submit failed, fallback", "error", err)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                logger.Errorw("panic recovered", "error", r)
            }
        }()
        task()
    }()
}
```

### 从 WaitGroup + goroutine 迁移

**原代码：**

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(i Item) {
        defer wg.Done()
        process(i)
    }(item)
}
wg.Wait()
```

**新代码：**

```go
var wg sync.WaitGroup
healthPool, _ := pool.GetByType(pool.HealthCheckPool)

for _, item := range items {
    wg.Add(1)
    i := item
    task := func() {
        defer wg.Done()
        process(i)
    }

    if err := healthPool.Submit(task); err != nil {
        go task() // 降级
    }
}
wg.Wait()
```

## 最佳实践

### 1. 任务设计原则

```go
// 好的实践：任务自包含，无外部依赖
task := func() {
    data := fetchData()
    result := processData(data)
    saveResult(result)
}

// 避免：任务依赖外部变量（闭包陷阱）
for i := 0; i < 10; i++ {
    // 错误：i 在所有 goroutine 中是同一个变量
    pool.Submit(func() {
        fmt.Println(i) // 可能打印 10 十次
    })
}

// 正确：捕获变量
for i := 0; i < 10; i++ {
    idx := i // 捕获当前值
    pool.Submit(func() {
        fmt.Println(idx) // 正确打印 0-9
    })
}
```

### 2. 超时控制

```go
// 为长任务添加超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := pool.SubmitWithContext(ctx, func() {
    select {
    case <-ctx.Done():
        return // 任务被取消
    default:
        longRunningTask()
    }
})
```

### 3. 监控和告警

```go
// 定期检查池状态
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        stats := pool.Stats()
        for name, info := range stats {
            // 等待任务过多时告警
            if info.Waiting > info.Capacity/2 {
                logger.Warnw("pool backlog warning",
                    "pool", name,
                    "waiting", info.Waiting,
                    "capacity", info.Capacity,
                )
            }
        }
    }
}()
```

### 4. 优雅关闭

```go
// 应用关闭时等待任务完成
func shutdown() {
    // 设置超时
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 优雅关闭，等待任务完成
    if err := pool.CloseGlobalTimeout(30 * time.Second); err != nil {
        logger.Errorw("pool shutdown timeout", "error", err)
    }
}
```

## 问题排查

### 常见问题

1. **ErrPoolOverload**：池容量不足
   - 解决：增加容量或使用阻塞模式

2. **任务堆积**：`Waiting` 数值过高
   - 解决：检查任务执行时间，考虑增加容量

3. **内存增长**：未正确关闭池
   - 解决：确保调用 `CloseGlobal()`

4. **panic 未捕获**：自定义 PanicHandler 有问题
   - 解决：检查 PanicHandler 实现

### 调试命令

```bash
# 查看 goroutine 数量
curl http://localhost:8080/debug/pprof/goroutine?debug=2

# 查看池统计（需实现 metrics 端点）
curl http://localhost:8080/metrics | grep pool
```

## 代码审查清单

- [ ] 是否使用了 `go func()` ？如有，是否属于允许的例外？
- [ ] 是否选择了合适的池类型？
- [ ] 是否处理了池不可用的降级情况？
- [ ] 是否正确捕获了闭包变量？
- [ ] 长任务是否添加了超时控制？
- [ ] 是否有适当的错误处理和日志记录？

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2025-01-XX | 初始版本，统一使用 ants 池 |
