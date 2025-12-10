# PostgreSQL Client Package

PostgreSQL 存储客户端，基于 GORM 实现，提供完整的数据库连接管理和健康检查功能。

## 功能特性

- 基于 GORM v2 和 PostgreSQL 驱动
- 连接池配置管理
- 健康检查支持
- 客户端工厂模式
- Context 支持
- SSL 模式配置
- 日志级别控制

## 目录结构

```
pkg/postgres/
├── client.go   - PostgreSQL 客户端实现
├── dsn.go      - DSN 构建工具
├── factory.go  - 客户端工厂
└── health.go   - 健康检查实现
```

## 基本用法

### 创建客户端

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/kart-io/sentinel-x/pkg/options/postgres"
    pgclient "github.com/kart-io/sentinel-x/pkg/postgres"
)

func main() {
    // 创建配置
    opts := &postgres.Options{
        Host:                  "localhost",
        Port:                  5432,
        Username:              "postgres",
        Password:              "password",
        Database:              "mydb",
        SSLMode:               "disable",
        MaxIdleConnections:    10,
        MaxOpenConnections:    100,
        MaxConnectionLifeTime: 10 * time.Second,
        LogLevel:              1, // Silent
    }

    // 创建客户端
    client, err := pgclient.New(opts)
    if err != nil {
        log.Fatalf("Failed to create postgres client: %v", err)
    }
    defer client.Close()

    // 使用 GORM
    db := client.DB()
    // 现在可以使用 db 进行数据库操作
}
```

### 使用 Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

client, err := pgclient.NewWithContext(ctx, opts)
if err != nil {
    log.Fatalf("Failed to create postgres client: %v", err)
}
defer client.Close()
```

### 健康检查

```go
// 执行健康检查
err := client.HealthCheck(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
}

// 获取连接统计
stats, err := client.Stats()
if err != nil {
    log.Printf("Failed to get stats: %v", err)
}
fmt.Printf("Open connections: %d\n", stats.OpenConnections)
```

### 使用工厂模式

```go
// 创建配置
opts := &postgres.Options{
    Host:     "localhost",
    Port:     5432,
    Username: "postgres",
    Password: "password",
    Database: "mydb",
    SSLMode:  "disable",
}

// 创建工厂
factory := pgclient.NewFactory(opts)

// 使用工厂创建客户端
client, err := factory.Create(context.Background())
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Close()

// 克隆工厂以创建不同配置的客户端
devFactory := factory.Clone()
devFactory.Options().Database = "dev_db"
devClient, err := devFactory.Create(context.Background())
if err != nil {
    log.Fatalf("Failed to create dev client: %v", err)
}
defer devClient.Close()
```

## DSN 构建

```go
import "github.com/kart-io/sentinel-x/pkg/postgres"

opts := &postgres.Options{
    Host:     "localhost",
    Port:     5432,
    Username: "postgres",
    Password: "password",
    Database: "mydb",
    SSLMode:  "disable",
}

dsn := postgres.BuildDSN(opts)
// 输出: host=localhost port=5432 user=postgres password=password dbname=mydb sslmode=disable
```

## 配置说明

### Options 结构

- **Host**: 数据库主机地址 (默认: 127.0.0.1)
- **Port**: 数据库端口 (默认: 5432)
- **Username**: 数据库用户名 (默认: postgres)
- **Password**: 数据库密码 (建议使用环境变量 `POSTGRES_PASSWORD`)
- **Database**: 数据库名称 (必填)
- **SSLMode**: SSL 模式 (默认: disable)
  - `disable`: 不使用 SSL
  - `require`: 要求 SSL
  - `verify-ca`: 验证 CA
  - `verify-full`: 完全验证
- **MaxIdleConnections**: 最大空闲连接数 (默认: 10)
- **MaxOpenConnections**: 最大打开连接数 (默认: 100)
- **MaxConnectionLifeTime**: 连接最大生命周期 (默认: 10s)
- **LogLevel**: 日志级别 (默认: 1)
  - `1`: Silent (静默)
  - `2`: Error (错误)
  - `3`: Warn (警告)
  - `4`: Info (信息)

## 环境变量

建议使用环境变量传递敏感信息:

```bash
export POSTGRES_PASSWORD="your-secure-password"
```

## 接口实现

### storage.Client 接口

```go
type Client interface {
    Name() string
    Ping(ctx context.Context) error
    Close() error
    Health() service.HealthChecker
}
```

### service.HealthChecker 接口

```go
type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}
```

## 高级用法

### 直接访问 GORM

```go
client, _ := pgclient.New(opts)

// 获取 GORM DB
db := client.DB()

// 使用 GORM 功能
type User struct {
    ID   uint
    Name string
}

db.AutoMigrate(&User{})
db.Create(&User{Name: "John"})

var user User
db.First(&user, 1)
```

### 访问原生 sql.DB

```go
client, _ := pgclient.New(opts)

// 获取 sql.DB
sqlDB, err := client.SqlDB()
if err != nil {
    log.Fatal(err)
}

// 使用标准库功能
rows, err := sqlDB.Query("SELECT * FROM users")
```

### 连接池配置

```go
opts := &postgres.Options{
    Host:                  "localhost",
    Port:                  5432,
    Database:              "mydb",
    MaxIdleConnections:    20,  // 增加空闲连接
    MaxOpenConnections:    200, // 增加最大连接数
    MaxConnectionLifeTime: 30 * time.Second, // 延长连接生命周期
}
```

## 错误处理

所有错误都使用 `fmt.Errorf` 进行包装，支持错误链:

```go
client, err := pgclient.New(opts)
if err != nil {
    // 错误已包含上下文信息
    log.Printf("Failed to create client: %v", err)
    return
}
```

## 最佳实践

1. **使用环境变量**: 通过环境变量传递密码，避免硬编码
2. **优雅关闭**: 使用 `defer client.Close()` 确保连接正确关闭
3. **健康检查**: 定期执行健康检查，确保连接可用
4. **Context 超时**: 为数据库操作设置合理的超时时间
5. **连接池调优**: 根据应用负载调整连接池参数
6. **日志级别**: 生产环境使用 Silent 或 Error 级别

## 示例项目

完整示例请参考 `example/` 目录中的代码。
