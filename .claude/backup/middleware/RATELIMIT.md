# Rate Limiting Middleware

速率限制中间件为 Sentinel-X 提供灵活且高性能的请求速率限制功能，有效防止暴力破解和 API 滥用。

## 功能特性

- **多种存储后端**：支持内存和 Redis 两种存储方式
- **灵活的限流键**：支持按 IP、用户 ID、API Key 等多种方式限流
- **滑动窗口算法**：采用精确的滑动窗口算法，避免突刺流量
- **路径跳过**：支持跳过特定路径的限流检查
- **自定义回调**：限流触发时可执行自定义回调
- **并发安全**：所有实现均支持高并发场景
- **优雅降级**：限流器错误时自动降级，不影响正常请求

## 快速开始

### 基本使用

```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
    "github.com/kart-io/sentinel-x/pkg/infra/server/transport/http"
)

// 使用默认配置（100 请求/分钟，按 IP 限流）
server := http.NewServer(
    http.WithAddr(":8080"),
    http.WithMiddleware(
        middleware.RateLimit(),
    ),
)
```

### 自定义配置

```go
config := middleware.RateLimitConfig{
    Limit:  50,                     // 50 次请求
    Window: 1 * time.Minute,        // 每分钟
    SkipPaths: []string{
        "/health",                  // 跳过健康检查
        "/metrics",                 // 跳过指标端点
    },
}

server := http.NewServer(
    http.WithMiddleware(
        middleware.RateLimitWithConfig(config),
    ),
)
```

## 配置说明

### RateLimitConfig

```go
type RateLimitConfig struct {
    // Limit: 时间窗口内允许的最大请求数
    // 默认: 100
    Limit int

    // Window: 时间窗口长度
    // 默认: 1 分钟
    Window time.Duration

    // KeyFunc: 提取限流键的函数
    // 默认: 使用客户端 IP
    KeyFunc func(c transport.Context) string

    // SkipPaths: 跳过限流的路径列表
    // 默认: []
    SkipPaths []string

    // OnLimitReached: 触发限流时的回调函数
    // 默认: nil
    OnLimitReached func(c transport.Context)

    // Limiter: 限流器实现
    // 默认: 使用内存限流器
    Limiter RateLimiter
}
```

### 默认配置

```go
DefaultRateLimitConfig = RateLimitConfig{
    Limit:          100,
    Window:         1 * time.Minute,
    KeyFunc:        nil,  // 使用默认 IP 提取函数
    SkipPaths:      []string{},
    OnLimitReached: nil,
    Limiter:        nil,  // 自动创建内存限流器
}
```

## 存储后端

### 内存限流器

适用于单实例部署，性能最优。

```go
// 方式 1: 自动创建（推荐）
config := middleware.RateLimitConfig{
    Limit:  100,
    Window: 1 * time.Minute,
    // Limiter 为 nil 时自动创建内存限流器
}

// 方式 2: 手动创建
limiter := middleware.NewMemoryRateLimiter(100, 1*time.Minute)
defer limiter.Stop()  // 停止清理协程

config := middleware.RateLimitConfig{
    Limit:   100,
    Window:  1 * time.Minute,
    Limiter: limiter,
}
```

**特性**：

- 使用 `sync.Map` 实现线程安全
- 采用滑动窗口算法，精确计数
- 自动清理过期条目，防止内存泄漏
- 支持并发调用，性能优异

### Redis 限流器

适用于分布式部署，多实例共享限流状态。

```go
// 创建 Redis 客户端
redisClient := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})

// 创建 Redis 限流器
limiter := middleware.NewRedisRateLimiter(redisClient, 100, 1*time.Minute)

config := middleware.RateLimitConfig{
    Limit:   100,
    Window:  1 * time.Minute,
    Limiter: limiter,
}
```

**特性**：

- 使用 Redis Sorted Set 实现分布式限流
- 支持多实例部署，共享限流状态
- 使用 Pipeline 优化性能
- 自动设置过期时间，避免内存累积

## 限流键策略

### 按 IP 限流（默认）

```go
// 自动使用客户端 IP 作为限流键
middleware.RateLimit()
```

提取优先级：`X-Forwarded-For` > `X-Real-IP` > `RemoteAddr`

### 按用户 ID 限流

```go
config := middleware.RateLimitConfig{
    Limit:  10,
    Window: 1 * time.Minute,
    KeyFunc: func(c transport.Context) string {
        userID := c.Header("X-User-ID")
        if userID == "" {
            return extractClientIP(c)  // 未登录用户使用 IP
        }
        return fmt.Sprintf("user:%s", userID)
    },
}
```

### 按 API Key 限流

```go
config := middleware.RateLimitConfig{
    Limit:  1000,  // API Key 用户更高限额
    Window: 1 * time.Minute,
    KeyFunc: func(c transport.Context) string {
        apiKey := c.Header("X-API-Key")
        if apiKey == "" {
            return fmt.Sprintf("ip:%s", extractClientIP(c))
        }
        return fmt.Sprintf("apikey:%s", apiKey)
    },
}
```

### 按端点限流

```go
config := middleware.RateLimitConfig{
    Limit:  50,
    Window: 1 * time.Minute,
    KeyFunc: func(c transport.Context) string {
        req := c.HTTPRequest()
        // 组合 IP 和路径，实现每个端点独立限流
        return fmt.Sprintf("%s:%s", extractClientIP(c), req.URL.Path)
    },
}
```

## 高级用法

### 跳过特定路径

```go
config := middleware.RateLimitConfig{
    Limit:  100,
    Window: 1 * time.Minute,
    SkipPaths: []string{
        "/health",      // 健康检查
        "/metrics",     // Prometheus 指标
        "/ready",       // 就绪探针
        "/live",        // 存活探针
    },
}
```

### 限流触发回调

```go
config := middleware.RateLimitConfig{
    Limit:  100,
    Window: 1 * time.Minute,
    OnLimitReached: func(c transport.Context) {
        req := c.HTTPRequest()
        logger.Warnw("rate limit exceeded",
            "ip", req.RemoteAddr,
            "path", req.URL.Path,
            "user_agent", req.UserAgent(),
        )

        // 可选：发送告警、记录黑名单等
    },
}
```

### 自定义限流器

实现 `RateLimiter` 接口：

```go
type CustomLimiter struct {
    // 自定义字段
}

func (l *CustomLimiter) Allow(ctx context.Context, key string) (bool, error) {
    // 自定义限流逻辑
    return true, nil
}

func (l *CustomLimiter) Reset(ctx context.Context, key string) error {
    // 自定义重置逻辑
    return nil
}

// 使用自定义限流器
limiter := &CustomLimiter{}
config := middleware.RateLimitConfig{
    Limiter: limiter,
}
```

## 错误处理

### 返回错误码

限流触发时返回 HTTP 429 状态码：

```json
{
  "code": 600001,
  "message": "Rate limit exceeded",
  "message_zh": "超出速率限制"
}
```

### 限流器错误

当限流器发生错误时（如 Redis 连接失败），中间件会：

1. 记录错误日志
2. 允许请求通过（优雅降级）
3. 不影响正常业务

```go
// 错误日志示例
{
  "level": "error",
  "message": "rate limiter error",
  "error": "redis connection error",
  "key": "192.168.1.1"
}
```

## 性能优化

### 内存限流器优化

1. **滑动窗口**：仅保留窗口内的请求记录
2. **自动清理**：定期清理过期条目，防止内存泄漏
3. **并发安全**：使用 `sync.Map` 和 `sync.Mutex` 保证线程安全
4. **零拷贝**：切片操作避免不必要的内存分配

### Redis 限流器优化

1. **Pipeline**：批量执行 Redis 命令，减少网络往返
2. **过期时间**：自动设置 key 过期，避免内存累积
3. **Sorted Set**：使用时间戳作为 score，高效实现滑动窗口

### 基准测试

```bash
# 运行性能测试
cd pkg/infra/middleware
go test -bench=. -benchmem

# 并发测试
go test -v -run TestMemoryRateLimiterConcurrency
```

## 最佳实践

### 1. 选择合适的限流策略

- **公开 API**：按 IP 限流，防止滥用
- **认证 API**：按用户 ID 限流，更精确控制
- **第三方集成**：按 API Key 限流，差异化配额

### 2. 设置合理的限流阈值

```go
// 不同场景的推荐配置
configs := map[string]middleware.RateLimitConfig{
    "login": {
        Limit:  5,
        Window: 1 * time.Minute,  // 登录接口：5 次/分钟
    },
    "api": {
        Limit:  100,
        Window: 1 * time.Minute,  // 普通 API：100 次/分钟
    },
    "search": {
        Limit:  20,
        Window: 1 * time.Minute,  // 搜索接口：20 次/分钟
    },
}
```

### 3. 跳过必要的路径

```go
SkipPaths: []string{
    "/health",    // 健康检查
    "/metrics",   // 监控指标
    "/ready",     // Kubernetes 就绪探针
    "/live",      // Kubernetes 存活探针
    "/static/",   // 静态资源（需使用前缀匹配）
}
```

### 4. 使用回调记录异常

```go
OnLimitReached: func(c transport.Context) {
    req := c.HTTPRequest()

    // 记录到数据库或日志系统
    logger.Warnw("rate limit exceeded",
        "ip", extractClientIP(c),
        "path", req.URL.Path,
        "user_agent", req.UserAgent(),
    )

    // 可选：增加黑名单计数
    // blacklist.Increment(extractClientIP(c))
}
```

### 5. 分布式部署使用 Redis

```go
// 生产环境推荐配置
redisClient := redis.NewClient(&redis.Options{
    Addr:         "redis-cluster:6379",
    Password:     os.Getenv("REDIS_PASSWORD"),
    DB:           0,
    PoolSize:     100,
    MinIdleConns: 10,
})

limiter := middleware.NewRedisRateLimiter(redisClient, 100, 1*time.Minute)
```

## 常见问题

### Q1: 如何实现不同用户不同限额？

```go
config := middleware.RateLimitConfig{
    KeyFunc: func(c transport.Context) string {
        userID := c.Header("X-User-ID")
        tier := getUserTier(userID)  // 获取用户等级
        return fmt.Sprintf("user:%s:tier:%s", userID, tier)
    },
}

// 配合多个限流中间件使用
```

### Q2: 如何实现按路径前缀跳过？

当前版本仅支持精确路径匹配，前缀匹配可通过自定义逻辑实现：

```go
// 在 KeyFunc 中实现前缀检查
KeyFunc: func(c transport.Context) string {
    path := c.HTTPRequest().URL.Path
    if strings.HasPrefix(path, "/static/") {
        return "skip"  // 返回特殊 key
    }
    return extractClientIP(c)
}
```

### Q3: 限流器占用多少内存？

内存限流器内存占用估算：

- 每个请求记录：约 40 字节（time.Time）
- 每个 key：约 50 字节（key 字符串 + 指针）
- 100 个 key，每个 100 次记录：约 500KB

Redis 限流器使用 Sorted Set，每条记录约 20 字节。

### Q4: 如何重置某个 key 的限流？

```go
// 获取限流器实例
limiter := middleware.NewMemoryRateLimiter(100, 1*time.Minute)

// 重置特定 key
err := limiter.Reset(context.Background(), "192.168.1.1")
```

## 测试

```bash
# 运行所有测试
go test -v ./pkg/infra/middleware -run TestRateLimit

# 运行性能测试
go test -bench=. -benchmem ./pkg/infra/middleware

# 并发测试
go test -v -run TestMemoryRateLimiterConcurrency
```

## 参考资料

- [错误码设计](../../utils/errors/README.md)
- [中间件架构](./README.md)
- [Redis Sorted Set](https://redis.io/docs/data-types/sorted-sets/)

## 许可证

MIT License
