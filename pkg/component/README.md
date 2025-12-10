# Component Package

## 概述

`pkg/component` 包提供了统一的组件配置接口和各种基础设施组件的实现，包括数据库、缓存、配置中心等。

## 核心接口

### ConfigOptions

`ConfigOptions` 是所有组件配置必须实现的统一接口，定义了组件配置的标准生命周期方法。

```go
type ConfigOptions interface {
    Complete() error
    Validate() error
    AddFlags(fs *pflag.FlagSet, namePrefix string)
}
```

详细文档请参见：[CONFIG_OPTIONS.md](./CONFIG_OPTIONS.md)

## 支持的组件

### 数据库组件

- **MySQL** - `pkg/component/mysql`
  - 支持连接池配置
  - 支持连接生命周期管理
  - 密码通过环境变量安全传递

- **PostgreSQL** - `pkg/component/postgres`
  - 支持 SSL 模式配置
  - 支持连接池管理
  - 密码通过环境变量安全传递

- **MongoDB** - `pkg/component/mongodb`
  - 支持副本集配置
  - 支持连接池管理
  - 支持多种超时配置

### 缓存组件

- **Redis** - `pkg/component/redis`
  - 支持连接池配置
  - 支持多种超时配置
  - 支持重试机制

### 配置中心

- **Etcd** - `pkg/component/etcd`
  - 支持多端点配置
  - 支持租约配置
  - 支持认证

### 存储抽象

- **Storage** - `pkg/component/storage`
  - 提供统一的存储接口
  - 提供通用配置选项
  - 支持多种存储后端

## 使用示例

### 基本使用

```go
import (
    "github.com/kart-io/sentinel-x/pkg/component"
    "github.com/kart-io/sentinel-x/pkg/component/mysql"
)

// 创建配置
opts := mysql.NewOptions()

// 添加命令行标志
fs := pflag.NewFlagSet("app", pflag.ExitOnError)
opts.AddFlags(fs, "mysql.")
fs.Parse(os.Args[1:])

// 完成配置（设置默认值）
if err := opts.Complete(); err != nil {
    log.Fatal(err)
}

// 验证配置
if err := opts.Validate(); err != nil {
    log.Fatal(err)
}

// 使用配置创建客户端
client, err := mysql.NewClient(opts)
```

### 多组件配置

```go
type AppOptions struct {
    MySQL    *mysql.Options
    Redis    *redis.Options
}

func (o *AppOptions) Complete() error {
    if err := o.MySQL.Complete(); err != nil {
        return err
    }
    return o.Redis.Complete()
}

func (o *AppOptions) Validate() error {
    if err := o.MySQL.Validate(); err != nil {
        return err
    }
    return o.Redis.Validate()
}

func (o *AppOptions) AddFlags(fs *pflag.FlagSet) {
    o.MySQL.AddFlags(fs, "mysql.")
    o.Redis.AddFlags(fs, "redis.")
}
```

### 完整示例

查看 [example/main.go](./example/main.go) 了解如何使用统一接口管理多个组件配置的完整示例。

运行示例：

```bash
cd pkg/component/example
go run main.go
```

带参数运行：

```bash
go run main.go --mysql.host=localhost --mysql.port=3306 --redis.port=6379
```

## 配置生命周期

所有组件配置都遵循以下生命周期：

```
1. 创建实例 (NewOptions)
   ↓
2. 添加标志 (AddFlags)
   ↓
3. 解析参数 (pflag.Parse)
   ↓
4. 完成配置 (Complete)
   ↓
5. 验证配置 (Validate)
   ↓
6. 使用配置
```

## 环境变量

所有组件都支持通过环境变量传递敏感信息（如密码）：

- MySQL: `MYSQL_PASSWORD`
- Redis: `REDIS_PASSWORD`
- MongoDB: `MONGODB_PASSWORD`
- PostgreSQL: `POSTGRES_PASSWORD`
- Etcd: `ETCD_PASSWORD`

示例：

```bash
export MYSQL_PASSWORD="your-secure-password"
./myapp --mysql.host=localhost --mysql.port=3306
```

## 测试

运行所有组件测试：

```bash
go test ./pkg/component/...
```

运行接口合规性测试：

```bash
go test ./pkg/component -run TestConfigOptions
```

## 项目结构

```
pkg/component/
├── interface.go              # ConfigOptions 接口定义
├── interface_test.go         # 接口合规性测试
├── CONFIG_OPTIONS.md         # 详细文档
├── README.md                 # 本文件
├── example/
│   └── main.go              # 使用示例
├── mysql/                    # MySQL 组件
│   ├── options.go
│   ├── client.go
│   └── ...
├── redis/                    # Redis 组件
│   ├── options.go
│   ├── client.go
│   └── ...
├── mongodb/                  # MongoDB 组件
│   ├── options.go
│   ├── client.go
│   └── ...
├── postgres/                 # PostgreSQL 组件
│   ├── options.go
│   ├── client.go
│   └── ...
├── etcd/                     # Etcd 组件
│   ├── options.go
│   ├── client.go
│   └── ...
└── storage/                  # 存储抽象
    ├── storage.go
    ├── options.go
    └── ...
```

## 最佳实践

### 1. 使用环境变量传递敏感信息

```bash
# 推荐
export MYSQL_PASSWORD="secret"
./app --mysql.host=localhost

# 不推荐（不安全）
./app --mysql.host=localhost --mysql.password=secret
```

### 2. 始终按顺序调用生命周期方法

```go
opts.AddFlags(fs, "prefix.")
fs.Parse(args)
opts.Complete()   // 必须在 Validate 之前
opts.Validate()   // 必须在 Complete 之后
```

### 3. 使用有意义的标志前缀

```go
// 推荐
mysqlOpts.AddFlags(fs, "mysql.")
redisOpts.AddFlags(fs, "redis.")

// 不推荐（可能冲突）
mysqlOpts.AddFlags(fs, "")
redisOpts.AddFlags(fs, "")
```

### 4. 提供合理的默认值

```go
func NewOptions() *Options {
    return &Options{
        Host: "127.0.0.1",
        Port: 3306,
        MaxConnections: 100,
        // 其他合理的默认值
    }
}
```

## 扩展新组件

要添加新组件，请：

1. 在 `pkg/component/<component-name>/` 创建新包
2. 定义 `Options` 结构体
3. 实现 `ConfigOptions` 接口的三个方法
4. 提供 `NewOptions()` 构造函数
5. 添加测试以验证接口实现
6. 更新本 README 文档

详细指南请参见：[CONFIG_OPTIONS.md](./CONFIG_OPTIONS.md#实现新组件)

## 参考文档

- [ConfigOptions 接口详细文档](./CONFIG_OPTIONS.md)
- [MySQL 组件文档](./mysql/README.md)
- [Redis 组件文档](./redis/doc.go)
- [MongoDB 组件文档](./mongodb/doc.go)
- [PostgreSQL 组件文档](./postgres/README.md)
- [Etcd 组件文档](./etcd/README.md)
- [Storage 抽象文档](./storage/README.md)

## 许可证

版权所有 (c) kart-io
