# Auth/Authz 模块设计文档

## 概述

本文档描述 Sentinel-X 框架的认证（Auth）和授权（Authz）模块的设计与实现。该模块遵循 sentinel-x 项目的架构理念，提供统一的安全能力。

## 设计原则

1. **接口驱动**：所有认证器和授权器都实现统一接口
2. **上下文感知**：通过 context 传递认证信息
3. **集中管理**：禁止在业务代码中直接解析 JWT
4. **可扩展性**：支持无侵入扩展为 OAuth2 / 第三方身份源

## 架构图

```text
┌─────────────────────────────────────────────────────────────────────┐
│                            请求流程                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────┐    ┌──────────────┐    ┌──────────────┐    ┌────────┐ │
│  │  Client  │───▶│ Auth Middleware │───▶│ Authz Middleware │───▶│ Handler│ │
│  └──────────┘    └──────────────┘    └──────────────┘    └────────┘ │
│       │                 │                    │                       │
│       │                 ▼                    ▼                       │
│       │          ┌────────────┐       ┌────────────┐                │
│       │          │    JWT     │       │    RBAC    │                │
│       │          │Authenticator│       │ Authorizer │                │
│       │          └────────────┘       └────────────┘                │
│       │                 │                    │                       │
│       │                 ▼                    ▼                       │
│       │          ┌────────────┐       ┌────────────┐                │
│       │          │ TokenStore │       │PolicyStore │                │
│       │          │  (Memory/  │       │ (Memory/   │                │
│       │          │   Redis)   │       │   DB)      │                │
│       │          └────────────┘       └────────────┘                │
│       │                                                              │
└───────┴──────────────────────────────────────────────────────────────┘
```

## 模块结构

```text
pkg/
├── options/
│   └── jwt/
│       └── options.go          # JWT 配置选项
├── auth/
│   ├── auth.go                 # 认证接口定义
│   ├── context.go              # 上下文辅助函数
│   └── jwt/
│       ├── jwt.go              # JWT 实现
│       └── store.go            # Token 存储接口
├── authz/
│   ├── authz.go                # 授权接口定义
│   └── rbac/
│       └── rbac.go             # RBAC 实现
└── middleware/
    ├── auth.go                 # HTTP 认证中间件
    ├── authz.go                # HTTP 授权中间件
    └── grpc/
        └── interceptors.go     # gRPC 拦截器
```

## 核心接口

### Authenticator 接口

```go
type Authenticator interface {
    // Sign 为指定主体创建令牌
    Sign(ctx context.Context, subject string, opts ...SignOption) (Token, error)

    // Verify 验证令牌并返回声明
    Verify(ctx context.Context, tokenString string) (*Claims, error)

    // Refresh 刷新令牌
    Refresh(ctx context.Context, tokenString string) (Token, error)

    // Revoke 撤销令牌
    Revoke(ctx context.Context, tokenString string) error

    // Type 返回认证器类型
    Type() string
}
```

### Authorizer 接口

```go
type Authorizer interface {
    // Authorize 检查主体是否可以对资源执行操作
    Authorize(ctx context.Context, subject, resource, action string) (bool, error)

    // AuthorizeWithContext 带额外上下文的授权检查
    AuthorizeWithContext(ctx context.Context, subject, resource, action string,
        context map[string]interface{}) (bool, error)
}
```

## JWT 配置选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `key` | string | - | 签名密钥（必填，≥32字符） |
| `signing-method` | string | HS256 | 签名算法 |
| `expired` | duration | 2h | 令牌过期时间 |
| `max-refresh` | duration | 24h | 最大刷新时间 |
| `issuer` | string | sentinel-x | 令牌签发者 |
| `audience` | []string | [] | 目标受众 |

### 支持的签名算法

- HMAC: `HS256`, `HS384`, `HS512`
- RSA: `RS256`, `RS384`, `RS512`
- ECDSA: `ES256`, `ES384`, `ES512`

## RBAC 权限模型

### 权限定义

```go
type Permission struct {
    Resource   string  // 资源名称（支持通配符 *）
    Action     string  // 操作名称（支持通配符 *）
    Effect     Effect  // allow 或 deny
    Conditions map[string]interface{} // 可选条件
}
```

### 角色管理

```go
// 创建角色
rbac.AddRole("admin", authz.NewPermission("*", "*"))
rbac.AddRole("editor",
    authz.NewPermission("posts", "*"),
    authz.NewPermission("users", "read"),
)

// 分配角色
rbac.AssignRole("user-123", "admin")

// 检查权限
allowed, _ := rbac.Authorize(ctx, "user-123", "posts", "delete")
```

### HTTP Method 到 Action 映射

| HTTP Method | Action |
|-------------|--------|
| GET | read |
| POST | create |
| PUT | update |
| PATCH | update |
| DELETE | delete |

## 中间件使用

### HTTP 中间件

```go
// 认证中间件
authMiddleware := middleware.Auth(
    middleware.AuthWithAuthenticator(jwtAuth),
    middleware.AuthWithSkipPaths("/login", "/health"),
    middleware.AuthWithTokenLookup("header:Authorization"),
)

// 授权中间件
authzMiddleware := middleware.Authz(
    middleware.AuthzWithAuthorizer(rbacAuthz),
    middleware.AuthzWithSkipPaths("/login", "/me"),
)
```

### gRPC 拦截器

```go
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcmw.ChainUnaryInterceptors(
        grpcmw.UnaryAuthInterceptor(
            grpcmw.WithAuthenticator(jwtAuth),
            grpcmw.WithSkipMethods("/api.Auth/Login"),
        ),
        grpcmw.UnaryAuthzInterceptor(
            grpcmw.WithAuthorizer(rbacAuthz),
        ),
    )),
)
```

## 上下文注入

认证成功后，以下信息会注入到 context 中：

```go
// 获取用户 ID
subject := auth.SubjectFromContext(ctx)

// 获取完整声明
claims := auth.ClaimsFromContext(ctx)

// 获取原始令牌
token := auth.TokenFromContext(ctx)

// 获取自定义声明
username := claims.GetExtraString("username")
```

## 错误码

| 错误码 | HTTP | gRPC | 说明 |
|--------|------|------|------|
| 0002000 | 401 | Unauthenticated | 未认证 |
| 0002001 | 401 | Unauthenticated | 令牌无效 |
| 0002002 | 401 | Unauthenticated | 令牌过期 |
| 0002003 | 401 | Unauthenticated | 凭证无效 |
| 0002004 | 401 | Unauthenticated | 令牌已撤销 |
| 0003000 | 403 | PermissionDenied | 禁止访问 |
| 0003001 | 403 | PermissionDenied | 无权限 |

## 令牌存储

### 内存存储（开发/单实例）

```go
store := jwt.NewMemoryStore(
    jwt.WithCleanupInterval(5 * time.Minute),
)
```

### Redis 存储（生产/分布式）

需自行实现 `Store` 接口：

```go
type Store interface {
    Revoke(ctx context.Context, token string, expiration time.Duration) error
    IsRevoked(ctx context.Context, token string) (bool, error)
    Close() error
}
```

## 扩展性设计

### 添加新的认证提供者

实现 `Authenticator` 接口即可：

```go
type OAuth2Authenticator struct {
    // ...
}

func (a *OAuth2Authenticator) Sign(ctx context.Context, subject string, opts ...auth.SignOption) (auth.Token, error) {
    // 实现 OAuth2 令牌生成
}

func (a *OAuth2Authenticator) Verify(ctx context.Context, token string) (*auth.Claims, error) {
    // 实现 OAuth2 令牌验证
}
```

### 添加新的授权策略

实现 `Authorizer` 接口：

```go
type CasbinAuthorizer struct {
    enforcer *casbin.Enforcer
}

func (a *CasbinAuthorizer) Authorize(ctx context.Context, sub, res, act string) (bool, error) {
    return a.enforcer.Enforce(sub, res, act)
}
```

## 安全最佳实践

1. **密钥管理**
   - 使用强随机密钥（≥32字符）
   - 定期轮换密钥
   - 使用环境变量或密钥管理服务存储密钥

2. **令牌安全**
   - 使用 HTTPS 传输
   - 设置合理的过期时间
   - 实现令牌撤销机制

3. **权限设计**
   - 遵循最小权限原则
   - 优先使用拒绝规则
   - 定期审计权限配置

## 运行示例

```bash
# 运行 Auth/Authz 示例
go run example/auth/main.go

# 登录获取令牌
curl -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 访问受保护资源
curl http://localhost:8082/api/v1/users \
  -H "Authorization: Bearer <token>"

# 刷新令牌
curl -X POST http://localhost:8082/api/v1/auth/refresh \
  -H "Authorization: Bearer <token>"

# 登出（撤销令牌）
curl -X POST http://localhost:8082/api/v1/auth/logout \
  -H "Authorization: Bearer <token>"
```

## 配置文件示例

```yaml
jwt:
  key: "your-secret-key-at-least-32-characters!"
  signing-method: "HS256"
  expired: "2h"
  max-refresh: "24h"
  issuer: "sentinel-x"
  audience:
    - "api"

middleware:
  disable-auth: false
  disable-authz: false
```
