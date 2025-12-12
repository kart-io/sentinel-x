# User Center 设计文档

## 1. 概述

User Center（用户中心）是 Sentinel-X 平台的核心服务之一，提供用户管理、认证授权和配置管理等功能。该服务采用分层架构设计，通过适配器模式实现框架无关的代码结构，支持 Gin 和 Echo 两种 HTTP 框架的无缝切换。

### 1.1 设计目标

| 目标 | 描述 |
|------|------|
| **统一用户管理** | 提供用户创建、查询、更新、删除等完整的生命周期管理 |
| **安全认证** | 基于 JWT 的令牌认证，支持令牌签发、验证和撤销 |
| **框架无关** | 通过适配器模式支持多种 HTTP 框架（Gin/Echo） |
| **高可用** | 支持多实例部署，无状态设计 |
| **可观测** | 完整的日志、指标和健康检查支持 |

### 1.2 核心职责

| 职责 | 描述 |
|------|------|
| **用户管理** | 用户 CRUD 操作、密码管理、状态管理 |
| **认证服务** | 用户登录、登出、注册、令牌管理 |
| **配置管理** | 用户 Profile 管理 |

## 2. 系统架构

### 2.1 整体架构

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                              客户端                                      │
│              (Web UI / CLI / SDK / 其他微服务)                           │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         User Center Service                              │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                       Gateway Layer                                │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │                  HTTP Server (transport)                     │  │  │
│  │  │  ┌─────────────────────────────────────────────────────┐    │  │  │
│  │  │  │              FrameworkBridge (适配器)                │    │  │  │
│  │  │  │    ┌───────────────┐      ┌───────────────┐         │    │  │  │
│  │  │  │    │  Gin Bridge   │  OR  │  Echo Bridge  │         │    │  │  │
│  │  │  │    │   :8081       │      │   :8081       │         │    │  │  │
│  │  │  │    └───────────────┘      └───────────────┘         │    │  │  │
│  │  │  └─────────────────────────────────────────────────────┘    │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │                  gRPC Server (待扩展)                        │  │  │
│  │  │                      :8101                                   │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────┴───────────────────────────────────┐  │
│  │                      Middleware Layer                              │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐│  │
│  │  │ Recovery │ │RequestID │ │  Logger  │ │   CORS   │ │   Auth   ││  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘│  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────┴───────────────────────────────────┐  │
│  │                       Handler Layer                                │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │              transport.Context (框架无关接口)                │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────────────┐        ┌─────────────────────┐          │  │
│  │  │    UserHandler      │        │    AuthHandler      │          │  │
│  │  └─────────────────────┘        └─────────────────────┘          │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────┴───────────────────────────────────┐  │
│  │                         Biz Layer                                  │  │
│  │  ┌─────────────────────┐        ┌─────────────────────┐          │  │
│  │  │    UserService      │        │    AuthService      │          │  │
│  │  └─────────────────────┘        └─────────────────────┘          │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────┴───────────────────────────────────┐  │
│  │                        Store Layer                                 │  │
│  │  ┌─────────────────────┐        ┌─────────────────────┐          │  │
│  │  │   store.Factory     │        │    UserStore        │          │  │
│  │  └─────────────────────┘        └─────────────────────┘          │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            数据层                                        │
│  ┌─────────────────────┐        ┌─────────────────────┐                 │
│  │       MySQL         │        │       Redis         │                 │
│  │   (用户数据存储)     │        │   (令牌撤销列表)    │                 │
│  └─────────────────────┘        └─────────────────────┘                 │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 HTTP 框架适配器模式

```text
┌─────────────────────────────────────────────────────────────────────┐
│              transport.Context (框架无关接口)                        │
│    Request(), Param(), Query(), Bind(), JSON(), Validate()          │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              │                 │                 │
              ▼                 ▼                 ▼
      ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
      │  Gin Bridge  │  │ Echo Bridge  │  │ 其他 Bridge  │
      │  (adapter/)  │  │  (adapter/)  │  │   (可扩展)   │
      └──────┬───────┘  └──────┬───────┘  └──────────────┘
             │                 │
             ▼                 ▼
      ┌──────────────┐  ┌──────────────┐
      │  gin.Engine  │  │  echo.Echo   │
      └──────────────┘  └──────────────┘
```

**适配器注册机制**：
```go
// 在 app.go 中导入适配器，自动注册到全局注册表
import (
    _ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
    _ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
)

// 各适配器的 init() 函数自动注册
func init() {
    httpserver.RegisterBridge(httpopts.AdapterGin, NewBridge)
}
```

**框架切换方式**：仅需修改配置文件，代码零改动
```yaml
server:
  http:
    adapter: gin   # 改为 echo 即可切换框架
```

### 2.3 分层架构说明

| 层级 | 职责 | 核心组件 |
|------|------|----------|
| **Gateway** | 协议适配、请求接收 | HTTP Server (Gin/Echo Bridge), gRPC Server |
| **Middleware** | 横切关注点处理 | Recovery, RequestID, Logger, CORS, Auth, Timeout |
| **Handler** | 请求解析、响应构建 | UserHandler, AuthHandler (使用 transport.Context) |
| **Biz** | 业务逻辑编排 | UserService, AuthService |
| **Store** | 数据持久化 | Factory, UserStore (GORM) |

### 2.4 请求处理流程

```text
HTTP 请求
    │
    ▼
FrameworkBridge (Gin/Echo)
    │
    ├── 创建 RequestContext (框架无关)
    │
    ▼
Middleware Chain
    │
    ├── Recovery (Panic 恢复)
    ├── RequestID (请求追踪)
    ├── Logger (访问日志)
    ├── CORS (跨域)
    ├── Timeout (超时控制)
    └── Auth (JWT 认证，可选)
    │
    ▼
Handler (transport.Context)
    │
    ├── c.ShouldBindAndValidate() (参数解析验证)
    ├── bizService.Method() (调用业务层)
    └── c.JSON() (响应输出)
    │
    ▼
Biz Service
    │
    └── store.Users().Method() (数据访问)
    │
    ▼
Store (GORM + MySQL)
```

## 3. 目录结构

```text
cmd/user-center/
└── main.go                          # 程序入口

internal/user-center/
├── app.go                           # 应用实例和启动逻辑
├── options.go                       # 配置选项定义
├── handler/
│   ├── doc.go                       # 包文档
│   ├── user.go                      # 用户处理器
│   └── auth.go                      # 认证处理器
├── biz/
│   ├── doc.go                       # 包文档
│   ├── user.go                      # 用户业务逻辑
│   └── auth.go                      # 认证业务逻辑
├── store/
│   ├── doc.go                       # 包文档
│   ├── store.go                     # 接口定义
│   ├── mysql.go                     # MySQL 工厂实现
│   └── user.go                      # 用户数据操作
└── router/
    └── router.go                    # 路由注册

internal/model/
├── user.go                          # 用户模型
└── auth.go                          # 认证模型

internal/bootstrap/
├── bootstrapper.go                  # 启动编排器
├── logging.go                       # 日志初始化
├── datasource.go                    # 数据源初始化
├── auth.go                          # 认证初始化
├── middleware.go                    # 中间件初始化
└── server.go                        # 服务器初始化

pkg/infra/adapter/
├── gin/
│   └── bridge.go                    # Gin 适配器实现
└── echo/
    └── bridge.go                    # Echo 适配器实现

configs/
└── user-center.yaml                 # 配置文件
```

## 4. 核心组件设计

### 4.1 Options 配置

```go
// Options 包含所有 User Center Service 配置
type Options struct {
    Server *serveropts.Options   // HTTP/gRPC 服务配置
    Log    *logopts.Options      // 日志配置
    JWT    *jwtopts.Options      // JWT 认证配置
    MySQL  *mysqlopts.Options    // MySQL 配置
    Redis  *redisopts.Options    // Redis 配置
}

func (o *Options) Validate() error { /* 验证所有子配置 */ }
func (o *Options) Complete() error { /* 完善默认值 */ }
func (o *Options) AddFlags(fs *pflag.FlagSet) { /* 添加命令行标志 */ }
```

**配置来源优先级**（从高到低）：
1. 命令行标志（flags）
2. 环境变量（前缀：`USER_CENTER_`）
3. 配置文件（YAML）
4. 默认值

### 4.2 Bootstrap 启动流程

```text
main.go
    │
    ▼
NewApp()
    │
    ├── NewOptions()                 # 创建配置对象
    │
    └── Run(opts)
           │
           ▼
         bootstrap.Run(bootstrapOpts)
           │
           ├── AppBootstrapper.Initialize()
           │   │
           │   ├── 1. LoggingInitializer      # 日志初始化
           │   ├── 2. DatasourceInitializer   # MySQL + Redis 连接
           │   ├── 3. AuthInitializer         # JWT 认证初始化
           │   ├── 4. MiddlewareInitializer   # HTTP 中间件初始化
           │   └── 5. ServerInitializer       # HTTP/gRPC 服务初始化
           │
           └── router.Register()              # 注册路由和业务逻辑
```

**初始化器依赖链**：
```text
LoggingInitializer
    ↓
DatasourceInitializer → MySQL DB + Redis Client
    ↓
AuthInitializer → JWT (需要 Redis 存储令牌撤销列表)
    ↓
MiddlewareInitializer → 注册中间件
    ↓
ServerInitializer → HTTP Server (选择 Gin/Echo Bridge)
```

### 4.3 Store 接口设计

```go
// Factory 定义 Store 工厂接口
type Factory interface {
    Users() UserStore
    Close() error
}

// UserStore 定义用户存储接口
type UserStore interface {
    Create(ctx context.Context, user *model.User) error
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, username string) error
    Get(ctx context.Context, username string) (*model.User, error)
    GetByUserId(ctx context.Context, userId uint64) (*model.User, error)
    List(ctx context.Context, offset, limit int) (int64, []*model.User, error)
}
```

**设计特点**：
- 工厂模式：支持多数据源实现切换
- 接口隔离：高层模块依赖抽象而非具体实现
- 上下文传递：所有方法支持 context 用于超时控制和链路追踪

### 4.4 transport.Context 接口

```go
// Context 框架无关的请求上下文接口
type Context interface {
    Request() context.Context
    HTTPRequest() *http.Request
    Param(key string) string
    Query(key string) string
    Header(key string) string
    Bind(v interface{}) error
    ShouldBindAndValidate(v interface{}) error
    JSON(code int, v interface{})
    GetRawContext() interface{}
}
```

**实现方式**：
- Gin Bridge 使用 `*gin.Context` 实现
- Echo Bridge 使用 `echo.Context` 实现
- Handler 层统一使用 `transport.Context`，无需关心底层框架

## 5. API 设计

### 5.1 认证接口

| 方法 | 路径 | 需要认证 | 描述 |
|------|------|----------|------|
| POST | `/auth/login` | 否 | 用户登录 |
| POST | `/auth/logout` | 否 | 用户登出（需提供 token） |
| POST | `/auth/register` | 否 | 用户注册 |
| GET | `/auth/me` | 是 | 获取当前用户信息 |

### 5.2 用户管理接口

| 方法 | 路径 | 需要认证 | 描述 |
|------|------|----------|------|
| POST | `/v1/users` | 否 | 创建用户 |
| GET | `/v1/users` | 是 | 列出用户（分页） |
| GET | `/v1/users/:username` | 是 | 获取单个用户 |
| PUT | `/v1/users/:username` | 是 | 更新用户 |
| DELETE | `/v1/users/:username` | 是 | 删除用户 |
| POST | `/v1/users/:username/password` | 是 | 修改密码 |

### 5.3 请求/响应模型

**登录请求**：
```json
{
    "username": "string (必填)",
    "password": "string (必填)"
}
```

**登录响应**：
```json
{
    "token": "string (JWT 令牌)",
    "expires_in": "int64 (过期时间戳)",
    "user_id": "uint64 (用户 ID)"
}
```

**注册请求**：
```json
{
    "username": "string (必填)",
    "password": "string (必填)",
    "email": "string (必填，需符合邮箱格式)"
}
```

**用户更新请求**（使用专用 DTO 防止敏感字段更新）：
```json
{
    "email": "string (可选)",
    "avatar": "string (可选)",
    "mobile": "string (可选)",
    "status": "int (可选，0 或 1)"
}
```

## 6. 数据模型

### 6.1 User 模型

```go
type User struct {
    ID        uint64         `json:"id" gorm:"primaryKey;autoIncrement"`
    Username  string         `json:"username" gorm:"size:64;not null;uniqueIndex:uk_username"`
    Email     *string        `json:"email" gorm:"size:128;uniqueIndex:uk_email"`
    Password  string         `json:"-" gorm:"size:255;not null"`  // 不序列化
    Avatar    string         `json:"avatar" gorm:"size:255"`
    Mobile    string         `json:"mobile" gorm:"size:20;index:idx_mobile"`
    Status    int            `json:"status" gorm:"default:1;index:idx_status"`
    CreatedAt int64          `json:"created_at" gorm:"autoCreateTime:milli"`
    UpdatedAt int64          `json:"updated_at" gorm:"autoUpdateTime:milli"`
    CreatedBy uint64         `json:"created_by" gorm:"default:0"`
    UpdatedBy uint64         `json:"updated_by" gorm:"default:0"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`  // 软删除
}
```

### 6.2 数据库索引

| 字段 | 索引类型 | 说明 |
|------|----------|------|
| `id` | 主键 | 自增主键 |
| `username` | 唯一索引 | 用户名唯一约束 |
| `email` | 唯一索引 | 邮箱唯一约束 |
| `mobile` | 普通索引 | 手机号查询优化 |
| `status` | 普通索引 | 状态过滤优化 |
| `deleted_at` | 普通索引 | 软删除查询优化 |

### 6.3 GORM 钩子

```go
// BeforeCreate 创建前设置时间戳
func (u *User) BeforeCreate(tx *gorm.DB) error {
    now := time.Now().UnixMilli()
    u.CreatedAt = now
    u.UpdatedAt = now
    return nil
}

// BeforeUpdate 更新前设置更新时间
func (u *User) BeforeUpdate(tx *gorm.DB) error {
    u.UpdatedAt = time.Now().UnixMilli()
    return nil
}
```

## 7. 业务逻辑实现

### 7.1 UserService

| 方法 | 功能 | 关键实现 |
|------|------|----------|
| `Create` | 创建用户 | bcrypt 密码加密 |
| `Update` | 更新用户 | 通过 Store 接口操作 |
| `Delete` | 删除用户 | 软删除 |
| `Get` | 获取用户 | 按用户名查询 |
| `GetByUserId` | 按 ID 获取 | 按用户 ID 查询 |
| `List` | 列出用户 | 分页查询，窗口函数优化 |
| `ChangePassword` | 修改密码 | 获取用户 → 加密新密码 → 更新 |
| `ValidatePassword` | 验证密码 | bcrypt.CompareHashAndPassword |

### 7.2 AuthService

| 方法 | 功能 | 关键实现 |
|------|------|----------|
| `Login` | 用户登录 | 验证密码 → 检查状态 → 生成 JWT |
| `Logout` | 用户登出 | 撤销 JWT 令牌 |
| `Register` | 用户注册 | 检查重复 → 加密密码 → 创建用户 |

**登录流程**：
```text
Login Request
    │
    ▼
获取用户信息 (store.Users().Get)
    │
    ▼
验证密码 (bcrypt.CompareHashAndPassword)
    │
    ├── 失败: 返回 "无效的用户名或密码"
    │
    ▼
检查用户状态
    │
    ├── Status == 0: 返回 "账号已被禁用"
    │
    ▼
生成 JWT 令牌 (jwtAuth.Sign)
    │
    ▼
返回 LoginResponse
```

## 8. 安全设计

### 8.1 密码安全

- **加密算法**：bcrypt（默认成本因子）
- **存储方式**：仅存储密码哈希，原始密码不持久化
- **JSON 排除**：`json:"-"` 标签确保密码不会被序列化

### 8.2 认证安全

- **JWT 签名**：支持 HS256/HS384/HS512/RS256 等算法
- **令牌撤销**：支持通过 Redis 存储撤销列表
- **过期控制**：可配置令牌过期时间和最大刷新时间

### 8.3 接口安全

- **认证中间件**：保护需要认证的路由
- **专用 DTO**：`UpdateUserRequest` 明确排除敏感字段（Password、Username）
- **输入验证**：使用 validator 进行参数校验

## 9. 配置说明

### 9.1 服务器配置

```yaml
server:
  mode: both                    # http, grpc, 或 both
  shutdown-timeout: 30s         # 优雅关闭超时

  http:
    addr: ":8081"              # HTTP 监听地址
    read-timeout: 30s
    write-timeout: 30s
    idle-timeout: 60s
    adapter: gin               # HTTP 框架: gin 或 echo

  grpc:
    addr: ":8101"              # gRPC 监听地址
    enable-reflection: true    # 启用反射服务
```

### 9.2 中间件配置

```yaml
middleware:
  disable-recovery: false      # Panic 恢复
  disable-request-id: false    # 请求 ID 追踪
  disable-logger: false        # 结构化日志
  disable-cors: false          # 跨域支持
  disable-timeout: false       # 请求超时
  disable-health: false        # 健康检查
  disable-metrics: false       # Prometheus 指标
  disable-pprof: true          # pprof 调试（生产禁用）
  disable-auth: false          # JWT 认证
  disable-authz: false         # 授权检查
```

### 9.3 数据库配置

```yaml
mysql:
  host: "localhost"
  port: 3306
  username: "root"
  password: ""                 # 通过环境变量设置
  database: "user_auth"
  max-idle-connections: 10
  max-open-connections: 100
  max-connection-life-time: 3600s

redis:
  host: "localhost"
  port: 6379
  password: ""                 # 通过环境变量设置
  database: 0
  pool-size: 10
```

### 9.4 环境变量

| 环境变量 | 说明 |
|----------|------|
| `USER_CENTER_JWT_KEY` | JWT 签名密钥（≥64 字符） |
| `USER_CENTER_MYSQL_HOST` | MySQL 主机地址 |
| `USER_CENTER_MYSQL_PASSWORD` | MySQL 密码 |
| `USER_CENTER_REDIS_HOST` | Redis 主机地址 |
| `USER_CENTER_REDIS_PASSWORD` | Redis 密码 |

## 10. 性能优化

### 10.1 数据库查询优化

**列表查询窗口函数**：
```sql
SELECT
    id, username, email, avatar, mobile, status,
    created_at, updated_at, created_by, updated_by,
    COUNT(*) OVER() as total_count
FROM users
OFFSET ? LIMIT ?
```

**优化点**：
- 单次查询获取数据和总数
- 避免 N+1 查询问题
- 明确指定字段，排除敏感字段（password）

### 10.2 响应池化

使用 `response.Release(resp)` 释放响应对象到对象池，减少 GC 压力。

### 10.3 上下文复用

Gin Bridge 中的 `RequestContext` 会被缓存到 `*gin.Context` 中，确保中间件链共享同一实例：
```go
func (b *Bridge) getOrCreateContext(c *gin.Context) *httpserver.RequestContext {
    if val, exists := c.Get(requestContextKey); exists {
        return val.(*httpserver.RequestContext)
    }
    ctx := b.createContext(c)
    c.Set(requestContextKey, ctx)
    return ctx
}
```

## 11. 可观测性

### 11.1 健康检查端点

| 路径 | 说明 |
|------|------|
| `/health` | 综合健康检查 |
| `/live` | 存活探针 |
| `/ready` | 就绪探针 |

### 11.2 指标端点

- **路径**：`/metrics`
- **格式**：Prometheus
- **命名空间**：`sentinel`
- **子系统**：`api`

### 11.3 日志配置

```yaml
log:
  level: info
  format: json
  output-paths: [stdout]
  error-output-paths: [stderr]
  enable-caller: true
  enable-stacktrace: false
```

## 12. 运行指南

### 12.1 编译

```bash
# 使用 Makefile
make build-user-center

# 或直接编译
go build -o user-center ./cmd/user-center
```

### 12.2 运行

```bash
# 使用默认配置
./user-center

# 指定配置文件
./user-center -c configs/user-center.yaml

# 使用环境变量
export USER_CENTER_JWT_KEY="your-secret-key-at-least-64-characters"
export USER_CENTER_MYSQL_PASSWORD="mysql-password"
./user-center
```

### 12.3 开发模式

```bash
make run-dev
```

## 13. 扩展性设计

### 13.1 添加新的 HTTP 框架

实现 `FrameworkBridge` 接口并注册：

```go
// 在 pkg/infra/adapter/newframework/bridge.go
func init() {
    httpserver.RegisterBridge(httpopts.AdapterNewFramework, NewBridge)
}

type Bridge struct {
    engine *newframework.Engine
}

func (b *Bridge) AddRoute(method, path string, handler httpserver.BridgeHandler) {
    // 实现路由注册
}

func (b *Bridge) wrapHandler(handler httpserver.BridgeHandler) newframework.HandlerFunc {
    return func(c *newframework.Context) {
        ctx := b.createContext(c)
        handler(ctx)
    }
}
```

### 13.2 添加新的存储后端

实现 `Factory` 和 `UserStore` 接口：

```go
type PostgresFactory struct {
    db *gorm.DB
}

func (f *PostgresFactory) Users() UserStore {
    return &postgresUsers{db: f.db}
}
```

### 13.3 gRPC 服务扩展

预留了 gRPC 服务注册位置：

```go
if grpcServer := mgr.GRPCServer(); grpcServer != nil {
    // 注册 gRPC 服务
    // pb.RegisterUserServiceServer(grpcServer.Server, userGRPCHandler)
}
```

## 14. 设计模式总结

| 模式 | 应用位置 | 目的 |
|------|---------|------|
| **工厂模式** | Store, Biz, Handler 创建 | 延迟初始化、依赖注入 |
| **适配器模式** | HTTP 框架选择 (Gin/Echo) | 框架无关的代码 |
| **桥接模式** | RequestContext + FrameworkBridge | 分离抽象和实现 |
| **策略模式** | 中间件链 | 灵活的中间件组合 |
| **单例模式** | JWT, Logger, DataSource | 全局共享资源 |
| **职责链模式** | 中间件和路由组 | 请求处理管道 |

## 参考资料

- [Gin Web Framework](https://gin-gonic.com/docs/)
- [Echo Web Framework](https://echo.labstack.com/)
- [GORM 文档](https://gorm.io/docs/)
- [JWT Go](https://github.com/golang-jwt/jwt)
- [bcrypt 密码加密](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
