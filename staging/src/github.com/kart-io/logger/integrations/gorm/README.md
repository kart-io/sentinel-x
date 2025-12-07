# GORM Integration

GORM 数据库 ORM 框架的统一日志器适配，提供完整的 SQL 查询记录、慢查询检测、错误处理和性能监控。

## 📋 特性

- ✅ **完整 GORM 接口**: 实现 GORM 官方 `logger.Interface`
- ✅ **SQL 查询记录**: 详细记录所有 SQL 操作和参数
- ✅ **慢查询检测**: 可配置阈值的慢查询监控和报警
- ✅ **智能错误处理**: 可配置的 RecordNotFound 错误过滤
- ✅ **性能监控**: 查询耗时、影响行数等性能指标
- ✅ **日志级别控制**: 支持 GORM 的四级日志控制
- ✅ **上下文感知**: 支持 context 传递和追踪
- ✅ **零依赖**: 无需引入 GORM 库即可使用

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "github.com/kart-io/logger/factory"
    "github.com/kart-io/logger/integrations/gorm"
    "github.com/kart-io/logger/option"
    
    // GORM 相关导入（实际项目中）
    // "gorm.io/gorm"
    // "gorm.io/driver/mysql"
)

func main() {
    // 创建统一日志器
    opt := option.DefaultLogOption()
    logger, err := factory.NewLogger(opt)
    if err != nil {
        panic(err)
    }

    // 创建 GORM 适配器
    gormLogger := gorm.NewGormAdapter(logger)

    // 配置 GORM 数据库（示例）
    // db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
    //     Logger: gormLogger,
    // })
}
```

### 高级配置

```go
// 自定义配置
config := gorm.Config{
    LogLevel:                  gorm.Info,              // 日志级别
    SlowThreshold:             300 * time.Millisecond, // 慢查询阈值  
    IgnoreRecordNotFoundError: false,                  // 记录 RecordNotFound 错误
}

gormLogger := gorm.NewGormAdapterWithConfig(logger, config)

// 动态配置调整
gormLogger.SetSlowThreshold(500 * time.Millisecond)
gormLogger.SetIgnoreRecordNotFoundError(true)
```

## 🔧 配置选项

### LogLevel 日志级别

```go
const (
    Silent gorm.LogLevel = iota + 1  // 静默模式，不记录任何日志
    Error                            // 仅记录错误
    Warn                             // 记录错误和警告（慢查询）
    Info                             // 记录所有操作（推荐）
)
```

### Config 配置结构

```go
type Config struct {
    LogLevel                  LogLevel      // 日志级别
    SlowThreshold             time.Duration // 慢查询阈值
    IgnoreRecordNotFoundError bool          // 是否忽略 RecordNotFound 错误
}

// 获取默认配置
config := gorm.DefaultConfig()
// LogLevel: Info
// SlowThreshold: 200ms  
// IgnoreRecordNotFoundError: true
```

## 📊 日志输出示例

### 正常查询日志

```json
{
  "level": "info",
  "timestamp": "2025-08-30T13:45:30.123456789Z",
  "message": "Database query executed",
  "component": "gorm",
  "operation": "query",
  "sql": "SELECT * FROM `users` WHERE `users`.`id` = ? AND `users`.`deleted_at` IS NULL",
  "duration_ms": 2.45,
  "rows": 1,
  "caller": "service/user.go:25"
}
```

### 慢查询警告

```json
{
  "level": "warn",
  "timestamp": "2025-08-30T13:45:31.456789012Z", 
  "message": "Slow database query detected",
  "component": "gorm",
  "operation": "slow_query",
  "sql": "SELECT * FROM `orders` WHERE created_at > ? ORDER BY amount DESC LIMIT 1000",
  "duration_ms": 856.7,
  "threshold_ms": 500.0,
  "slowdown_factor": 1.71,
  "rows": 1000,
  "caller": "service/order.go:45"
}
```

### 错误日志

```json
{
  "level": "error",
  "timestamp": "2025-08-30T13:45:32.789012345Z",
  "message": "Database query failed", 
  "component": "gorm",
  "operation": "query",
  "sql": "INSERT INTO `products` (`name`,`price`) VALUES (?,?)",
  "error": "Error 1062: Duplicate entry 'product-123' for key 'name'",
  "duration_ms": 1.23,
  "caller": "service/product.go:67"
}
```

## 🎯 使用场景

### 1. 开发环境 - 详细调试

```go
func setupDevelopmentDB(logger core.Logger) gorm.Interface {
    config := gorm.Config{
        LogLevel:      gorm.Info,  // 记录所有操作
        SlowThreshold: 100 * time.Millisecond, // 较低的慢查询阈值
        IgnoreRecordNotFoundError: false, // 记录所有错误
    }
    
    return gorm.NewGormAdapterWithConfig(logger, config)
}
```

### 2. 生产环境 - 性能监控

```go
func setupProductionDB(logger core.Logger) gorm.Interface {
    config := gorm.Config{
        LogLevel:      gorm.Warn, // 只记录警告和错误
        SlowThreshold: 1 * time.Second, // 较高的慢查询阈值
        IgnoreRecordNotFoundError: true, // 忽略常见的 NotFound 错误
    }
    
    return gorm.NewGormAdapterWithConfig(logger, config)
}
```

### 3. 测试环境 - 静默模式

```go
func setupTestDB(logger core.Logger) gorm.Interface {
    config := gorm.Config{
        LogLevel:      gorm.Silent, // 静默模式
        SlowThreshold: 0,           // 禁用慢查询检测
        IgnoreRecordNotFoundError: true,
    }
    
    return gorm.NewGormAdapterWithConfig(logger, config)
}
```

## 🔍 高级功能

### 动态日志级别调整

```go
// 运行时调整日志级别
gormLogger := gorm.NewGormAdapter(logger)

// 切换到详细模式进行调试
debugLogger := gormLogger.LogMode(gorm.Info)

// 切换到静默模式减少日志
silentLogger := gormLogger.LogMode(gorm.Silent)

// 只记录错误
errorOnlyLogger := gormLogger.LogMode(gorm.Error)
```

### 上下文追踪

```go
import "context"

// 带有追踪 ID 的上下文
ctx := context.WithValue(context.Background(), "trace_id", "trace-123")

// GORM 会自动传递上下文到日志适配器
// result := db.WithContext(ctx).Find(&users)
// 日志中会包含上下文信息
```

### 慢查询监控

```go
// 设置慢查询阈值
gormLogger.SetSlowThreshold(500 * time.Millisecond)

// 获取当前阈值
threshold := gormLogger.GetSlowThreshold()
fmt.Printf("当前慢查询阈值: %v\n", threshold)

// 慢查询会自动触发警告日志
// 包含 slowdown_factor 字段显示超出阈值的倍数
```

### 错误过滤控制

```go
// 忽略 RecordNotFound 错误（常用于查询不存在的记录）
gormLogger.SetIgnoreRecordNotFoundError(true)

// 记录所有错误（调试模式）
gormLogger.SetIgnoreRecordNotFoundError(false)

// 检查当前设置
ignore := gormLogger.GetIgnoreRecordNotFoundError()
fmt.Printf("忽略 RecordNotFound 错误: %v\n", ignore)
```

## 🧪 测试支持

### 单元测试

```go
func TestGormAdapter(t *testing.T) {
    // 创建测试日志器
    opt := option.DefaultLogOption()
    logger, err := factory.NewLogger(opt)
    require.NoError(t, err)
    
    // 创建适配器
    gormLogger := gorm.NewGormAdapter(logger)
    
    // 测试基本功能
    assert.Equal(t, "GORM", gormLogger.Name())
    assert.Equal(t, "v1.x", gormLogger.Version())
    
    // 测试配置
    gormLogger.SetSlowThreshold(1 * time.Second)
    assert.Equal(t, 1*time.Second, gormLogger.GetSlowThreshold())
}
```

### 运行测试

```bash
# 运行 GORM 适配器测试
go test github.com/kart-io/logger/integrations/gorm -v

# 测试覆盖率
go test github.com/kart-io/logger/integrations/gorm -cover

# 运行基准测试
go test github.com/kart-io/logger/integrations/gorm -bench=.
```

## 📊 性能特征

### 基准测试结果

| 操作类型 | 时延 | 内存分配 | 说明 |
|----------|------|----------|------|
| Info 级别查询 | +15ns | +1 alloc | 正常查询记录 |
| Error 级别查询 | +20ns | +2 allocs | 错误查询记录 |
| Silent 模式 | +2ns | +0 allocs | 静默模式几乎无开销 |
| 慢查询检测 | +25ns | +2 allocs | 包含慢查询分析 |

### 内存优化

- **字段复用**: 重用常用字段切片减少分配
- **条件记录**: 根据日志级别避免不必要的字符串格式化
- **上下文缓存**: 高效的上下文信息提取

## 🔧 实战示例

### 电商系统集成

```go
// models/database.go
func InitDatabase() *gorm.DB {
    // 创建日志器
    opt := &option.LogOption{
        Engine: "zap",
        Level:  "INFO", 
        Format: "json",
    }
    logger, _ := factory.NewLogger(opt)
    
    // GORM 配置
    config := gorm.Config{
        LogLevel:      gorm.Info,
        SlowThreshold: 500 * time.Millisecond, // 电商查询通常较复杂
        IgnoreRecordNotFoundError: true,       // 商品不存在是正常情况
    }
    
    gormLogger := gorm.NewGormAdapterWithConfig(logger, config)
    
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: gormLogger,
    })
    
    return db
}

// service/product.go  
func (s *ProductService) GetProduct(id uint) (*Product, error) {
    var product Product
    
    // 这个查询会被自动记录
    result := s.db.First(&product, id)
    if result.Error != nil {
        // 如果是 RecordNotFound，且配置为忽略，则不会产生错误日志
        return nil, result.Error
    }
    
    return &product, nil
}
```

### 微服务架构集成

```go
// 服务启动配置
func setupLogger() core.Logger {
    opt := &option.LogOption{
        Engine: "zap",
        Level:  "INFO",
        Format: "json",
        OTLPEndpoint: "http://jaeger:14268/api/traces", // 链路追踪
    }
    
    logger, err := factory.NewLogger(opt)
    if err != nil {
        panic(err)
    }
    
    return logger
}

// 每个微服务的数据库配置
func setupUserServiceDB(logger core.Logger) *gorm.DB {
    config := gorm.Config{
        LogLevel:      gorm.Info,
        SlowThreshold: 200 * time.Millisecond, // 用户服务要求快速响应
        IgnoreRecordNotFoundError: true,
    }
    
    gormLogger := gorm.NewGormAdapterWithConfig(logger, config)
    // ... 数据库连接配置
}

func setupOrderServiceDB(logger core.Logger) *gorm.DB {
    config := gorm.Config{
        LogLevel:      gorm.Warn,
        SlowThreshold: 1 * time.Second, // 订单服务允许较慢的复杂查询
        IgnoreRecordNotFoundError: false, // 订单不存在需要记录
    }
    
    gormLogger := gorm.NewGormAdapterWithConfig(logger, config)
    // ... 数据库连接配置
}
```

## 🔗 相关资源

- [GORM 官方文档](https://gorm.io/docs/)
- [GORM Logger 接口文档](https://gorm.io/docs/logger.html)
- [`integrations`](../README.md) - 集成包总览
- [`core`](../../core/) - 核心接口定义  
- [`factory`](../../factory/) - 日志器工厂
- [`example/comprehensive`](../../example/comprehensive/) - 完整使用示例

## ⚠️ 注意事项

### 性能影响

1. **日志级别**: `Silent` 模式性能影响最小，`Info` 模式会记录所有查询
2. **慢查询阈值**: 设置合理的阈值避免过多警告日志
3. **上下文传递**: 大量上下文字段可能影响性能

### 错误处理

1. **RecordNotFound**: 根据业务需求决定是否忽略
2. **连接错误**: 数据库连接问题会产生 Error 级别日志
3. **SQL 语法错误**: 会产生详细的错误日志和 SQL 语句

### 安全考虑

1. **SQL 参数**: 参数会被安全地记录，不会暴露在 SQL 字符串中
2. **敏感数据**: 避免在表名、字段名中包含敏感信息
3. **日志轮转**: 确保日志文件定期轮转避免磁盘占满

## 🚀 最佳实践

### 环境区分配置

```go
// ✅ 推荐：根据环境调整配置
func createGormConfig(env string) gorm.Config {
    switch env {
    case "development":
        return gorm.Config{
            LogLevel:      gorm.Info,
            SlowThreshold: 100 * time.Millisecond,
            IgnoreRecordNotFoundError: false,
        }
    case "production":
        return gorm.Config{
            LogLevel:      gorm.Warn, 
            SlowThreshold: 1 * time.Second,
            IgnoreRecordNotFoundError: true,
        }
    case "test":
        return gorm.Config{
            LogLevel:      gorm.Silent,
            SlowThreshold: 0,
            IgnoreRecordNotFoundError: true,
        }
    default:
        return gorm.DefaultConfig()
    }
}
```

### 监控仪表板集成

```go
// ✅ 推荐：结构化日志便于监控系统解析
// 配置 ELK、Prometheus 等监控系统
// 基于 component=gorm 和 operation 字段创建仪表板
// 监控慢查询数量、错误率等关键指标
```

### 错误告警

```go
// ✅ 推荐：基于日志配置告警规则
// 1. 慢查询数量超过阈值
// 2. 数据库错误率过高
// 3. 特定 SQL 模式的异常
```

选择 GORM 集成，为你的数据库操作提供专业的日志记录和监控能力！ 🚀