# Storage Package

统一的存储接口和基础类型，为 sentinel-x 项目中的所有存储实现提供一致的抽象层。

## 概述

`storage` 包定义了所有存储客户端必须实现的核心接口，支持多种存储类型（Redis、MySQL、MongoDB 等），并提供以下功能:

- **Client 接口**: 所有存储客户端的基础接口
- **Manager**: 多客户端注册和生命周期管理
- **Options**: 配置验证和通用选项
- **Errors**: 标准化错误类型，支持上下文信息
- **Health Checking**: 内置健康检查功能

## 目录结构

```text
pkg/storage/
├── storage.go   # 核心接口定义 (Client, Factory, HealthChecker)
├── options.go   # 配置选项接口和通用选项
├── manager.go   # 存储管理器实现
├── errors.go    # 标准化错误类型
├── doc.go       # 包文档和使用示例
└── README.md    # 本文件
```

## 核心接口

### Client 接口

所有存储客户端必须实现的基础接口:

```go
type Client interface {
    Name() string                    // 返回存储类型名称
    Ping(ctx context.Context) error  // 检查连接是否正常
    Close() error                    // 优雅关闭连接
    Health() HealthChecker           // 返回健康检查函数
}
```

### Factory 接口

用于创建存储客户端的工厂接口:

```go
type Factory interface {
    Create(ctx context.Context) (Client, error)
}
```

### Options 接口

配置选项必须实现的验证接口:

```go
type Options interface {
    Validate() error
}
```

## 使用示例

### 基本使用

```go
import (
    "context"
    "github.com/kart-io/sentinel-x/pkg/storage"
    "github.com/kart-io/sentinel-x/pkg/storage/redis"
)

func main() {
    // 创建 Redis 客户端
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 验证连接
    ctx := context.Background()
    if err := client.Ping(ctx); err != nil {
        log.Fatalf("连接失败: %v", err)
    }
    defer client.Close()

    // 使用客户端...
}
```

### 使用 Manager 管理多个存储

```go
// 创建管理器
mgr := storage.NewManager()

// 注册多个客户端
mgr.MustRegister("redis-cache", redisClient)
mgr.MustRegister("mysql-primary", mysqlClient)
mgr.MustRegister("mongo-events", mongoClient)

// 获取特定客户端
cache, err := mgr.Get("redis-cache")
if err != nil {
    log.Printf("缓存不可用: %v", err)
}

// 健康检查所有客户端
statuses := mgr.HealthCheckAll(ctx)
for name, status := range statuses {
    if status.Healthy {
        log.Printf("%s: 健康 (延迟: %v)", name, status.Latency)
    } else {
        log.Printf("%s: 不健康 - %v", name, status.Error)
    }
}

// 关闭所有客户端
defer mgr.CloseAll()
```

### 健康检查

```go
// 直接健康检查
if err := client.Ping(ctx); err != nil {
    log.Printf("健康检查失败: %v", err)
}

// 使用健康检查函数
checker := client.Health()
if err := checker(); err != nil {
    log.Printf("健康检查失败: %v", err)
}

// 管理器级别的健康检查
status := mgr.HealthCheck(ctx, "redis-cache")
if !status.Healthy {
    log.Printf("不健康: %v (延迟: %v)", status.Error, status.Latency)
}
```

### 错误处理

```go
err := client.Ping(ctx)
if err != nil {
    // 检查特定错误类型
    if errors.Is(err, storage.ErrNotConnected) {
        log.Println("客户端未连接")
    } else if errors.Is(err, storage.ErrTimeout) {
        log.Println("操作超时")
    }

    // 提取存储错误详情
    if storageErr, ok := storage.GetStorageError(err); ok {
        log.Printf("错误代码: %s", storageErr.Code)
        log.Printf("消息: %s", storageErr.Message)
        if ctx, ok := storageErr.GetContext("operation"); ok {
            log.Printf("操作: %v", ctx)
        }
    }
}
```

## 实现自定义存储客户端

要实现新的存储类型，需要:

1. 实现 `Client` 接口的所有方法
2. 定义特定的 Options 结构体，实现 `Options` 接口
3. (可选) 实现 `Factory` 接口用于客户端创建

### 示例实现

```go
package mystorage

import (
    "context"
    "time"
    "github.com/kart-io/sentinel-x/pkg/storage"
)

// Options 定义存储配置
type Options struct {
    storage.CommonOptions
    Addr     string
    Password string
}

// Validate 验证配置
func (o *Options) Validate() error {
    if err := o.CommonOptions.Validate(); err != nil {
        return err
    }
    if o.Addr == "" {
        return storage.ErrInvalidConfig.WithMessage("地址不能为空")
    }
    return nil
}

// Client 实现存储客户端
type Client struct {
    conn *SomeConnection
}

// Name 返回存储类型名称
func (c *Client) Name() string {
    return "mystorage"
}

// Ping 检查连接健康状态
func (c *Client) Ping(ctx context.Context) error {
    return c.conn.Ping(ctx)
}

// Close 关闭连接
func (c *Client) Close() error {
    return c.conn.Close()
}

// Health 返回健康检查函数
func (c *Client) Health() storage.HealthChecker {
    return func() error {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        return c.Ping(ctx)
    }
}
```

## 标准错误类型

包提供以下标准错误:

- `ErrNotConnected` - 客户端未连接
- `ErrConnectionFailed` - 连接失败
- `ErrTimeout` - 操作超时
- `ErrInvalidConfig` - 配置无效
- `ErrClientNotFound` - 客户端未找到
- `ErrClientAlreadyExists` - 客户端已存在
- `ErrOperationFailed` - 操作失败

### 错误增强

```go
// 添加消息
err := storage.ErrConnectionFailed.WithMessage("连接 Redis 失败: localhost:6379")

// 添加原因
err := storage.ErrConnectionFailed.WithCause(netErr)

// 添加上下文
err := storage.ErrTimeout.WithContext(map[string]interface{}{
    "operation": "GET",
    "key": "user:123",
    "timeout": "5s",
})
```

## CommonOptions

所有存储实现可以嵌入 `CommonOptions` 以获得通用配置:

```go
type CommonOptions struct {
    MaxRetries    int   // 最大重试次数
    Timeout       int64 // 操作超时时间(毫秒)
    PoolSize      int   // 连接池大小
    MinIdleConns  int   // 最小空闲连接数
    EnableTracing bool  // 启用追踪
    EnableMetrics bool  // 启用指标收集
}
```

### 使用 CommonOptions

```go
type RedisOptions struct {
    storage.CommonOptions
    Addr string
    DB   int
}

opts := &RedisOptions{
    Addr: "localhost:6379",
    DB:   0,
}

// 设置默认值
opts.CommonOptions.SetDefaults()

// 验证
if err := opts.Validate(); err != nil {
    log.Fatal(err)
}
```

## Manager API

### 注册客户端

```go
// 注册客户端(返回错误)
err := mgr.Register("redis-cache", client)

// 注册客户端(失败时 panic)
mgr.MustRegister("redis-cache", client)
```

### 查询客户端

```go
// 获取客户端
client, err := mgr.Get("redis-cache")

// 检查是否存在
if mgr.Has("redis-cache") {
    // ...
}

// 列出所有客户端名称
names := mgr.List()

// 获取客户端数量
count := mgr.Count()
```

### 健康检查

```go
// 检查单个客户端
status := mgr.HealthCheck(ctx, "redis-cache")

// 检查所有客户端
statuses := mgr.HealthCheckAll(ctx)

// 检查所有客户端是否健康
if mgr.AllHealthy(ctx) {
    log.Println("所有存储都健康")
}
```

### 生命周期管理

```go
// 注销客户端(不关闭)
mgr.Unregister("redis-cache")

// 关闭并注销客户端
mgr.Close("redis-cache")

// 关闭所有客户端
mgr.CloseAll()

// 清空管理器(不关闭客户端)
mgr.Clear()
```

## 线程安全

- `Manager` 是线程安全的，可以在多个 goroutine 中并发使用
- 存储客户端实现应该在文档中说明其线程安全保证

## 上下文支持

所有可能阻塞的操作都接受 `context.Context` 用于取消和超时控制:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := client.Ping(ctx); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("操作超时")
    }
}
```

## 最佳实践

1. **使用 Manager**: 对于多存储场景，使用 Manager 进行集中管理
2. **设置超时**: 始终为操作设置合理的超时时间
3. **健康检查**: 在应用启动时和运行时定期进行健康检查
4. **优雅关闭**: 使用 defer 确保资源被正确释放
5. **错误处理**: 使用标准错误类型并添加适当的上下文信息
6. **配置验证**: 在创建客户端前验证配置

## 未来扩展

该包设计为可扩展的，支持未来添加:

- 新的存储类型 (Redis, MySQL, MongoDB, PostgreSQL, etc.)
- 连接池管理
- 重试策略
- 断路器模式
- 分布式追踪集成
- 指标收集

## 相关包

- `pkg/storage/redis` - Redis 存储实现 (即将实现)
- `pkg/storage/mysql` - MySQL 存储实现 (即将实现)
- `pkg/storage/mongo` - MongoDB 存储实现 (即将实现)
- `pkg/storage/postgres` - PostgreSQL 存储实现 (即将实现)
