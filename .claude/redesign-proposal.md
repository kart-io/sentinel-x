# Sentinel-X 项目重新设计方案

**版本**: v2.0
**日期**: 2025-12-11
**状态**: 提案

---

## 一、设计原则

### 1.1 核心原则

| 原则 | 说明 | 禁止行为 |
|------|------|----------|
| **单一选择** | 每个技术决策只选择一种方案 | 禁止同时支持 Gin 和 Echo |
| **颠覆式变更** | 不向后兼容，直接删除旧代码 | 禁止保留 legacy 代码 |
| **简单优先** | 选择最简单的实现方式 | 禁止过度抽象 |
| **显式优于隐式** | 明确的依赖和初始化顺序 | 禁止复杂的拓扑排序 |
| **YAGNI** | 不预留扩展点 | 禁止"可能用到"的设计 |

### 1.2 禁止的设计模式

以下设计模式在本项目中**绝对禁止**：

1. **双适配器模式** - 选择 Gin，删除 Echo 支持
2. **通用拓扑排序** - 使用固定顺序初始化
3. **多策略模式** - ID 生成只用 UUID
4. **Enable/Disable 对** - 使用 nil 表示禁用
5. **类型别名导出** - 直接导入原始包
6. **12 个错误创建函数** - 合并为 3 个
7. **5 层配置嵌套** - 最多 2 层

---

## 二、技术选型（最终决策）

### 2.1 技术栈精简

| 领域 | 选择 | 删除 |
|------|------|------|
| **Web 框架** | Gin | Echo |
| **数据库 ORM** | GORM | - |
| **缓存** | go-redis | - |
| **ID 生成** | UUID | Snowflake, ULID |
| **JSON 解析** | Sonic | encoding/json |
| **配置管理** | Viper | - |
| **日志** | kart-io/logger | - |
| **认证** | JWT (HS256) | RS256, ES256 |

### 2.2 删除清单

```bash
# 删除的文件和目录
pkg/infra/adapter/echo/           # Echo 适配器
pkg/infra/server/transport/http/adapter.go  # 双适配器系统
pkg/infra/middleware/exports.go   # 298行类型别名
pkg/utils/id/snowflake.go         # Snowflake 实现
pkg/utils/id/ulid.go              # ULID 实现
internal/bootstrap/dependency.go  # 拓扑排序系统
```

---

## 三、新目录结构

### 3.1 项目结构

```text
sentinel-x/
├── cmd/
│   ├── api/
│   │   └── main.go                  # API 服务入口
│   ├── user-center/
│   │   └── main.go                  # 用户中心入口
│   └── scheduler/
│       └── main.go                  # 调度器入口
│
├── internal/
│   ├── app/                         # 应用启动（简化后）
│   │   ├── app.go                   # 统一应用结构
│   │   ├── options.go               # 配置选项
│   │   └── run.go                   # 启动流程
│   │
│   ├── user-center/                 # 用户中心业务
│   │   ├── handler/
│   │   │   ├── user.go
│   │   │   └── auth.go
│   │   ├── service/                 # 重命名 biz -> service
│   │   │   ├── user.go
│   │   │   └── auth.go
│   │   ├── repository/              # 重命名 store -> repository
│   │   │   ├── interface.go
│   │   │   └── mysql.go
│   │   └── router.go                # 路由注册（移出 router/）
│   │
│   ├── scheduler/                   # 调度器（按设计文档）
│   │   ├── scheduler.go
│   │   ├── schedule_one.go
│   │   ├── framework/
│   │   │   ├── interface.go
│   │   │   └── runtime.go
│   │   ├── plugins/
│   │   │   ├── queuesort/
│   │   │   ├── filter/
│   │   │   ├── score/
│   │   │   └── executor/
│   │   └── queue/
│   │
│   └── model/
│       ├── user.go
│       └── task.go
│
├── pkg/
│   ├── server/                      # 简化后的服务器
│   │   ├── server.go                # HTTP 服务器（仅 Gin）
│   │   └── grpc.go                  # gRPC 服务器
│   │
│   ├── middleware/                  # 简化后的中间件
│   │   ├── recovery.go
│   │   ├── request_id.go
│   │   ├── logger.go
│   │   ├── cors.go
│   │   ├── auth.go
│   │   └── health.go
│   │
│   ├── security/
│   │   └── jwt/
│   │       ├── jwt.go               # JWT 实现（仅 HS256）
│   │       └── store.go             # Token 存储
│   │
│   ├── database/                    # 重命名 datasource
│   │   ├── mysql.go
│   │   └── redis.go
│   │
│   ├── errors/                      # 简化后的错误系统
│   │   ├── errno.go                 # 核心错误类型
│   │   ├── codes.go                 # 错误码定义
│   │   └── builder.go               # 错误构建器（3个）
│   │
│   ├── response/
│   │   └── response.go              # 统一响应
│   │
│   └── config/                      # 简化后的配置
│       └── config.go                # 平坦化配置
│
├── configs/
│   ├── api.yaml
│   └── user-center.yaml
│
└── docs/
    └── design/
        ├── api-server.md
        ├── user-center.md
        ├── auth-authz.md
        ├── scheduler.md
        └── error-code-design.md
```

### 3.2 删除的目录

```bash
# 完全删除
pkg/infra/                    # 过度抽象的基础设施层
pkg/options/                  # 30+ 配置类型
pkg/component/                # 冗余的组件层
internal/bootstrap/           # 复杂的启动系统
pkg/infra/adapter/            # 双适配器
```

---

## 四、核心组件重新设计

### 4.1 应用启动（简化）

**之前**: 复杂的 Bootstrap + 拓扑排序 + 双职责
**之后**: 简单的顺序初始化

```go
// internal/app/app.go
package app

import (
    "context"
    "github.com/gin-gonic/gin"
    "github.com/kart-io/logger"
)

// App 应用实例
type App struct {
    config *Config
    server *gin.Engine
    logger logger.Logger
}

// New 创建应用
func New(cfg *Config) *App {
    return &App{config: cfg}
}

// Run 启动应用（固定顺序初始化）
func (a *App) Run(ctx context.Context) error {
    // 1. 初始化日志
    a.logger = initLogger(a.config.Log)

    // 2. 初始化数据库
    db, err := initDatabase(a.config.MySQL)
    if err != nil {
        return err
    }
    defer db.Close()

    // 3. 初始化缓存
    cache, err := initCache(a.config.Redis)
    if err != nil {
        return err
    }
    defer cache.Close()

    // 4. 初始化 JWT
    jwt := initJWT(a.config.JWT)

    // 5. 创建 HTTP 服务器
    a.server = gin.New()

    // 6. 注册中间件
    a.registerMiddleware(jwt)

    // 7. 注册路由
    a.registerRoutes(db, cache, jwt)

    // 8. 启动服务器
    return a.server.Run(a.config.Server.Addr)
}
```

**关键变化**:
- 删除 Initializer 接口
- 删除拓扑排序
- 删除 Bootstrap 层
- 固定的初始化顺序（显式）

### 4.2 配置系统（平坦化）

**之前**: 5 层嵌套 + 30+ 配置类型
**之后**: 2 层嵌套 + 8 个配置类型

```go
// pkg/config/config.go
package config

import "time"

// Config 应用配置（平坦化）
type Config struct {
    Server ServerConfig `yaml:"server"`
    Log    LogConfig    `yaml:"log"`
    MySQL  MySQLConfig  `yaml:"mysql"`
    Redis  RedisConfig  `yaml:"redis"`
    JWT    JWTConfig    `yaml:"jwt"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
    Addr            string        `yaml:"addr"`
    ReadTimeout     time.Duration `yaml:"read-timeout"`
    WriteTimeout    time.Duration `yaml:"write-timeout"`
    ShutdownTimeout time.Duration `yaml:"shutdown-timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
    DSN             string `yaml:"dsn"`  // 单一 DSN 字符串
    MaxIdleConns    int    `yaml:"max-idle-conns"`
    MaxOpenConns    int    `yaml:"max-open-conns"`
    ConnMaxLifetime time.Duration `yaml:"conn-max-lifetime"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
    Addr     string `yaml:"addr"`
    Password string `yaml:"password"`
    DB       int    `yaml:"db"`
    PoolSize int    `yaml:"pool-size"`
}

// JWTConfig JWT 配置（仅 HS256）
type JWTConfig struct {
    Key       string        `yaml:"key"`        // 从环境变量
    Expired   time.Duration `yaml:"expired"`
    MaxRefresh time.Duration `yaml:"max-refresh"`
    Issuer    string        `yaml:"issuer"`
}
```

**关键变化**:
- 删除 Enable/Disable 对
- 删除 Validate() 方法（启动时验证）
- MySQL 使用单一 DSN
- 删除多签名算法支持

### 4.3 HTTP 服务器（仅 Gin，完全删除抽象层）

**之前**: 双适配器 + Bridge + 4-5 层转换
```
请求 → Server → Adapter → FrameworkBridge → RequestContext → gin.Context
```

**之后**: 直接使用 Gin，无中间层
```
请求 → Server → gin.Engine → gin.Context
```

#### ⚠️ 重要说明：必须完全删除 Bridge 层

仅删除 Echo 适配器是**不够的**，必须同时删除 Bridge 抽象层，否则会导致：
1. **过度设计残留** - FrameworkBridge 接口对单一实现无意义
2. **性能开销** - 每请求多一层 RequestContext 转换
3. **代码混乱** - AdapterType 枚举只剩一个值

#### 删除清单（完整）

```bash
# 必须删除的文件/目录
pkg/infra/adapter/echo/                        # Echo 适配器
pkg/infra/adapter/gin/bridge.go                # Gin Bridge（改为直接使用）
pkg/infra/server/transport/http/bridge.go      # FrameworkBridge 接口
pkg/infra/server/transport/http/adapter.go     # Adapter 注册机制
pkg/infra/server/transport/transport.go        # transport.Context 抽象
pkg/infra/server/transport/http/context.go     # RequestContext 实现
pkg/options/server/http/options.go             # AdapterType 枚举
```

#### 新实现

```go
// pkg/server/server.go
package server

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

// Server HTTP 服务器（直接使用 Gin，无抽象层）
type Server struct {
    engine *gin.Engine
    server *http.Server
    config *Config
}

// Config 服务器配置（删除 AdapterType）
type Config struct {
    Addr            string
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    ShutdownTimeout time.Duration
    // 注意：没有 Adapter 字段，因为只支持 Gin
}

// New 创建服务器
func New(cfg *Config) *Server {
    engine := gin.New()

    return &Server{
        engine: engine,
        config: cfg,
        server: &http.Server{
            Addr:         cfg.Addr,
            Handler:      engine,
            ReadTimeout:  cfg.ReadTimeout,
            WriteTimeout: cfg.WriteTimeout,
        },
    }
}

// Engine 返回 Gin 引擎（直接暴露，不封装）
func (s *Server) Engine() *gin.Engine {
    return s.engine
}

// Run 启动服务器
func (s *Server) Run() error {
    return s.server.ListenAndServe()
}

// Shutdown 优雅关闭
func (s *Server) Shutdown(ctx context.Context) error {
    return s.server.Shutdown(ctx)
}
```

**关键变化**:
- 删除 `FrameworkBridge` 接口
- 删除 `Adapter` 注册机制
- 删除 `transport.Context` 抽象
- 删除 `RequestContext` 实现
- 删除 `AdapterType` 枚举
- 直接使用 `gin.Engine` 和 `gin.Context`
- Handler 直接使用 `gin.HandlerFunc` 类型

### 4.4 中间件（直接导出）

**之前**: 298 行类型别名 + re-export
**之后**: 直接导入使用

```go
// pkg/middleware/auth.go
package middleware

import (
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/kart/sentinel-x/pkg/errors"
    "github.com/kart/sentinel-x/pkg/response"
    "github.com/kart/sentinel-x/pkg/security/jwt"
)

// Auth JWT 认证中间件
func Auth(jwtAuth *jwt.JWT, skipPaths ...string) gin.HandlerFunc {
    skip := make(map[string]bool)
    for _, path := range skipPaths {
        skip[path] = true
    }

    return func(c *gin.Context) {
        // 跳过路径
        if skip[c.Request.URL.Path] {
            c.Next()
            return
        }

        // 获取 Token
        token := extractToken(c)
        if token == "" {
            response.Fail(c, errors.ErrUnauthorized)
            c.Abort()
            return
        }

        // 验证 Token
        claims, err := jwtAuth.Verify(c.Request.Context(), token)
        if err != nil {
            response.Fail(c, errors.ErrInvalidToken)
            c.Abort()
            return
        }

        // 注入上下文
        c.Set("claims", claims)
        c.Set("user_id", claims.Subject)
        c.Next()
    }
}

func extractToken(c *gin.Context) string {
    auth := c.GetHeader("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return auth[7:]
    }
    return ""
}
```

**关键变化**:
- 删除 exports.go
- 删除类型别名
- 直接使用 gin.HandlerFunc
- 简化的 Skip 机制

### 4.5 错误系统（3 个构建器）

**之前**: 12 个错误创建函数
**之后**: 3 个构建器

```go
// pkg/errors/builder.go
package errors

import "net/http"

// 三种错误构建器（覆盖所有场景）

// NewClientError 客户端错误 (4xx)
func NewClientError(code int, messageEN, messageZH string) *Errno {
    return &Errno{
        Code:      code,
        HTTPCode:  http.StatusBadRequest,
        MessageEN: messageEN,
        MessageZH: messageZH,
    }
}

// NewAuthError 认证/授权错误 (401/403)
func NewAuthError(code int, httpCode int, messageEN, messageZH string) *Errno {
    return &Errno{
        Code:      code,
        HTTPCode:  httpCode,
        MessageEN: messageEN,
        MessageZH: messageZH,
    }
}

// NewServerError 服务器错误 (5xx)
func NewServerError(code int, messageEN, messageZH string) *Errno {
    return &Errno{
        Code:      code,
        HTTPCode:  http.StatusInternalServerError,
        MessageEN: messageEN,
        MessageZH: messageZH,
    }
}
```

**预定义错误码**:

```go
// pkg/errors/codes.go
package errors

import "net/http"

// 通用错误
var (
    OK = &Errno{Code: 0, HTTPCode: http.StatusOK, MessageEN: "Success", MessageZH: "成功"}

    // 客户端错误 (01xx)
    ErrBadRequest     = NewClientError(10001, "Bad request", "请求错误")
    ErrInvalidParam   = NewClientError(10002, "Invalid parameter", "参数无效")
    ErrValidationFailed = NewClientError(10003, "Validation failed", "验证失败")

    // 认证错误 (02xx)
    ErrUnauthorized   = NewAuthError(10020, http.StatusUnauthorized, "Unauthorized", "未认证")
    ErrInvalidToken   = NewAuthError(10021, http.StatusUnauthorized, "Invalid token", "令牌无效")
    ErrTokenExpired   = NewAuthError(10022, http.StatusUnauthorized, "Token expired", "令牌已过期")
    ErrForbidden      = NewAuthError(10030, http.StatusForbidden, "Forbidden", "禁止访问")

    // 资源错误 (04xx)
    ErrNotFound       = NewClientError(10040, "Resource not found", "资源不存在")
    ErrUserNotFound   = NewClientError(10041, "User not found", "用户不存在")
    ErrAlreadyExists  = NewClientError(10050, "Resource already exists", "资源已存在")

    // 服务器错误 (07xx)
    ErrInternal       = NewServerError(10070, "Internal server error", "服务器内部错误")
    ErrDatabase       = NewServerError(10080, "Database error", "数据库错误")
)
```

### 4.6 JWT 认证（仅 HS256）

**之前**: 支持 HS256/HS384/HS512/RS256/ES256
**之后**: 仅 HS256

```go
// pkg/security/jwt/jwt.go
package jwt

import (
    "context"
    "errors"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// JWT 认证器（仅 HS256）
type JWT struct {
    key        []byte
    expired    time.Duration
    maxRefresh time.Duration
    issuer     string
    store      Store
}

// Config JWT 配置
type Config struct {
    Key        string
    Expired    time.Duration
    MaxRefresh time.Duration
    Issuer     string
}

// New 创建 JWT 认证器
func New(cfg *Config, store Store) (*JWT, error) {
    if len(cfg.Key) < 32 {
        return nil, errors.New("JWT key must be at least 32 characters")
    }

    return &JWT{
        key:        []byte(cfg.Key),
        expired:    cfg.Expired,
        maxRefresh: cfg.MaxRefresh,
        issuer:     cfg.Issuer,
        store:      store,
    }, nil
}

// Sign 签发令牌
func (j *JWT) Sign(ctx context.Context, subject string) (string, error) {
    now := time.Now()
    claims := &Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   subject,
            Issuer:    j.issuer,
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(j.expired)),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(j.key)
}

// Verify 验证令牌
func (j *JWT) Verify(ctx context.Context, tokenString string) (*Claims, error) {
    // 检查是否已撤销
    if j.store != nil {
        revoked, err := j.store.IsRevoked(ctx, tokenString)
        if err != nil {
            return nil, err
        }
        if revoked {
            return nil, ErrTokenRevoked
        }
    }

    // 解析令牌
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidSigningMethod
        }
        return j.key, nil
    })

    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }

    return claims, nil
}

// Revoke 撤销令牌
func (j *JWT) Revoke(ctx context.Context, tokenString string) error {
    if j.store == nil {
        return nil
    }

    // 解析获取过期时间
    claims, err := j.Verify(ctx, tokenString)
    if err != nil {
        return err
    }

    // 撤销到过期时间
    ttl := time.Until(claims.ExpiresAt.Time)
    return j.store.Revoke(ctx, tokenString, ttl)
}
```

**关键变化**:
- 删除多签名算法支持
- 删除 SignOption
- 简化 Claims 结构
- 修复 TTL 使用 ExpiresAt

### 4.7 Repository 层（重命名 Store）

```go
// internal/user-center/repository/interface.go
package repository

import (
    "context"

    "github.com/kart/sentinel-x/internal/model"
)

// UserRepository 用户仓库接口
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, username string) error
    Get(ctx context.Context, username string) (*model.User, error)
    GetByID(ctx context.Context, id uint64) (*model.User, error)
    List(ctx context.Context, offset, limit int) (int64, []*model.User, error)
}

// 删除 Factory 接口，直接使用具体实现
```

```go
// internal/user-center/repository/mysql.go
package repository

import (
    "context"
    "errors"

    "gorm.io/gorm"
    "github.com/kart/sentinel-x/internal/model"
    apperrors "github.com/kart/sentinel-x/pkg/errors"
)

// MySQLUserRepository MySQL 用户仓库实现
type MySQLUserRepository struct {
    db *gorm.DB
}

// NewMySQLUserRepository 创建 MySQL 用户仓库
func NewMySQLUserRepository(db *gorm.DB) *MySQLUserRepository {
    return &MySQLUserRepository{db: db}
}

// Get 获取用户
func (r *MySQLUserRepository) Get(ctx context.Context, username string) (*model.User, error) {
    var user model.User
    err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, apperrors.ErrUserNotFound
        }
        return nil, apperrors.ErrDatabase.WithCause(err)
    }
    return &user, nil
}

// ... 其他方法
```

**关键变化**:
- 重命名 store -> repository（更清晰）
- 删除 Factory 接口
- 直接使用具体实现
- 线程安全由调用方保证

---

## 五、Handler 层重构

### 5.1 提取公共处理

```go
// internal/user-center/handler/base.go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/kart/sentinel-x/pkg/errors"
    "github.com/kart/sentinel-x/pkg/response"
)

// bindAndValidate 绑定并验证请求
func bindAndValidate[T any](c *gin.Context) (*T, bool) {
    var req T
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Fail(c, errors.ErrInvalidParam.WithMessage(err.Error()))
        return nil, false
    }
    return &req, true
}

// handleError 统一错误处理
func handleError(c *gin.Context, err error) {
    if errno, ok := err.(*errors.Errno); ok {
        response.Fail(c, errno)
    } else {
        response.Fail(c, errors.ErrInternal.WithCause(err))
    }
}
```

```go
// internal/user-center/handler/user.go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/kart/sentinel-x/internal/user-center/service"
    "github.com/kart/sentinel-x/pkg/response"
)

// UserHandler 用户处理器
type UserHandler struct {
    svc *service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(svc *service.UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

// Create 创建用户
func (h *UserHandler) Create(c *gin.Context) {
    req, ok := bindAndValidate[CreateUserRequest](c)
    if !ok {
        return
    }

    user, err := h.svc.Create(c.Request.Context(), req.Username, req.Password, req.Email)
    if err != nil {
        handleError(c, err)
        return
    }

    response.OK(c, user)
}

// Get 获取用户
func (h *UserHandler) Get(c *gin.Context) {
    username := c.Param("username")

    user, err := h.svc.Get(c.Request.Context(), username)
    if err != nil {
        handleError(c, err)
        return
    }

    response.OK(c, user)
}

// List 列出用户
func (h *UserHandler) List(c *gin.Context) {
    var req ListUserRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        response.Fail(c, errors.ErrInvalidParam)
        return
    }

    total, users, err := h.svc.List(c.Request.Context(), req.Offset, req.Limit)
    if err != nil {
        handleError(c, err)
        return
    }

    response.OKWithPagination(c, users, total, req.Offset, req.Limit)
}
```

**关键变化**:
- 使用泛型简化绑定
- 统一错误处理
- 删除重复代码
- 直接使用 gin.Context

---

## 六、路由注册（简化）

```go
// internal/user-center/router.go
package usercenter

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/kart/sentinel-x/internal/user-center/handler"
    "github.com/kart/sentinel-x/internal/user-center/repository"
    "github.com/kart/sentinel-x/internal/user-center/service"
    "github.com/kart/sentinel-x/pkg/middleware"
    "github.com/kart/sentinel-x/pkg/security/jwt"
)

// RegisterRoutes 注册路由
func RegisterRoutes(r *gin.Engine, db *gorm.DB, jwtAuth *jwt.JWT) {
    // 创建依赖
    userRepo := repository.NewMySQLUserRepository(db)
    userSvc := service.NewUserService(userRepo)
    authSvc := service.NewAuthService(userRepo, jwtAuth)

    userHandler := handler.NewUserHandler(userSvc)
    authHandler := handler.NewAuthHandler(authSvc)

    // 公开路由
    auth := r.Group("/auth")
    {
        auth.POST("/login", authHandler.Login)
        auth.POST("/register", authHandler.Register)
        auth.POST("/logout", authHandler.Logout)
    }

    // 需要认证的路由
    authMiddleware := middleware.Auth(jwtAuth)

    auth.GET("/me", authMiddleware, authHandler.Me)

    users := r.Group("/v1/users")
    users.POST("", userHandler.Create)  // 创建用户公开

    users.Use(authMiddleware)
    {
        users.GET("", userHandler.List)
        users.GET("/:username", userHandler.Get)
        users.PUT("/:username", userHandler.Update)
        users.DELETE("/:username", userHandler.Delete)
        users.POST("/:username/password", userHandler.ChangePassword)
    }
}
```

**关键变化**:
- 删除 router/ 子目录
- 删除依赖注入复杂性
- 直接在路由文件中创建依赖
- 简化的中间件注册

---

## 七、配置文件示例

```yaml
# configs/user-center.yaml
server:
  addr: ":8081"
  read-timeout: 30s
  write-timeout: 30s
  shutdown-timeout: 30s

log:
  level: info
  format: json

mysql:
  dsn: "user:password@tcp(localhost:3306)/user_auth?charset=utf8mb4&parseTime=True&loc=Local"
  max-idle-conns: 10
  max-open-conns: 100
  conn-max-lifetime: 1h

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool-size: 100

jwt:
  key: "${USER_CENTER_JWT_KEY}"  # 必须从环境变量读取
  expired: 2h
  max-refresh: 24h
  issuer: "sentinel-x"
```

**关键变化**:
- MySQL 使用单一 DSN
- 删除所有 Enable/Disable 对
- 最多 2 层嵌套
- 强制环境变量

---

## 八、迁移计划

### 8.1 Phase 1: 删除过度设计（第 1 周）

```bash
# 删除文件
rm -rf pkg/infra/adapter/echo/
rm -rf pkg/infra/server/transport/http/adapter.go
rm -rf pkg/infra/middleware/exports.go
rm -rf pkg/utils/id/snowflake.go
rm -rf pkg/utils/id/ulid.go
rm -rf internal/bootstrap/dependency.go

# 修改文件
# - pkg/infra/server/ -> pkg/server/
# - 删除 Bridge 层
```

### 8.2 Phase 2: 简化配置系统（第 1 周）

```bash
# 删除目录
rm -rf pkg/options/

# 创建新文件
touch pkg/config/config.go

# 迁移配置
# - 平坦化所有配置
# - 删除 Validate() 方法
```

### 8.3 Phase 3: 重构应用启动（第 2 周）

```bash
# 删除目录
rm -rf internal/bootstrap/

# 创建新文件
mkdir -p internal/app
touch internal/app/app.go
touch internal/app/options.go
touch internal/app/run.go
```

### 8.4 Phase 4: 简化错误系统（第 2 周）

```bash
# 修改文件
# - 删除 12 个创建函数
# - 合并为 3 个构建器
# - 更新所有调用点
```

### 8.5 Phase 5: 更新业务代码（第 3 周）

```bash
# 重命名
mv internal/user-center/biz internal/user-center/service
mv internal/user-center/store internal/user-center/repository

# 删除
rm -rf internal/user-center/router/

# 创建
touch internal/user-center/router.go
```

### 8.6 Phase 6: 测试和文档（第 4 周）

```bash
# 运行测试
make test

# 更新文档
# - 更新 README
# - 更新设计文档
# - 更新 API 文档
```

---

## 九、验收标准

### 9.1 代码指标

| 指标 | 当前 | 目标 | 改进 |
|------|------|------|------|
| 代码行数 | ~50K | ~35K | -30% |
| 文件数 | 150+ | 80 | -47% |
| 接口数 | 25+ | 10 | -60% |
| 配置类型 | 30+ | 8 | -73% |
| 启动时间 | 基准 | -20% | 显著 |

### 9.2 架构检查

- [ ] 无 Echo 相关代码
- [ ] 无 Bridge/Adapter 层
- [ ] 无拓扑排序代码
- [ ] 无 Enable/Disable 对
- [ ] 无类型别名导出
- [ ] 配置嵌套 ≤2 层
- [ ] 错误构建器 ≤3 个

### 9.3 功能验证

- [ ] 用户 CRUD 正常
- [ ] JWT 认证正常
- [ ] 健康检查正常
- [ ] 指标收集正常
- [ ] 优雅关闭正常

---

## 十、风险评估

### 10.1 破坏性变更

| 变更 | 影响 | 缓解措施 |
|------|------|----------|
| 删除 Echo 支持 | 使用 Echo 的项目 | 提供迁移指南 |
| 配置结构变化 | 现有配置文件 | 提供配置转换脚本 |
| API 变化 | 现有客户端 | 版本号升级到 v2 |

### 10.2 回滚计划

1. 保留当前版本的 Git 标签 (v1.x)
2. 新设计在新分支开发 (v2-redesign)
3. 完成后合并到 master 并标记 v2.0
4. 如需回滚，切回 v1.x 标签

---

## 附录

### A. 删除文件完整清单

```
pkg/infra/adapter/echo/
pkg/infra/adapter/gin/bridge.go
pkg/infra/server/transport/http/adapter.go
pkg/infra/middleware/exports.go
pkg/utils/id/snowflake.go
pkg/utils/id/ulid.go
internal/bootstrap/dependency.go
pkg/options/app/
pkg/options/server/
pkg/options/middleware/
pkg/options/logger/
pkg/options/auth/
pkg/options/mysql/
pkg/options/postgres/
pkg/options/redis/
pkg/options/mongodb/
pkg/options/etcd/
pkg/options/tracing/
```

### B. 重命名清单

```
internal/user-center/biz -> internal/user-center/service
internal/user-center/store -> internal/user-center/repository
pkg/infra/datasource -> pkg/database
pkg/infra/server -> pkg/server
pkg/infra/middleware -> pkg/middleware
```

### C. 新增文件清单

```
internal/app/app.go
internal/app/options.go
internal/app/run.go
pkg/config/config.go
pkg/errors/builder.go
internal/user-center/router.go
internal/user-center/handler/base.go
```

---

**文档版本**: 1.0
**最后更新**: 2025-12-11
**状态**: 待审批
