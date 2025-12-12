# Sentinel-X 项目代码布局分析报告

> 生成时间：2025-12-11
> 分析范围：完整项目结构、模块职责、依赖关系

---

## 一、项目概览

### 1.1 项目简介

**Sentinel-X** 是一个基于 Go 语言的企业级微服务平台，采用 Monorepo 架构。项目集成了用户中心、认证授权、可观测性等核心功能，适用于构建分布式运维系统、微服务网关、API 管理平台等中大型应用。

### 1.2 技术栈

| 类别 | 技术选型 |
|------|----------|
| **语言** | Go 1.25.0 |
| **Web 框架** | Gin（主）、Echo（备用） |
| **数据库** | MySQL、PostgreSQL（GORM） |
| **缓存** | Redis（go-redis） |
| **认证** | JWT |
| **授权** | Casbin RBAC |
| **链路追踪** | OpenTelemetry |
| **日志** | github.com/kart-io/logger（Zap/Slog 双引擎） |
| **配置** | Viper（YAML） |
| **JSON** | Sonic（高性能） |

### 1.3 代码统计

| 指标 | 数量 |
|------|------|
| 总 Go 文件 | 237 个 |
| 测试文件 | 54 个 |
| pkg 目录文件 | 179 个 |
| internal 目录文件 | 26 个 |
| 文档文件 | 18 个 |
| 配置文件 | 3 个 |

---

## 二、目录结构

```text
sentinel-x/
├── bin/                          # 编译输出目录
├── cmd/                          # 应用程序入口
│   ├── api/                      # API Server
│   ├── user-center/              # 用户中心服务
│   └── scheduler/                # 调度器服务（占位）
├── internal/                     # 私有应用代码
│   ├── api/                      # API Server 应用层
│   ├── bootstrap/                # 启动引导逻辑
│   ├── model/                    # 数据模型
│   └── user-center/              # 用户中心业务逻辑
├── pkg/                          # 公共库
│   ├── component/                # 基础组件（数据库、缓存）
│   ├── infra/                    # 基础设施层
│   ├── security/                 # 安全组件
│   └── utils/                    # 通用工具
├── configs/                      # 配置文件
├── docs/                         # 项目文档
├── example/                      # 代码示例
├── examples/                     # 高级示例
├── scripts/                      # 脚本工具
├── staging/                      # 内部依赖库
├── hack/                         # 维护脚本
├── vendor/                       # 第三方依赖
└── .claude/                      # Claude Code 工作目录
```

---

## 三、模块职责说明

### 3.1 应用入口层（cmd/）

#### cmd/api

- **职责**：Sentinel-X API Server 主入口
- **关键文件**：`main.go`
- **启动流程**：
  1. 创建 `api.App` 实例
  2. 调用 `Run()` 触发 bootstrap 流程
  3. 初始化日志、数据源、中间件、服务器
  4. 启动 HTTP/gRPC 监听

```go
// cmd/api/main.go
func main() {
    api.NewApp().Run()
}
```

#### cmd/user-center

- **职责**：用户中心服务入口
- **关键文件**：`main.go`
- **特点**：
  - 注册自定义路由
  - 复用公共 bootstrap 逻辑
  - 独立的业务处理层

#### cmd/scheduler

- **职责**：调度器服务入口
- **状态**：占位符，待实现

---

### 3.2 启动引导层（internal/bootstrap/）

负责应用生命周期管理的核心模块。

| 文件 | 职责 |
|------|------|
| `bootstrapper.go` | 协调所有初始化器，实现 AppBootstrapper 模式 |
| `initializer.go` | 定义 Initializer 和 Shutdowner 接口 |
| `logging.go` | 初始化 Logger（Zap/Slog 引擎） |
| `datasource.go` | 初始化 MySQL、Redis 等数据源 |
| `server.go` | 初始化 HTTP/gRPC 服务器 |
| `middleware.go` | 注册中间件栈 |
| `auth.go` | 初始化 JWT 认证 |
| `run.go` | 主运行循环和优雅关闭 |

**初始化顺序**：

```text
Logging → Datasource → Auth → Middleware → Server
```

---

### 3.3 用户中心服务（internal/user-center/）

采用标准的分层架构 + Repository 模式。

```text
user-center/
├── handler/     ← HTTP 请求处理层
├── router/      ← 路由配置
├── biz/         ← 业务逻辑层
├── store/       ← 数据访问层
└── pkg/         ← 包内工具
```

#### handler/ - HTTP 处理层

| 文件 | 职责 |
|------|------|
| `auth.go` | 认证相关 HTTP 处理（登录、注册、Token 刷新） |
| `user.go` | 用户管理 HTTP 处理（CRUD、查询） |

#### biz/ - 业务逻辑层

| 文件 | 职责 |
|------|------|
| `auth.go` | 认证业务逻辑（密码验证、Token 生成） |
| `user.go` | 用户业务逻辑（创建、更新、删除） |

#### store/ - 数据访问层

| 文件 | 职责 |
|------|------|
| `store.go` | 存储接口定义（Repository Pattern） |
| `mysql.go` | MySQL 实现 |
| `user.go` | 用户数据访问实现 |

#### router/ - 路由配置

| 文件 | 职责 |
|------|------|
| `router.go` | 路由注册、中间件绑定 |

---

### 3.4 基础设施层（pkg/infra/）

#### adapter/ - Web 框架适配器

支持多框架集成：

- **gin/**：Gin Web 框架适配
- **echo/**：Echo Web 框架适配

#### app/ - 应用管理

| 文件 | 职责 |
|------|------|
| `app.go` | 应用生命周期管理 |
| `version.go` | 版本信息 |
| `logger.go` | 日志初始化 |
| `options.go` | 配置选项解析 |

#### datasource/ - 数据源管理

| 文件 | 职责 |
|------|------|
| `manager.go` | 统一管理所有数据源 |
| `generic.go` | 通用数据源实现 |
| `clients.go` | 客户端池管理 |

#### logger/ - 日志管理

| 文件 | 职责 |
|------|------|
| `context.go` | 日志上下文传递 |
| `fields.go` | 标准化日志字段 |
| `reloadable.go` | 热重载支持 |
| `options.go` | 日志配置选项 |

#### middleware/ - HTTP/gRPC 中间件

共 22 种中间件：

| 中间件 | 文件 | 功能 |
|--------|------|------|
| Recovery | `recovery.go` | Panic 恢复，堆栈追踪 |
| RequestID | `request_id.go` | 请求 ID 生成和传递 |
| Logger | `logger.go` | 请求日志记录 |
| Logger Enhanced | `logger_enhanced.go` | 增强的结构化日志 |
| CORS | `cors.go` | 跨域资源共享 |
| Timeout | `timeout.go` | 请求超时控制 |
| Health | `health.go` | 健康检查端点 |
| Metrics | `metrics.go` | Prometheus 指标 |
| Pprof | `pprof.go` | 性能分析端点 |
| Auth | `auth.go` | JWT 认证验证 |
| Authz | `authz.go` | 角色授权检查 |
| Security Headers | `security_headers.go` | HTTP 安全头注入 |
| RateLimit | `ratelimit.go` | 速率限制 |
| Tracing | `tracing.go` | OpenTelemetry 链路追踪 |

#### server/ - 服务器管理

- **http/**：HTTP 服务器实现
- **grpc/**：gRPC 服务器实现
- **transport/**：传输协议适配
- **service/**：服务注册和发现

#### tracing/ - 链路追踪

| 文件 | 职责 |
|------|------|
| `provider.go` | 追踪提供者配置 |
| `context.go` | 上下文传递 |
| `options.go` | 追踪配置选项 |

---

### 3.5 基础组件层（pkg/component/）

数据库/缓存驱动集合：

| 组件 | 文件数 | 说明 |
|------|--------|------|
| **mysql/** | 8 | GORM MySQL 驱动、DSN 生成、连接池管理 |
| **postgres/** | 5 | GORM PostgreSQL 驱动 |
| **mongodb/** | 多 | MongoDB 驱动 |
| **redis/** | 3 | go-redis 客户端集成 |
| **etcd/** | 多 | ETCD 分布式配置 |
| **storage/** | 多 | 统一存储管理器 |

#### MySQL 组件详析

| 文件 | 职责 |
|------|------|
| `factory.go` | 工厂方法 |
| `client.go` | 连接管理 |
| `dsn.go` | DSN 生成和验证 |
| `options.go` | 配置选项 |
| `health.go` | 健康检查 |
| `logger.go` | 日志集成 |

---

### 3.6 安全模块（pkg/security/）

#### auth/ - 认证

**jwt/** - JWT Token 管理：

| 文件 | 职责 |
|------|------|
| `token.go` | Token 生成和验证 |
| `claims.go` | Claims 定义 |
| `options.go` | JWT 配置 |

#### authz/ - 授权

**casbin/** - RBAC 权限控制：

- `infrastructure/mysql/`：MySQL 后端存储
- `infrastructure/redis/`：Redis 缓存加速
- `enforcer.go`：Casbin 执行器

---

### 3.7 工具库（pkg/utils/）

#### errors/ - 错误处理

统一错误码系统：

| 文件 | 职责 |
|------|------|
| `errno.go` | 错误码定义和注册 |
| `registry.go` | 错误码注册表 |
| `code.go` | 错误码常量 |
| `builder.go` | 错误构建器 |
| `base.go` | 基础错误类型 |
| `helpers.go` | 错误辅助函数 |

#### id/ - ID 生成

多种 ID 生成策略：

| 文件 | 职责 |
|------|------|
| `snowflake.go` | 雪花算法（分布式 ID） |
| `ulid.go` | ULID（可排序的 ID） |
| `uuid.go` | UUID（标准 UUID） |
| `id.go` | ID 接口定义 |

#### json/ - JSON 处理

高性能 JSON 编解码，集成 Sonic 库。

#### response/ - 响应管理

| 文件 | 职责 |
|------|------|
| `response.go` | 响应结构定义 |
| `writer.go` | 响应编写器 |

#### validator/ - 参数验证

基于 `go-playground/validator`：

| 文件 | 职责 |
|------|------|
| `validator.go` | 验证器封装 |
| `rules.go` | 自定义验证规则 |
| `translations.go` | 多语言错误提示 |
| `errors.go` | 验证错误处理 |

---

## 四、架构图

### 4.1 整体架构

```text
┌─────────────────────────────────────────────────────────┐
│                   Client / Frontend                     │
└──────────┬──────────────────────────────────────────────┘
           │
           ├─────────────────────────────────────────────┐
           │                                             │
     ┌─────▼─────────────────┐        ┌──────────────────▼────┐
     │   HTTP Server         │        │  gRPC Server          │
     │ (Gin/Echo)            │        │                       │
     └─────┬─────────────────┘        └──────────────────┬────┘
           │                                             │
     ┌─────▼─────────────────────────────────────────────▼────┐
     │              Middleware Stack                          │
     │ Recovery → RequestID → Logger → CORS → Timeout         │
     │ → Health → Auth → Authz → Metrics → Tracing ...       │
     └─────┬──────────────────────────────────────────────────┘
           │
     ┌─────▼─────────────────────────────────────────────────┐
     │       Handler / Router Layer                          │
     └─────┬──────────────────────────────────────────────────┘
           │
     ┌─────▼─────────────────────────────────────────────────┐
     │       Business Logic Layer (Biz)                      │
     └─────┬──────────────────────────────────────────────────┘
           │
     ┌─────▼─────────────────────────────────────────────────┐
     │       Data Access Layer (Store/Repository)            │
     └─────┬──────────────────────────────────────────────────┘
           │
     ┌─────▼──────────────────┬─────────────────┬────────────┐
     │                        │                 │            │
┌────▼──────────┐  ┌─────────▼─────┐  ┌───────▼──┐  ┌──────▼──┐
│  MySQL/PG     │  │  Redis Cache  │  │  ETCD    │  │ MongoDB │
└───────────────┘  └───────────────┘  └──────────┘  └─────────┘
```

### 4.2 模块依赖关系

```text
cmd/api
  └── internal/api
        └── internal/bootstrap
              ├── pkg/infra/*
              ├── pkg/security/*
              └── pkg/component/*

cmd/user-center
  └── internal/user-center
        ├── internal/bootstrap
        └── pkg/*
```

---

## 五、配置体系

### 5.1 配置文件

| 文件 | 用途 | 端口 |
|------|------|------|
| `sentinel-api.yaml` | API Server 配置 | 8080(HTTP), 9090(gRPC) |
| `sentinel-api-dev.yaml` | 开发环境配置 | 8080 |
| `user-center.yaml` | 用户中心配置 | 8081(HTTP), 9091(gRPC) |

### 5.2 配置节点

- `server`：服务器模式、超时、地址
- `http`：HTTP 框架适配器、中间件
- `grpc`：gRPC 配置
- `log`：日志级别、格式
- `mysql`：数据库连接
- `redis`：缓存连接
- `jwt`：Token 配置
- `middleware`：各中间件开关和参数

---

## 六、开发规范

### 6.1 构建命令

```bash
make build              # 构建 API Server
make build-user-center  # 构建用户中心
make run-dev            # 开发模式运行
make test               # 运行测试
make test-coverage      # 覆盖率报告
make fmt                # 代码格式化
make lint               # 静态检查
make tidy               # 依赖整理
```

### 6.2 文件组织约定

- **cmd/**：只包含 main.go 和 README
- **internal/**：不导出（包私有）
- **pkg/**：可被外部导入（包公共）
- **staging/**：Monorepo 内部库
- **vendor/**：第三方依赖

### 6.3 命名规范

- 包名小写短词：`user`, `auth`, `repo`
- 文件名下划线分隔：`user_service.go`
- 结构体导出名驼峰：`UserService`

---

## 七、核心特性

### 7.1 双协议支持

- **HTTP**：Gin（默认）或 Echo 适配器
- **gRPC**：完整 protobuf 集成

### 7.2 完整的认证授权

- JWT Token 管理
- Casbin RBAC 权限控制
- MySQL/Redis 双后端

### 7.3 可观测性三支柱

1. **日志**：Zap/Slog 双引擎，结构化日志
2. **链路追踪**：OpenTelemetry + Jaeger/Zipkin
3. **指标**：Prometheus 集成

### 7.4 企业级特性

- 热重载配置
- 优雅关闭
- 性能分析（pprof）
- 速率限制
- 安全响应头

### 7.5 高性能优化

- Sonic 高性能 JSON
- 响应对象池
- 连接池管理
- 并发控制

---

## 八、扩展指南

### 8.1 新增 API 模块步骤

1. 在 `internal/model/` 添加数据模型
2. 在 `internal/<module>/store/` 添加存储接口和实现
3. 在 `internal/<module>/biz/` 添加业务逻辑
4. 在 `internal/<module>/handler/` 添加 HTTP 处理器
5. 在 `internal/<module>/router/` 注册路由
6. 编写单元测试
7. 更新文档

### 8.2 新增中间件步骤

1. 在 `pkg/infra/middleware/` 创建中间件文件
2. 实现 `gin.HandlerFunc` 或 `echo.MiddlewareFunc`
3. 在 `middleware.go` 中注册
4. 在配置文件中添加开关
5. 编写测试

---

## 九、文档索引

| 目录 | 文档 | 主题 |
|------|------|------|
| docs/design/ | architecture.md | 系统架构 |
| docs/design/ | api-server.md | API 设计 |
| docs/design/ | auth-authz.md | 认证授权设计 |
| docs/design/ | error-code-design.md | 错误码设计 |
| docs/development/ | guide.md | 开发指南 |
| docs/configuration/ | environment-variables.md | 环境变量 |
| docs/ | ENHANCED_LOGGING.md | 增强日志 |
| docs/ | RATE_LIMIT_SECURITY_GUIDE.md | 速率限制 |
