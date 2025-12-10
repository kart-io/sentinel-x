# MySQL Storage Client

MySQL 存储客户端实现，为 sentinel-x 项目提供统一的 MySQL 数据库访问接口。

## 功能特性

- **统一接口**: 实现 `storage.Client` 接口，提供标准化的存储访问方式
- **连接池管理**: 可配置的连接池参数（最大空闲连接、最大打开连接、连接生命周期）
- **健康检查**: 内置健康检查功能，支持延迟测量和连接池状态监控
- **上下文支持**: 所有关键操作都支持 context，便于超时控制和取消操作
- **GORM 集成**: 基于 GORM 构建，提供完整的 ORM 功能
- **日志级别**: 支持配置 GORM 日志级别（Silent、Error、Warn、Info）
- **优雅关闭**: 支持优雅关闭，确保所有资源正确释放
- **工厂模式**: 提供工厂模式支持，便于依赖注入和测试

## 安装

```bash
go get github.com/kart-io/sentinel-x/pkg/mysql
```

## 快速开始

### 基本使用

```go
package main

import (
    "log"

    "github.com/kart-io/sentinel-x/pkg/mysql"
    mysqlOpts "github.com/kart-io/sentinel-x/pkg/options/mysql"
)

func main() {
    // 创建配置选项
    opts := mysqlOpts.NewOptions()
    opts.Host = "localhost"
    opts.Port = 3306
    opts.Username = "root"
    opts.Password = "password"
    opts.Database = "myapp"

    // 创建 MySQL 客户端
    client, err := mysql.New(opts)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 使用客户端
    log.Printf("Connected to %s", client.Name())
}
```

### 使用 Context 超时控制

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

client, err := mysql.NewWithContext(ctx, opts)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 健康检查

```go
// 基本健康检查
if err := client.Ping(context.Background()); err != nil {
    log.Printf("MySQL unhealthy: %v", err)
}

// 详细健康检查
status := mysql.CheckHealth(client, 5*time.Second)
if status.Healthy {
    log.Printf("MySQL healthy (latency: %v)", status.Latency)
} else {
    log.Printf("MySQL unhealthy: %v", status.Error)
}

// 健康检查 + 统计信息
status, stats := mysql.HealthWithStats(client, 5*time.Second)
log.Printf("Pool stats: %+v", stats)
```

### 使用 GORM

```go
type User struct {
    ID   uint   `gorm:"primaryKey"`
    Name string `gorm:"size:100"`
    Age  int
}

// 获取 GORM DB
db := client.DB()

// 自动迁移
db.AutoMigrate(&User{})

// 创建记录
db.Create(&User{Name: "Alice", Age: 30})

// 查询记录
var users []User
db.Where("age > ?", 25).Find(&users)
```

### 工厂模式

```go
// 创建工厂
factory := mysql.NewFactory(opts)

// 使用工厂创建客户端
client, err := factory.Create(context.Background())
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 克隆工厂并自定义配置
devFactory := factory.Clone()
devFactory.Options().Database = "dev_db"
```

## 配置选项

### Options 结构

```go
type Options struct {
    Host                  string        // MySQL 主机地址
    Port                  int           // MySQL 端口
    Username              string        // 用户名
    Password              string        // 密码（推荐使用环境变量 MYSQL_PASSWORD）
    Database              string        // 数据库名称
    MaxIdleConnections    int           // 最大空闲连接数
    MaxOpenConnections    int           // 最大打开连接数
    MaxConnectionLifeTime time.Duration // 连接最大生命周期
    LogLevel              int           // 日志级别 (1=Silent, 2=Error, 3=Warn, 4=Info)
}
```

### 默认值

- Host: `127.0.0.1`
- Port: `3306`
- Username: `root`
- Password: `""` (从环境变量 `MYSQL_PASSWORD` 读取)
- MaxIdleConnections: `10`
- MaxOpenConnections: `100`
- MaxConnectionLifeTime: `10s`
- LogLevel: `1` (Silent)

### 环境变量

推荐使用环境变量传递敏感信息：

```bash
export MYSQL_PASSWORD="your-secure-password"
```

## 连接池配置

### 配置最佳实践

```go
opts := mysqlOpts.NewOptions()

// 连接池配置
opts.MaxIdleConnections = 10     // 保持 10 个空闲连接
opts.MaxOpenConnections = 100    // 最多 100 个打开连接
opts.MaxConnectionLifeTime = 10 * time.Second // 连接最多存活 10 秒

client, err := mysql.New(opts)
```

### 查看连接池状态

```go
sqlDB, err := client.SqlDB()
if err != nil {
    log.Fatal(err)
}

stats := sqlDB.Stats()
log.Printf("Max open connections: %d", stats.MaxOpenConnections)
log.Printf("Open connections: %d", stats.OpenConnections)
log.Printf("In use: %d", stats.InUse)
log.Printf("Idle: %d", stats.Idle)
```

## 健康检查集成

### 与健康检查中间件集成

```go
import (
    "github.com/kart-io/sentinel-x/pkg/middleware"
    "github.com/kart-io/sentinel-x/pkg/mysql"
)

func main() {
    // 创建 MySQL 客户端
    client, err := mysql.New(opts)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 注册到健康检查管理器
    healthMgr := middleware.GetHealthManager()
    healthMgr.RegisterChecker("mysql", client.Health())

    // 现在健康检查端点会包含 MySQL 状态
}
```

## API 文档

### Client

```go
type Client struct {
    // 私有字段
}

// 创建新的 MySQL 客户端
func New(opts *Options) (*Client, error)

// 使用 context 创建客户端
func NewWithContext(ctx context.Context, opts *Options) (*Client, error)

// 返回存储类型名称
func (c *Client) Name() string

// 检查连接是否存活
func (c *Client) Ping(ctx context.Context) error

// 关闭连接
func (c *Client) Close() error

// 返回健康检查函数
func (c *Client) Health() storage.HealthChecker

// 返回 GORM DB 实例
func (c *Client) DB() *gorm.DB

// 返回底层 sql.DB 实例
func (c *Client) SqlDB() (*sql.DB, error)
```

### Factory

```go
type Factory struct {
    // 私有字段
}

// 创建新的工厂
func NewFactory(opts *Options) *Factory

// 创建客户端
func (f *Factory) Create(ctx context.Context) (storage.Client, error)

// 返回工厂的配置选项
func (f *Factory) Options() *Options

// 克隆工厂
func (f *Factory) Clone() *Factory
```

### 健康检查函数

```go
// 执行健康检查
func CheckHealth(client *Client, timeout time.Duration) storage.HealthStatus

// 执行健康检查并返回详细统计信息
func HealthWithStats(client *Client, timeout time.Duration) (storage.HealthStatus, map[string]interface{})
```

### DSN 构建

```go
// 从选项构建 DSN
func BuildDSN(opts *Options) string
```

## 错误处理

### 常见错误

```go
client, err := mysql.New(opts)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "invalid mysql options"):
        // 配置错误
        log.Printf("Configuration error: %v", err)
    case strings.Contains(err.Error(), "failed to connect"):
        // 连接错误
        log.Printf("Connection error: %v", err)
    case strings.Contains(err.Error(), "failed to ping"):
        // Ping 失败
        log.Printf("Ping error: %v", err)
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

### 验证错误

创建客户端前会验证以下配置：

- Host 不能为空
- Database 不能为空
- Port 必须在 1-65535 之间
- Username 不能为空

## 线程安全

MySQL 客户端是线程安全的，可以在多个 goroutine 中并发使用。底层的 GORM 和 `database/sql` 包会自动处理连接池和线程安全。

```go
// 在多个 goroutine 中安全使用
for i := 0; i < 10; i++ {
    go func() {
        db := client.DB()
        var count int64
        db.Model(&User{}).Count(&count)
    }()
}
```

## 最佳实践

### 1. 使用环境变量管理敏感信息

```bash
# 不要在代码中硬编码密码
export MYSQL_PASSWORD="your-password"
export MYSQL_USERNAME="your-username"
```

### 2. 合理配置连接池

```go
// 根据应用负载调整连接池大小
opts.MaxIdleConnections = 10     // 低流量应用
opts.MaxOpenConnections = 50     // 中等流量应用

// 高流量应用
opts.MaxIdleConnections = 50
opts.MaxOpenConnections = 200
```

### 3. 使用 Context 控制超时

```go
// 为长时间操作设置超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

var users []User
db := client.DB().WithContext(ctx)
db.Where("active = ?", true).Find(&users)
```

### 4. 定期健康检查

```go
// 定期执行健康检查
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for range ticker.C {
    status := mysql.CheckHealth(client, 5*time.Second)
    if !status.Healthy {
        log.Printf("MySQL health check failed: %v", status.Error)
        // 触发告警或重连逻辑
    }
}
```

### 5. 优雅关闭

```go
// 使用 defer 确保资源释放
client, err := mysql.New(opts)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 或者在信号处理中优雅关闭
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

<-sigChan
log.Println("Shutting down...")
client.Close()
```

## 示例项目

查看 `example_test.go` 文件获取更多使用示例。

## 依赖

- `gorm.io/gorm` - ORM 框架
- `gorm.io/driver/mysql` - MySQL 驱动
- `github.com/kart-io/sentinel-x/pkg/options/mysql` - 配置选项
- `github.com/kart-io/sentinel-x/pkg/storage` - 存储接口

## 许可证

遵循 sentinel-x 项目许可证。
