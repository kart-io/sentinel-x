# 配置文件迁移对比详情

## user-center.yaml 迁移对比

### 迁移前（嵌套结构）

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

    middleware:
      enabled:
        - recovery
        - request-id
        - logger
        - cors
        - timeout
        - health
        - metrics

      recovery:
        enable-stack-trace: false

      request-id:
        header: X-Request-ID

      logger:
        skip-paths:
          - /health
          - /live
          - /ready
          - /metrics
        use-structured-logger: true

      cors:
        allow-origins:
          - "*"
        allow-methods: [...]
        allow-headers: [...]
        allow-credentials: false
        max-age: 86400

      timeout:
        timeout: 30s
        skip-paths: [...]

      health:
        path: /health
        liveness-path: /live
        readiness-path: /ready

      metrics:
        path: /metrics
        namespace: sentinel
        subsystem: user-center

      pprof:
        prefix: /debug/pprof
        enable-cmdline: true
        enable-profile: true
        enable-symbol: true
        enable-trace: true

  grpc:
    addr: ":8101"
    timeout: 30s
    max-recv-msg-size: 4194304
    max-send-msg-size: 4194304
    enable-reflection: true
```

### 迁移后（扁平结构）

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

# 中间件配置（扁平化）
metrics:
  path: /metrics
  namespace: sentinel
  subsystem: user-center

health:
  path: /health
  liveness-path: /live
  readiness-path: /ready

pprof:
  prefix: /debug/pprof
  enable-cmdline: true
  enable-profile: true
  enable-symbol: true
  enable-trace: true
  block-profile-rate: 0
  mutex-profile-fraction: 0

recovery:
  enable-stack-trace: false

logger:
  skip-paths:
    - /health
    - /live
    - /ready
    - /metrics
  use-structured-logger: true

cors:
  allow-origins:
    - "*"
  allow-methods: [...]
  allow-headers: [...]
  allow-credentials: false
  max-age: 86400

timeout:
  timeout: 30s
  skip-paths:
    - /health
    - /live
    - /ready
    - /metrics

request-id:
  header: X-Request-ID

version:  # 新增
  enabled: true
  path: /version
  hide-details: false
```

## sentinel-api.yaml 迁移对比

### 迁移前（嵌套结构）

```yaml
server:
  mode: both
  shutdown-timeout: 30s

  http:
    addr: ":8080"
    read-timeout: 30s
    write-timeout: 30s
    idle-timeout: 60s
    adapter: gin

    middleware:
      enabled:
        - recovery
        - request-id
        - logger
        - cors
        - timeout
        - health
        - metrics
        - auth      # 特殊：包含认证中间件
        - authz     # 特殊：包含鉴权中间件

      # ... 其他中间件配置同 user-center.yaml

      auth:  # 特殊配置
        token-lookup: "header:Authorization"
        auth-scheme: "Bearer"
        skip-paths:
          - /api/v1/auth/login
          - /api/v1/auth/register
          - /api/v1/hello
          - /health
          - /live
          - /ready
          - /metrics
        skip-path-prefixes:
          - /api/v1/public/

  grpc:
    addr: ":8100"
    timeout: 30s
    max-recv-msg-size: 4194304
    max-send-msg-size: 4194304
    enable-reflection: true
```

### 迁移后（扁平结构）

```yaml
server:
  mode: both
  shutdown-timeout: 30s

http:
  addr: ":8080"
  read-timeout: 30s
  write-timeout: 30s
  idle-timeout: 60s
  adapter: gin

grpc:
  addr: ":8100"
  timeout: 30s
  max-recv-msg-size: 4194304
  max-send-msg-size: 4194304
  enable-reflection: true

# 中间件配置（扁平化）
metrics:
  path: /metrics
  namespace: sentinel
  subsystem: api

health:
  path: /health
  liveness-path: /live
  readiness-path: /ready

# ... 其他中间件配置同 user-center.yaml

auth:  # 认证中间件配置提升到顶层
  token-lookup: "header:Authorization"
  auth-scheme: "Bearer"
  skip-paths:
    - /api/v1/auth/login
    - /api/v1/auth/register
    - /api/v1/hello
    - /health
    - /live
    - /ready
    - /metrics
  skip-path-prefixes:
    - /api/v1/public/

version:  # 新增
  enabled: true
  path: /version
  hide-details: false
```

## sentinel-api-dev.yaml 迁移对比

### 迁移前（使用 disable-* 标志）

```yaml
server:
  mode: http  # 开发环境仅启用 HTTP
  shutdown-timeout: 10s

http:
  addr: ":8100"
  read-timeout: 30s
  write-timeout: 30s
  idle-timeout: 60s
  adapter: gin

middleware:
  # 使用 disable-* 标志
  disable-recovery: false
  disable-request-id: false
  disable-logger: false
  disable-health: false
  disable-metrics: false

  # 开发环境禁用
  disable-auth: true
  disable-authz: true
  disable-cors: true
  disable-timeout: true
  disable-pprof: true

  health:
    path: /health
    liveness-path: /live
    readiness-path: /ready

  metrics:
    path: /metrics
    namespace: sentinel
    subsystem: api

grpc:
  addr: ":8103"
  timeout: 30s
  max-recv-msg-size: 4194304
  max-send-msg-size: 4194304
  enable-reflection: true

log:
  level: debug
  format: console
  development: true
  output-paths:
    - stdout

jwt:
  disable-auth: false
  key: "your-super-secret-key-change-in-production-minimum-64-characters-required-here"
  signing-method: "HS256"
  issuer: "sentinel-x"
  expired: 24h
  max-refresh: 168h

mysql:
  host: ""
  database: ""

redis:
  host: ""
```

### 迁移后（扁平结构，移除 disable-* 标志）

```yaml
server:
  mode: http
  shutdown-timeout: 10s

http:
  addr: ":8100"
  read-timeout: 30s
  write-timeout: 30s
  idle-timeout: 60s
  adapter: gin

grpc:
  addr: ":8103"
  timeout: 30s
  max-recv-msg-size: 4194304
  max-send-msg-size: 4194304
  enable-reflection: true

# 中间件配置（扁平化，保留所有配置）
metrics:
  path: /metrics
  namespace: sentinel
  subsystem: api

health:
  path: /health
  liveness-path: /live
  readiness-path: /ready

pprof:
  prefix: /debug/pprof
  enable-cmdline: true
  enable-profile: true
  enable-symbol: true
  enable-trace: true

recovery:
  enable-stack-trace: false

logger:
  skip-paths:
    - /health
    - /live
    - /ready
    - /metrics
  use-structured-logger: true

cors:
  allow-origins:
    - "*"
  allow-methods:
    - GET
    - POST
    - PUT
    - PATCH
    - DELETE
    - HEAD
    - OPTIONS
  allow-headers:
    - Origin
    - Content-Type
    - Accept
    - Authorization
    - X-Request-ID
  allow-credentials: false
  max-age: 86400

timeout:
  timeout: 30s
  skip-paths: []

request-id:
  header: X-Request-ID

version:  # 新增
  enabled: true
  path: /version
  hide-details: false

log:
  level: debug
  format: console
  development: true
  output-paths:
    - stdout

jwt:
  disable-auth: false
  key: "your-super-secret-key-change-in-production-minimum-64-characters-required-here"
  signing-method: "HS256"
  issuer: "sentinel-x"
  expired: 24h
  max-refresh: 168h

# 通过空值禁用（而非 disable 标志）
mysql:
  host: ""
  database: ""

redis:
  host: ""
```

## 关键变更总结

### 1. 结构变更

| 迁移前 | 迁移后 |
|--------|--------|
| `server.http.middleware.*` | 顶层 `*` |
| `server.http.addr` | 顶层 `http.addr` |
| `server.grpc.addr` | 顶层 `grpc.addr` |
| `middleware.enabled: [...]` | 移除（配置即启用） |
| `middleware.disable-*: true` | 移除（通过空值或代码控制） |

### 2. 配置路径映射

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
| `server.http.middleware.auth` | `auth` | 提升到顶层（仅 sentinel-api） |
| 无 | `version` | 新增 |

### 3. 中间件启用控制变更

#### 迁移前（3 种方式）

**方式1：enabled 列表**（user-center.yaml, sentinel-api.yaml）

```yaml
middleware:
  enabled:
    - recovery
    - request-id
    - logger
    - cors
    - timeout
    - health
    - metrics
```

**方式2：disable-* 标志**（user-center-dev.yaml）

```yaml
middleware:
  disable-recovery: false
  disable-request-id: false
  disable-logger: false
  disable-cors: false
  disable-timeout: false
  disable-health: false
  disable-metrics: false
  disable-pprof: true
  disable-auth: false
  disable-authz: false
```

**方式3：简化版 disable-***（sentinel-api-dev.yaml）

```yaml
middleware:
  disable-recovery: false
  disable-request-id: false
  disable-logger: false
  disable-health: false
  disable-metrics: false
  disable-auth: true
  disable-authz: true
  disable-cors: true
  disable-timeout: true
  disable-pprof: true
```

#### 迁移后（统一方式）

**统一方式：配置存在即启用**

```yaml
# 配置存在 = 启用该中间件
metrics:
  path: /metrics
  namespace: sentinel
  subsystem: user-center

health:
  path: /health
  liveness-path: /live
  readiness-path: /ready

# 如需禁用，移除整个配置段或在代码层面处理
```

### 4. 新增配置段

所有配置文件都新增了 `version` 配置段：

```yaml
version:
  enabled: true
  path: /version
  hide-details: false
```

## 代码适配建议

### 1. 配置加载代码

**迁移前**：

```go
// 从嵌套结构加载
middlewareConfig := config.Server.HTTP.Middleware

if slices.Contains(middlewareConfig.Enabled, "recovery") {
    // 启用 recovery 中间件
}

if !middlewareConfig.DisableMetrics {
    // 启用 metrics 中间件
}
```

**迁移后**：

```go
// 从顶层加载
metricsConfig := config.Metrics
healthConfig := config.Health

// 配置存在即启用
if metricsConfig.Path != "" {
    // 启用 metrics 中间件
}

if healthConfig.Path != "" {
    // 启用 health 中间件
}
```

### 2. 配置结构体定义

**迁移前**：

```go
type ServerConfig struct {
    Mode            string        `yaml:"mode"`
    ShutdownTimeout time.Duration `yaml:"shutdown-timeout"`
    HTTP            HTTPConfig    `yaml:"http"`
    GRPC            GRPCConfig    `yaml:"grpc"`
}

type HTTPConfig struct {
    Addr        string           `yaml:"addr"`
    ReadTimeout time.Duration    `yaml:"read-timeout"`
    Middleware  MiddlewareConfig `yaml:"middleware"`
}

type MiddlewareConfig struct {
    Enabled          []string       `yaml:"enabled"`
    DisableRecovery  bool          `yaml:"disable-recovery"`
    DisableMetrics   bool          `yaml:"disable-metrics"`
    // ... 更多 disable-* 字段

    Recovery  RecoveryConfig  `yaml:"recovery"`
    Metrics   MetricsConfig   `yaml:"metrics"`
    Health    HealthConfig    `yaml:"health"`
    // ... 更多中间件配置
}
```

**迁移后**：

```go
type Config struct {
    Server  ServerConfig  `yaml:"server"`
    HTTP    HTTPConfig    `yaml:"http"`
    GRPC    GRPCConfig    `yaml:"grpc"`

    // 中间件配置（扁平化）
    Metrics   MetricsConfig   `yaml:"metrics"`
    Health    HealthConfig    `yaml:"health"`
    Pprof     PprofConfig     `yaml:"pprof"`
    Recovery  RecoveryConfig  `yaml:"recovery"`
    Logger    LoggerConfig    `yaml:"logger"`
    CORS      CORSConfig      `yaml:"cors"`
    Timeout   TimeoutConfig   `yaml:"timeout"`
    RequestID RequestIDConfig `yaml:"request-id"`
    Auth      *AuthConfig     `yaml:"auth,omitempty"`
    Version   VersionConfig   `yaml:"version"`

    // 业务配置
    Log   LogConfig   `yaml:"log"`
    JWT   JWTConfig   `yaml:"jwt"`
    MySQL MySQLConfig `yaml:"mysql"`
    Redis RedisConfig `yaml:"redis"`
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

### 3. 中间件注册代码

**迁移前**：

```go
func RegisterMiddlewares(r *gin.Engine, cfg *config.Config) {
    mw := cfg.Server.HTTP.Middleware

    if slices.Contains(mw.Enabled, "recovery") {
        r.Use(middleware.Recovery(mw.Recovery))
    }

    if slices.Contains(mw.Enabled, "request-id") {
        r.Use(middleware.RequestID(mw.RequestID))
    }

    if slices.Contains(mw.Enabled, "logger") {
        r.Use(middleware.Logger(mw.Logger))
    }

    // ... 更多中间件
}
```

**迁移后**：

```go
func RegisterMiddlewares(r *gin.Engine, cfg *config.Config) {
    // 配置存在即注册
    if cfg.Recovery.EnableStackTrace || true {
        r.Use(middleware.Recovery(cfg.Recovery))
    }

    if cfg.RequestID.Header != "" {
        r.Use(middleware.RequestID(cfg.RequestID))
    }

    if len(cfg.Logger.SkipPaths) > 0 || cfg.Logger.UseStructuredLogger {
        r.Use(middleware.Logger(cfg.Logger))
    }

    // 或使用辅助函数判断
    if isMiddlewareEnabled(cfg.Metrics) {
        r.Use(middleware.Metrics(cfg.Metrics))
    }

    if isMiddlewareEnabled(cfg.Health) {
        r.Use(middleware.Health(cfg.Health))
    }

    // ... 更多中间件
}

// 辅助函数：判断中间件是否启用
func isMiddlewareEnabled(mw interface{}) bool {
    v := reflect.ValueOf(mw)
    if v.Kind() == reflect.Ptr {
        return !v.IsNil()
    }
    return !v.IsZero()
}
```

## 迁移验证步骤

1. **语法验证**：

```bash
# 验证 YAML 语法
for f in configs/*.yaml; do
    echo "验证 $f"
    yamllint "$f"
done
```

2. **配置加载验证**：

```bash
# 使用新配置启动服务（dry-run 模式）
go run cmd/user-center/main.go --config configs/user-center.yaml --dry-run

go run cmd/api/main.go --config configs/sentinel-api.yaml --dry-run
```

3. **端点验证**：

```bash
# 启动服务后验证端点
curl http://localhost:8081/health
curl http://localhost:8081/metrics
curl http://localhost:8081/version  # 新增端点

curl http://localhost:8080/health
curl http://localhost:8080/metrics
curl http://localhost:8080/version  # 新增端点
```

4. **中间件功能验证**：

```bash
# 验证 CORS
curl -H "Origin: http://example.com" \
     -H "Access-Control-Request-Method: POST" \
     -X OPTIONS http://localhost:8081/api/v1/users

# 验证超时
curl http://localhost:8081/slow-endpoint

# 验证日志（检查日志输出）
tail -f logs/user-center.log
```

## 回滚方案

如需回滚到旧配置格式，可以使用以下命令：

```bash
# 备份新配置
cp configs/user-center.yaml configs/user-center.yaml.new
cp configs/user-center-dev.yaml configs/user-center-dev.yaml.new
cp configs/sentinel-api.yaml configs/sentinel-api.yaml.new
cp configs/sentinel-api-dev.yaml configs/sentinel-api-dev.yaml.new
cp configs/rag.yaml configs/rag.yaml.new

# 从 Git 恢复旧配置
git checkout HEAD~1 -- configs/*.yaml

# 或从备份恢复（如果有）
cp configs/*.yaml.backup configs/
```
