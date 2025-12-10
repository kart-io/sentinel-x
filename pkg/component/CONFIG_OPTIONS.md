# Component Options Interface

## 概述

`component.ConfigOptions` 是所有组件配置选项必须实现的统一接口。该接口定义了组件配置的标准生命周期方法，确保系统中所有组件具有一致的配置行为。

## 接口定义

```go
type ConfigOptions interface {
    Complete() error
    Validate() error
    AddFlags(fs *pflag.FlagSet, namePrefix string)
}
```

### 方法说明

#### Complete() error

填充未设置但需要有效数据的字段。此方法应该：

- 为可选字段设置默认值
- 从其他配置派生计算字段
- 完成配置初始化

**返回值：**

- `nil` - 成功完成配置
- `error` - 完成失败（例如，无法派生必需值）

**调用时机：** 在配置加载后、验证前调用

#### Validate() error

验证选项并在任何选项无效时返回错误。此方法应该检查：

- 必填字段是否已填充
- 字段值是否在可接受范围内
- 字段组合是否逻辑一致

**返回值：**

- `nil` - 所有验证通过
- `error` - 描述无效或缺失内容的错误

**调用时机：** 在 `Complete()` 之后调用，以确保所有字段都已正确设置

#### AddFlags(fs *pflag.FlagSet, namePrefix string)

将选项的标志添加到指定的 FlagSet。

**参数：**

- `fs` - 要添加标志的标志集
- `namePrefix` - 标志名称前缀，用于避免冲突（例如 "mysql." 生成 "--mysql.host", "--mysql.port" 等标志）

**行为：** 实现应使用有意义的标志名称并提供清晰的描述

## 实现组件

以下组件已实现 `ConfigOptions` 接口：

- `mysql.Options` - MySQL 数据库配置
- `redis.Options` - Redis 缓存配置
- `mongodb.Options` - MongoDB 数据库配置
- `postgres.Options` - PostgreSQL 数据库配置
- `etcd.Options` - Etcd 配置中心配置

## 使用示例

### 基本使用

```go
import (
    "github.com/kart-io/sentinel-x/pkg/component"
    "github.com/kart-io/sentinel-x/pkg/component/mysql"
    "github.com/spf13/pflag"
)

func main() {
    // 创建选项实例
    var opts component.ConfigOptions = mysql.NewOptions()

    // 1. 添加命令行标志
    fs := pflag.NewFlagSet("myapp", pflag.ExitOnError)
    opts.AddFlags(fs, "mysql.")
    fs.Parse(os.Args[1:])

    // 2. 完成配置（填充默认值）
    if err := opts.Complete(); err != nil {
        log.Fatalf("配置完成失败: %v", err)
    }

    // 3. 验证配置
    if err := opts.Validate(); err != nil {
        log.Fatalf("配置验证失败: %v", err)
    }

    // 配置已就绪，可以使用
}
```

### 多组件配置

```go
type AppOptions struct {
    MySQL    *mysql.Options
    Redis    *redis.Options
    MongoDB  *mongodb.Options
}

func (o *AppOptions) Complete() error {
    // 完成所有子配置
    if err := o.MySQL.Complete(); err != nil {
        return err
    }
    if err := o.Redis.Complete(); err != nil {
        return err
    }
    if err := o.MongoDB.Complete(); err != nil {
        return err
    }
    return nil
}

func (o *AppOptions) Validate() error {
    // 验证所有子配置
    if err := o.MySQL.Validate(); err != nil {
        return err
    }
    if err := o.Redis.Validate(); err != nil {
        return err
    }
    if err := o.MongoDB.Validate(); err != nil {
        return err
    }
    return nil
}

func (o *AppOptions) AddFlags(fs *pflag.FlagSet) {
    // 为所有子配置添加标志，使用不同的前缀
    o.MySQL.AddFlags(fs, "mysql.")
    o.Redis.AddFlags(fs, "redis.")
    o.MongoDB.AddFlags(fs, "mongodb.")
}
```

### 接口类型断言

```go
// 检查类型是否实现接口
func checkImplementation(opt interface{}) bool {
    _, ok := opt.(component.ConfigOptions)
    return ok
}

// 使用接口进行统一处理
func processConfig(opts []component.ConfigOptions) error {
    for _, opt := range opts {
        if err := opt.Complete(); err != nil {
            return err
        }
        if err := opt.Validate(); err != nil {
            return err
        }
    }
    return nil
}
```

## 实现新组件

如果需要为新组件实现 `ConfigOptions` 接口：

### 1. 定义 Options 结构

```go
package mycomponent

import "github.com/spf13/pflag"

type Options struct {
    Host     string
    Port     int
    Timeout  time.Duration
    // ... 其他字段
}
```

### 2. 实现 Complete() 方法

```go
// Complete 填充未设置的字段为默认值
func (o *Options) Complete() error {
    // 设置默认值
    if o.Port == 0 {
        o.Port = 8080
    }
    if o.Timeout == 0 {
        o.Timeout = 30 * time.Second
    }
    return nil
}
```

### 3. 实现 Validate() 方法

```go
// Validate 验证配置选项
func (o *Options) Validate() error {
    if o.Host == "" {
        return fmt.Errorf("host 不能为空")
    }
    if o.Port < 1 || o.Port > 65535 {
        return fmt.Errorf("port 必须在 1-65535 范围内")
    }
    if o.Timeout < 0 {
        return fmt.Errorf("timeout 不能为负数")
    }
    return nil
}
```

### 4. 实现 AddFlags() 方法

```go
// AddFlags 添加命令行标志
func (o *Options) AddFlags(fs *pflag.FlagSet, namePrefix string) {
    fs.StringVar(&o.Host, namePrefix+"host", o.Host,
        "组件主机地址")
    fs.IntVar(&o.Port, namePrefix+"port", o.Port,
        "组件端口")
    fs.DurationVar(&o.Timeout, namePrefix+"timeout", o.Timeout,
        "操作超时时间")
}
```

### 5. 提供构造函数

```go
// NewOptions 创建具有默认值的新 Options 实例
func NewOptions() *Options {
    return &Options{
        Host:    "127.0.0.1",
        Port:    8080,
        Timeout: 30 * time.Second,
    }
}
```

## 最佳实践

### 1. 配置生命周期顺序

始终按以下顺序调用方法：

```go
opts := component.NewOptions()
opts.AddFlags(fs, "prefix.")  // 1. 添加标志
fs.Parse(args)                // 2. 解析参数
opts.Complete()               // 3. 完成配置
opts.Validate()               // 4. 验证配置
// 5. 使用配置
```

### 2. 环境变量处理

在 `Validate()` 中从环境变量读取敏感信息：

```go
func (o *Options) Validate() error {
    // 从环境变量读取密码
    if o.Password == "" {
        o.Password = os.Getenv("MYSQL_PASSWORD")
    }

    // 警告不安全的使用方式
    if o.Password != "" && os.Getenv("MYSQL_PASSWORD") == "" {
        fmt.Fprintf(os.Stderr, "警告: 通过 CLI 传递密码不安全\n")
    }

    return nil
}
```

### 3. 默认值设置

在 `NewOptions()` 中设置所有默认值，在 `Complete()` 中处理条件默认值：

```go
// NewOptions 设置静态默认值
func NewOptions() *Options {
    return &Options{
        Host: "127.0.0.1",
        Port: 3306,
    }
}

// Complete 处理动态/条件默认值
func (o *Options) Complete() error {
    // 如果未设置超时，根据连接类型设置不同的默认值
    if o.Timeout == 0 {
        if o.UseSSL {
            o.Timeout = 10 * time.Second
        } else {
            o.Timeout = 5 * time.Second
        }
    }
    return nil
}
```

### 4. 验证顺序

按依赖关系顺序验证字段：

```go
func (o *Options) Validate() error {
    // 先验证基础字段
    if o.Host == "" {
        return fmt.Errorf("host 必填")
    }

    // 再验证依赖字段
    if o.PoolSize > 0 && o.MinIdleConns > o.PoolSize {
        return fmt.Errorf("MinIdleConns 不能超过 PoolSize")
    }

    return nil
}
```

### 5. 标志命名约定

使用一致的标志命名约定：

```go
func (o *Options) AddFlags(fs *pflag.FlagSet, namePrefix string) {
    // 使用 kebab-case
    fs.StringVar(&o.Host, namePrefix+"host", o.Host, "...")
    fs.IntVar(&o.MaxConnections, namePrefix+"max-connections", o.MaxConnections, "...")

    // 不要使用 camelCase 或 snake_case
    // 错误: namePrefix+"maxConnections"
    // 错误: namePrefix+"max_connections"
}
```

## 测试

每个实现都应包含测试以验证接口合规性：

```go
func TestOptionsImplementsInterface(t *testing.T) {
    var _ component.ConfigOptions = (*Options)(nil)
}

func TestOptionsLifecycle(t *testing.T) {
    opts := NewOptions()

    // 测试 Complete
    if err := opts.Complete(); err != nil {
        t.Fatalf("Complete() 失败: %v", err)
    }

    // 测试 Validate
    if err := opts.Validate(); err != nil {
        t.Fatalf("Validate() 失败: %v", err)
    }

    // 测试 AddFlags
    fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
    opts.AddFlags(fs, "test.")

    // 验证标志已添加
    if flag := fs.Lookup("test.host"); flag == nil {
        t.Error("未找到期望的标志 'test.host'")
    }
}
```

## 注意事项

### 禁止模式

1. **不要在 Complete() 中进行验证** - 验证逻辑应该在 `Validate()` 中
2. **不要在 Validate() 中修改配置** - 修改应该在 `Complete()` 中
3. **不要让 AddFlags() 有副作用** - 仅添加标志，不修改配置
4. **不要在构造函数中进行复杂初始化** - 保持 `NewOptions()` 简单

### 线程安全

- `Complete()` 和 `Validate()` 应该是幂等的
- 可以安全地多次调用这些方法
- 不应有全局状态或副作用

### 错误处理

- 使用描述性错误消息
- 返回可操作的错误
- 考虑使用 `fmt.Errorf()` 包装错误以提供上下文

## 参考实现

查看以下文件以获取参考实现：

- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/pkg/component/mysql/options.go`
- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/pkg/component/redis/options.go`
- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/pkg/component/mongodb/options.go`
- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/pkg/component/postgres/options.go`
- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/pkg/component/etcd/options.go`

测试参考：

- `/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/pkg/component/interface_test.go`
