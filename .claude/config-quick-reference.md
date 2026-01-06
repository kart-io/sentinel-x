# 配置文件快速参考

## 配置文件端口映射

| 服务 | 配置文件 | HTTP 端口 | gRPC 端口 | Metrics Subsystem |
|------|----------|-----------|-----------|-------------------|
| User Center | user-center.yaml | :8081 | :8101 | user-center |
| User Center (Dev) | user-center-dev.yaml | :8081 | :8104 | user-center |
| Sentinel API | sentinel-api.yaml | :8080 | :8100 | api |
| Sentinel API (Dev) | sentinel-api-dev.yaml | :8100 | :8103 | api |
| RAG Service | rag.yaml | :8082 | :8102 | rag |

## 中间件配置对照表

### 通用中间件（所有服务）

| 中间件 | 配置段 | 关键配置项 |
|--------|--------|-----------|
| Metrics | `metrics:` | `path`, `namespace`, `subsystem` |
| Health Check | `health:` | `path`, `liveness-path`, `readiness-path` |
| Pprof | `pprof:` | `prefix`, `enable-cmdline`, `enable-profile` |
| Recovery | `recovery:` | `enable-stack-trace` |
| Logger | `logger:` | `skip-paths`, `use-structured-logger` |
| CORS | `cors:` | `allow-origins`, `allow-methods`, `allow-headers` |
| Timeout | `timeout:` | `timeout`, `skip-paths` |
| Request ID | `request-id:` | `header` |
| Version | `version:` | `enabled`, `path`, `hide-details` |

### 特殊中间件（仅 Sentinel API）

| 中间件 | 配置段 | 关键配置项 |
|--------|--------|-----------|
| Auth | `auth:` | `token-lookup`, `auth-scheme`, `skip-paths`, `skip-path-prefixes` |

## 配置路径变更对照表

| 迁移前路径 | 迁移后路径 | 说明 |
|-----------|-----------|------|
| `server.http.middleware.metrics` | `metrics` | 提升到顶层 |
| `server.http.middleware.health` | `health` | 提升到顶层 |
| `server.http.middleware.pprof` | `pprof` | 提升到顶层 |
| `server.http.middleware.recovery` | `recovery` | 提升到顶层 |
| `server.http.middleware.logger` | `logger` | 提升到顶层 |
| `server.http.middleware.cors` | `cors` | 提升到顶层 |
| `server.http.middleware.timeout` | `timeout` | 提升到顶层 |
| `server.http.middleware.request-id` | `request-id` | 提升到顶层 |
| `server.http.middleware.auth` | `auth` | 提升到顶层 |
| `server.http.addr` | `http.addr` | HTTP 配置独立 |
| `server.grpc.addr` | `grpc.addr` | gRPC 配置独立 |
| 无 | `version` | 新增 |

## 中间件启用控制变更

### 迁移前（3 种方式）

#### 方式 1：enabled 列表

```yaml
middleware:
  enabled:
    - recovery
    - request-id
    - logger
```

#### 方式 2：disable-* 标志

```yaml
middleware:
  disable-recovery: false
  disable-metrics: false
  disable-auth: true
```

#### 方式 3：混合方式

```yaml
middleware:
  enabled: [...]
  disable-pprof: true
```

### 迁移后（统一方式）

配置存在即启用：

```yaml
# 配置存在 = 启用
metrics:
  path: /metrics

health:
  path: /health

# 如需禁用，移除整个配置段或在代码层面处理
```

## 配置结构对比

### 服务器配置

#### 迁移前

```yaml
server:
  mode: both
  shutdown-timeout: 30s
  http:
    addr: ":8081"
    middleware:
      enabled: [...]
  grpc:
    addr: ":8101"
```

#### 迁移后

```yaml
server:
  mode: both
  shutdown-timeout: 30s

http:
  addr: ":8081"
  read-timeout: 30s
  write-timeout: 30s
  idle-timeout: 60s
  adapter: gin

grpc:
  addr: ":8101"
  timeout: 30s
  max-recv-msg-size: 4194304
  max-send-msg-size: 4194304
  enable-reflection: true
```

### 中间件配置

#### 迁移前

```yaml
server:
  http:
    middleware:
      metrics:
        path: /metrics
        namespace: sentinel
        subsystem: user-center
      health:
        path: /health
        liveness-path: /live
        readiness-path: /ready
```

#### 迁移后

```yaml
metrics:
  path: /metrics
  namespace: sentinel
  subsystem: user-center

health:
  path: /health
  liveness-path: /live
  readiness-path: /ready
```

## 代码适配速查

### 配置结构体

#### 迁移前

```go
type Config struct {
    Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
    Mode string     `yaml:"mode"`
    HTTP HTTPConfig `yaml:"http"`
    GRPC GRPCConfig `yaml:"grpc"`
}

type HTTPConfig struct {
    Addr       string           `yaml:"addr"`
    Middleware MiddlewareConfig `yaml:"middleware"`
}

type MiddlewareConfig struct {
    Enabled []string      `yaml:"enabled"`
    Metrics MetricsConfig `yaml:"metrics"`
    Health  HealthConfig  `yaml:"health"`
}
```

#### 迁移后

```go
type Config struct {
    Server    ServerConfig    `yaml:"server"`
    HTTP      HTTPConfig      `yaml:"http"`
    GRPC      GRPCConfig      `yaml:"grpc"`
    Metrics   MetricsConfig   `yaml:"metrics"`
    Health    HealthConfig    `yaml:"health"`
    Pprof     PprofConfig     `yaml:"pprof"`
    Recovery  RecoveryConfig  `yaml:"recovery"`
    Logger    LoggerConfig    `yaml:"logger"`
    CORS      CORSConfig      `yaml:"cors"`
    Timeout   TimeoutConfig   `yaml:"timeout"`
    RequestID RequestIDConfig `yaml:"request-id"`
    Version   VersionConfig   `yaml:"version"`
    Auth      *AuthConfig     `yaml:"auth,omitempty"`
}

type ServerConfig struct {
    Mode            string        `yaml:"mode"`
    ShutdownTimeout time.Duration `yaml:"shutdown-timeout"`
}

type HTTPConfig struct {
    Addr         string        `yaml:"addr"`
    ReadTimeout  time.Duration `yaml:"read-timeout"`
    WriteTimeout time.Duration `yaml:"write-timeout"`
    IdleTimeout  time.Duration `yaml:"idle-timeout"`
    Adapter      string        `yaml:"adapter"`
}
```

### 配置访问

#### 迁移前

```go
// 访问中间件配置
middlewareConfig := config.Server.HTTP.Middleware
metricsPath := middlewareConfig.Metrics.Path

// 检查中间件是否启用
if slices.Contains(middlewareConfig.Enabled, "recovery") {
    // 启用 recovery
}

if !middlewareConfig.DisableMetrics {
    // 启用 metrics
}
```

#### 迁移后

```go
// 直接访问顶层配置
metricsPath := config.Metrics.Path

// 配置存在即启用
if config.Metrics.Path != "" {
    // 启用 metrics
}

if config.Health.Path != "" {
    // 启用 health
}
```

### 中间件注册

#### 迁移前

```go
func RegisterMiddlewares(r *gin.Engine, cfg *config.Config) {
    mw := cfg.Server.HTTP.Middleware

    if slices.Contains(mw.Enabled, "recovery") {
        r.Use(middleware.Recovery(mw.Recovery))
    }

    if slices.Contains(mw.Enabled, "metrics") {
        r.Use(middleware.Metrics(mw.Metrics))
    }
}
```

#### 迁移后

```go
func RegisterMiddlewares(r *gin.Engine, cfg *config.Config) {
    // 配置存在即注册
    if cfg.Recovery.EnableStackTrace || true {
        r.Use(middleware.Recovery(cfg.Recovery))
    }

    if cfg.Metrics.Path != "" {
        r.Use(middleware.Metrics(cfg.Metrics))
    }
}
```

## 端点快速测试

### User Center

```bash
# 生产配置（:8081）
curl http://localhost:8081/health
curl http://localhost:8081/metrics
curl http://localhost:8081/version

# 开发配置（:8081）
curl http://localhost:8081/health
curl http://localhost:8081/metrics
curl http://localhost:8081/version
```

### Sentinel API

```bash
# 生产配置（:8080）
curl http://localhost:8080/health
curl http://localhost:8080/metrics
curl http://localhost:8080/version

# 开发配置（:8100）
curl http://localhost:8100/health
curl http://localhost:8100/metrics
curl http://localhost:8100/version
```

### RAG Service

```bash
# 生产配置（:8082）
curl http://localhost:8082/health
curl http://localhost:8082/metrics
curl http://localhost:8082/version
```

## 常见问题速查

### Q1：如何禁用某个中间件？

**迁移前**：

```yaml
middleware:
  disable-pprof: true
```

**迁移后**：

方案 1：移除配置段

```yaml
# 删除 pprof 配置段
# pprof:
#   prefix: /debug/pprof
```

方案 2：在代码中控制

```go
if cfg.Pprof.Prefix != "" && !isDevelopment() {
    r.Use(middleware.Pprof(cfg.Pprof))
}
```

### Q2：如何知道哪些中间件被启用？

**迁移前**：

```yaml
middleware:
  enabled:
    - recovery
    - request-id
    - logger
```

**迁移后**：

查看配置文件中存在的中间件配置段即可。

### Q3：开发环境和生产环境如何区分？

使用不同的配置文件：

- 生产环境：`user-center.yaml`
- 开发环境：`user-center-dev.yaml`

启动时指定：

```bash
# 生产
go run cmd/user-center/main.go --config configs/user-center.yaml

# 开发
go run cmd/user-center/main.go --config configs/user-center-dev.yaml
```

### Q4：version 端点如何实现？

```go
// 在路由注册中添加
router.GET("/version", func(c *gin.Context) {
    c.JSON(200, gin.H{
        "version": "v1.0.0",
        "commit":  "abc123",
        "date":    "2026-01-06",
    })
})
```

### Q5：如何验证配置加载是否正确？

```bash
# 方案 1：dry-run 模式（如果支持）
go run cmd/user-center/main.go --config configs/user-center.yaml --dry-run

# 方案 2：添加日志
logger.Infof("Loaded config: %+v", cfg)

# 方案 3：使用配置验证函数
if err := ValidateConfig(cfg); err != nil {
    log.Fatal(err)
}
```

## 相关文档

- **详细迁移报告**：`.claude/config-migration-report.md`
- **迁移对比详情**：`.claude/config-migration-comparison.md`
- **完成总结**：`.claude/config-migration-summary.md`
- **快速参考**：`.claude/config-quick-reference.md`（本文件）

---

**生成时间**：2026-01-06
**配置格式版本**：扁平化格式 v1.0
