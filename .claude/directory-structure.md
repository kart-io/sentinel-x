# Sentinel-X 目录结构详解

> 生成时间：2025-12-11
> 本文档详细说明项目各目录和文件的用途

---

## 一、根目录结构

```text
sentinel-x/
├── bin/                    # 编译输出
├── cmd/                    # 应用入口
├── configs/                # 配置文件
├── docs/                   # 项目文档
├── example/                # 代码示例
├── examples/               # 高级示例
├── hack/                   # 维护脚本
├── internal/               # 私有代码
├── pkg/                    # 公共库
├── scripts/                # 工具脚本
├── staging/                # 内部依赖
├── vendor/                 # 第三方依赖
├── .claude/                # Claude 工作目录
├── .env.example            # 环境变量示例
├── .gitignore              # Git 忽略规则
├── CLAUDE.md               # 开发指南
├── LICENSE                 # 许可证
├── Makefile                # 构建脚本
├── README.md               # 项目说明
├── coverage.out            # 测试覆盖率
├── go.mod                  # Go 模块定义
└── go.sum                  # 依赖哈希
```

---

## 二、cmd/ - 应用入口

```text
cmd/
├── api/                    # API Server 入口
│   ├── main.go             # 主程序入口
│   └── README.md           # 模块说明
├── user-center/            # 用户中心服务入口
│   └── main.go             # 主程序入口
└── scheduler/              # 调度器服务入口（占位）
    └── main.go             # 主程序入口
```

### 使用规则

- 每个子目录对应一个可执行程序
- `main.go` 保持轻量，仅包含初始化流程
- 业务逻辑放入 `internal/` 或 `pkg/`

---

## 三、internal/ - 私有应用代码

```text
internal/
├── api/                    # API Server 应用层
│   ├── app.go              # 应用实例定义
│   └── options.go          # 配置选项
├── bootstrap/              # 启动引导逻辑
│   ├── auth.go             # 认证初始化
│   ├── bootstrapper.go     # 启动协调器
│   ├── datasource.go       # 数据源初始化
│   ├── initializer.go      # 初始化器接口
│   ├── logging.go          # 日志初始化
│   ├── middleware.go       # 中间件初始化
│   ├── run.go              # 运行器
│   └── server.go           # 服务器初始化
├── model/                  # 数据模型
│   ├── auth.go             # 认证相关模型
│   └── user.go             # 用户相关模型
└── user-center/            # 用户中心服务
    ├── app.go              # 应用入口
    ├── options.go          # 配置选项
    ├── biz/                # 业务逻辑层
    │   ├── auth.go         # 认证业务逻辑
    │   ├── user.go         # 用户业务逻辑
    │   └── doc.go          # 包文档
    ├── handler/            # HTTP 处理层
    │   ├── auth.go         # 认证处理
    │   ├── user.go         # 用户处理
    │   └── doc.go          # 包文档
    ├── router/             # 路由配置
    │   └── router.go       # 路由注册
    ├── store/              # 数据存储层
    │   ├── mysql.go        # MySQL 实现
    │   ├── store.go        # 存储接口
    │   ├── user.go         # 用户存储
    │   └── doc.go          # 包文档
    └── pkg/                # 包内工具
        └── doc.go          # 包文档
```

### 使用规则

- `internal/` 下的包不可被外部项目引用
- 按功能模块划分子目录
- 每个模块遵循分层架构：handler → biz → store

---

## 四、pkg/ - 公共库

### 4.1 整体结构

```text
pkg/
├── component/              # 基础组件
├── infra/                  # 基础设施
├── security/               # 安全组件
└── utils/                  # 工具库
```

### 4.2 component/ - 基础组件

```text
pkg/component/
├── etcd/                   # ETCD 客户端
│   └── ...
├── mongodb/                # MongoDB 驱动
│   └── ...
├── mysql/                  # MySQL 驱动
│   ├── client.go           # 连接管理
│   ├── dsn.go              # DSN 生成
│   ├── factory.go          # 工厂方法
│   ├── health.go           # 健康检查
│   ├── logger.go           # 日志集成
│   ├── options.go          # 配置选项
│   └── ...
├── postgres/               # PostgreSQL 驱动
│   └── ...
├── redis/                  # Redis 驱动
│   ├── client.go           # 客户端
│   ├── options.go          # 配置选项
│   └── ...
├── storage/                # 存储管理器
│   └── ...
├── interface.go            # 组件接口
├── interface_test.go       # 接口测试
├── CONFIG_OPTIONS.md       # 配置说明
└── README.md               # 模块说明
```

### 4.3 infra/ - 基础设施

```text
pkg/infra/
├── adapter/                # Web 框架适配器
│   ├── echo/               # Echo 框架适配
│   │   └── ...
│   └── gin/                # Gin 框架适配
│       └── ...
├── app/                    # 应用管理
│   ├── app.go              # 应用生命周期
│   ├── logger.go           # 日志初始化
│   ├── options.go          # 配置选项
│   └── version.go          # 版本信息
├── config/                 # 配置管理
│   └── example/            # 配置示例
├── datasource/             # 数据源管理
│   ├── clients.go          # 客户端池
│   ├── generic.go          # 通用实现
│   ├── generic_test.go     # 测试
│   ├── manager.go          # 管理器
│   └── manager_test.go     # 测试
├── logger/                 # 日志管理
│   ├── context.go          # 上下文传递
│   ├── context_test.go     # 测试
│   ├── fields.go           # 标准化字段
│   ├── options.go          # 配置选项
│   └── reloadable.go       # 热重载
├── middleware/             # HTTP/gRPC 中间件
│   ├── auth.go             # JWT 认证
│   ├── authz.go            # 授权
│   ├── cors.go             # 跨域
│   ├── health.go           # 健康检查
│   ├── logger.go           # 日志
│   ├── logger_enhanced.go  # 增强日志
│   ├── metrics.go          # 指标
│   ├── middleware.go       # 中间件管理
│   ├── pprof.go            # 性能分析
│   ├── ratelimit.go        # 速率限制
│   ├── recovery.go         # 异常恢复
│   ├── reloadable.go       # 热重载
│   ├── request_id.go       # 请求 ID
│   ├── security_headers.go # 安全头
│   ├── timeout.go          # 超时控制
│   ├── tracing.go          # 链路追踪
│   ├── grpc/               # gRPC 中间件
│   │   └── ...
│   └── *_test.go           # 测试文件
├── server/                 # 服务器管理
│   ├── grpc/               # gRPC 服务器
│   ├── http/               # HTTP 服务器
│   ├── service/            # 服务注册
│   └── transport/          # 传输适配
└── tracing/                # 链路追踪
    ├── context.go          # 上下文
    ├── options.go          # 配置
    ├── provider.go         # 提供者
    └── provider_test.go    # 测试
```

### 4.4 security/ - 安全组件

```text
pkg/security/
├── auth/                   # 认证模块
│   ├── jwt/                # JWT 实现
│   │   ├── claims.go       # Claims 定义
│   │   ├── options.go      # 配置选项
│   │   ├── token.go        # Token 管理
│   │   └── ...
│   └── middleware/         # 认证中间件
│       └── ...
└── authz/                  # 授权模块
    ├── casbin/             # Casbin RBAC
    │   ├── enforcer.go     # 执行器
    │   ├── infrastructure/ # 后端存储
    │   │   ├── mysql/      # MySQL 后端
    │   │   └── redis/      # Redis 缓存
    │   └── ...
    └── rbac/               # RBAC 接口
        └── ...
```

### 4.5 utils/ - 工具库

```text
pkg/utils/
├── errors/                 # 错误处理
│   ├── base.go             # 基础错误
│   ├── builder.go          # 错误构建器
│   ├── code.go             # 错误码常量
│   ├── errno.go            # 错误码定义
│   ├── helpers.go          # 辅助函数
│   ├── registry.go         # 注册表
│   └── *_test.go           # 测试
├── id/                     # ID 生成
│   ├── id.go               # 接口定义
│   ├── snowflake.go        # 雪花算法
│   ├── ulid.go             # ULID
│   ├── uuid.go             # UUID
│   └── *_test.go           # 测试
├── json/                   # JSON 处理
│   ├── json.go             # Sonic 封装
│   ├── json_test.go        # 测试
│   └── benchmark_test.go   # 性能测试
├── response/               # 响应管理
│   ├── response.go         # 响应结构
│   ├── writer.go           # 响应写入
│   └── pool_test.go        # 对象池测试
└── validator/              # 参数验证
    ├── errors.go           # 验证错误
    ├── rules.go            # 自定义规则
    ├── translations.go     # 多语言
    ├── validator.go        # 验证器
    └── *_test.go           # 测试
```

---

## 五、configs/ - 配置文件

```text
configs/
├── sentinel-api.yaml       # API Server 生产配置
├── sentinel-api-dev.yaml   # API Server 开发配置
└── user-center.yaml        # 用户中心配置
```

### 配置文件结构

```yaml
server:
  mode: release             # 运行模式
  name: sentinel-api        # 服务名称

http:
  addr: :8080               # HTTP 地址
  adapter: gin              # 框架适配器

grpc:
  addr: :9090               # gRPC 地址

log:
  level: info               # 日志级别
  format: json              # 输出格式

mysql:
  host: localhost           # 数据库地址
  port: 3306                # 端口
  database: sentinel        # 数据库名

redis:
  addr: localhost:6379      # Redis 地址

jwt:
  secret: your-secret       # JWT 密钥
  expire: 24h               # 过期时间

middleware:
  recovery: true            # 开启恢复
  logger: true              # 开启日志
  cors: true                # 开启跨域
```

---

## 六、docs/ - 项目文档

```text
docs/
├── api/                    # API 文档
│   └── README.md
├── configuration/          # 配置文档
│   └── environment-variables.md
├── design/                 # 设计文档
│   ├── architecture.md     # 系统架构
│   ├── api-server.md       # API 设计
│   ├── auth-authz.md       # 认证授权
│   ├── error-code-design.md
│   ├── error-code-migration.md
│   ├── MIGRATION.md
│   └── scheduler.md
├── development/            # 开发指南
│   └── guide.md
├── usage/                  # 使用指南
│   └── README.md
├── ENHANCED_LOGGING.md
├── ENHANCED_LOGGING_QUICKREF.md
├── ENHANCED_LOGGING_SUMMARY.md
├── RATE_LIMIT_SECURITY_GUIDE.md
├── response-pooling.md
├── SECURITY_FIX_REPORT.md
└── SONIC_INTEGRATION.md
```

---

## 七、staging/ - 内部依赖库

```text
staging/
└── src/github.com/kart-io/
    └── logger/             # Logger 库源码
        ├── engines/        # 日志引擎
        │   ├── slog/       # Slog 引擎
        │   └── zap/        # Zap 引擎
        ├── example/        # 使用示例
        ├── go.mod
        └── ...
```

### 使用说明

- 通过 `go.mod` 的 `replace` 指令引用
- 统一版本管理
- 便于开发调试

---

## 八、scripts/ 和 hack/

```text
scripts/
├── data/                   # 测试数据
└── verify_sonic_integration.sh

hack/
├── staging-replaces.txt    # replace 配置
├── sync-from-upstream.sh   # 从上游同步
├── sync-to-upstream.sh     # 同步到上游
└── update-vendor.sh        # 更新 vendor
```

---

## 九、文件命名规范

### Go 文件

| 类型 | 命名规则 | 示例 |
|------|----------|------|
| 接口定义 | `interface.go` 或 `<name>.go` | `store.go` |
| 实现文件 | `<name>_<impl>.go` | `user_mysql.go` |
| 测试文件 | `<name>_test.go` | `user_test.go` |
| 性能测试 | `benchmark_test.go` | `benchmark_test.go` |
| 包文档 | `doc.go` | `doc.go` |
| 选项模式 | `options.go` | `options.go` |

### 配置文件

| 类型 | 命名规则 | 示例 |
|------|----------|------|
| 生产配置 | `<service>.yaml` | `sentinel-api.yaml` |
| 开发配置 | `<service>-dev.yaml` | `sentinel-api-dev.yaml` |
| 示例配置 | `<name>.example.yaml` | `config.example.yaml` |

---

## 十、目录使用决策树

```text
需要新增代码？
├─ 可执行程序入口 → cmd/<app>/main.go
├─ 业务逻辑代码 → internal/<module>/
│   ├─ HTTP 处理 → handler/
│   ├─ 业务逻辑 → biz/
│   ├─ 数据访问 → store/
│   └─ 路由注册 → router/
├─ 可复用工具 → pkg/utils/
├─ 数据库驱动 → pkg/component/
├─ 中间件 → pkg/infra/middleware/
├─ 安全组件 → pkg/security/
└─ 内部依赖 → staging/
```
