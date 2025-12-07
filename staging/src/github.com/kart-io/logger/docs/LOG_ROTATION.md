# 日志轮转集成指南

## 概述

kart-io/logger 库专注于提供统一的日志接口和多引擎支持，**不直接实现日志文件轮转功能**。对于生产环境中的日志轮转需求（如按天分割、定期清理），建议采用以下集成方案：

## 集成方案

### 方案一：lumberjack 集成（推荐）

[lumberjack](https://github.com/natefinch/lumberjack) 是 Go 生态中最流行的日志轮转库，提供了丰富的轮转策略。

#### 安装依赖

```bash
go get gopkg.in/natefinch/lumberjack.v2
```

#### 基本集成

```go
package main

import (
    "io"
    "gopkg.in/natefinch/lumberjack.v2"
    "github.com/kart-io/logger"
    "github.com/kart-io/logger/option"
)

func main() {
    // 创建 lumberjack 轮转写入器
    rotateWriter := &lumberjack.Logger{
        Filename:   "/var/log/app/app.log",  // 日志文件路径
        MaxSize:    100,                     // 每个日志文件最大尺寸（MB）
        MaxBackups: 15,                      // 保留的备份文件数量（15天）
        MaxAge:     15,                      // 文件最大保存天数
        Compress:   true,                    // 是否压缩旧文件
        LocalTime:  true,                    // 使用本地时间
    }

    // 配置 logger 使用自定义写入器
    opt := &option.LogOption{
        Engine:      "zap",
        Level:       "INFO",
        Format:      "json",
        // 注意：不设置 OutputPaths，而是通过自定义配置
    }

    // 创建支持轮转的 logger（需要扩展工厂方法）
    logger := createLoggerWithRotation(opt, rotateWriter)
    
    // 使用 logger
    logger.Info("应用程序启动")
    logger.Infow("用户登录", "user_id", "12345", "ip", "192.168.1.1")
}
```

#### 高级配置示例

```go
// 按小时轮转的配置
func createHourlyRotationLogger() core.Logger {
    rotateWriter := &lumberjack.Logger{
        Filename:   "/var/log/app/app.log",
        MaxSize:    50,    // 50MB 触发轮转
        MaxBackups: 168,   // 保留 7天 * 24小时 = 168 个文件
        MaxAge:     7,     // 7天后自动清理
        Compress:   true,
        LocalTime:  true,
    }
    
    return createLoggerWithRotation(defaultOption(), rotateWriter)
}

// 按天轮转的配置
func createDailyRotationLogger() core.Logger {
    rotateWriter := &lumberjack.Logger{
        Filename:   "/var/log/app/daily.log", 
        MaxSize:    500,   // 500MB 触发轮转
        MaxBackups: 15,    // 保留 15 天
        MaxAge:     15,    // 15天后自动清理
        Compress:   true,
        LocalTime:  true,
    }
    
    return createLoggerWithRotation(defaultOption(), rotateWriter)
}

func defaultOption() *option.LogOption {
    return &option.LogOption{
        Engine: "zap",
        Level:  "INFO", 
        Format: "json",
    }
}
```

### 方案二：系统级轮转（logrotate）

对于 Linux 系统，可以使用系统自带的 `logrotate` 工具。

#### logrotate 配置文件

创建 `/etc/logrotate.d/myapp`：

```bash
/var/log/myapp/*.log {
    daily                    # 每天轮转
    missingok               # 文件不存在时不报错
    rotate 15               # 保留 15 天
    compress                # 压缩旧文件
    delaycompress           # 延迟压缩（下次轮转时压缩）
    notifempty             # 空文件不轮转
    create 0644 app app     # 创建新文件的权限和所有者
    postrotate
        # 重新打开日志文件（发送 USR1 信号）
        /bin/kill -USR1 $(cat /var/run/myapp.pid) 2>/dev/null || true
    endscript
}
```

#### 应用程序配置

```go
// 使用标准文件输出，依赖 logrotate 轮转
func main() {
    opt := &option.LogOption{
        Engine:      "zap",
        Level:       "INFO",
        Format:      "json",
        OutputPaths: []string{"/var/log/myapp/app.log"},
    }
    
    logger, err := logger.New(opt)
    if err != nil {
        panic(err)
    }
    
    // 设置信号处理，支持 logrotate 重新打开文件
    setupSignalHandling(logger)
    
    logger.Info("应用程序启动")
}
```

### 方案三：多写入器组合

结合控制台输出和文件轮转：

```go
func createMultiOutputLogger() core.Logger {
    // 控制台输出（开发环境）
    consoleWriter := os.Stdout
    
    // 文件轮转输出（生产环境）
    fileWriter := &lumberjack.Logger{
        Filename:   "/var/log/app/app.log",
        MaxSize:    100,
        MaxBackups: 15, 
        MaxAge:     15,
        Compress:   true,
        LocalTime:  true,
    }
    
    // 组合多个写入器
    multiWriter := io.MultiWriter(consoleWriter, fileWriter)
    
    return createLoggerWithWriter(defaultOption(), multiWriter)
}
```

## 生产环境最佳实践

### 1. 轮转策略选择

```go
// 高流量应用：按大小 + 时间双重触发
rotateWriter := &lumberjack.Logger{
    Filename:   "/var/log/app/app.log",
    MaxSize:    200,    // 200MB 或者...
    MaxBackups: 72,     // 保留 3天 * 24小时 = 72个文件
    MaxAge:     3,      // 3天后强制清理
    Compress:   true,
    LocalTime:  true,
}

// 中等流量：每日轮转
rotateWriter := &lumberjack.Logger{
    Filename:   "/var/log/app/daily.log",
    MaxSize:    500,    // 500MB 触发紧急轮转
    MaxBackups: 15,     // 保留 15 天
    MaxAge:     15,     // 15天清理
    Compress:   true,
    LocalTime:  true,
}

// 低流量应用：按周轮转
rotateWriter := &lumberjack.Logger{
    Filename:   "/var/log/app/weekly.log", 
    MaxSize:    1000,   // 1GB
    MaxBackups: 4,      // 保留 4 周
    MaxAge:     28,     // 28天清理
    Compress:   true,
    LocalTime:  true,
}
```

### 2. 磁盘空间管理

```go
// 严格控制磁盘使用的配置
func createSpaceEfficientLogger() core.Logger {
    rotateWriter := &lumberjack.Logger{
        Filename:   "/var/log/app/app.log",
        MaxSize:    50,     // 较小的文件尺寸
        MaxBackups: 10,     // 较少的备份数量
        MaxAge:     7,      // 较短的保留时间
        Compress:   true,   // 必须压缩
        LocalTime:  true,
    }
    
    return createLoggerWithRotation(opt, rotateWriter)
}
```

### 3. 监控和告警

```go
// 集成监控的日志轮转
func createMonitoredLogger() core.Logger {
    rotateWriter := &MonitoredLumberjack{
        Logger: &lumberjack.Logger{
            Filename:   "/var/log/app/app.log",
            MaxSize:    100,
            MaxBackups: 15,
            MaxAge:     15,
            Compress:   true,
            LocalTime:  true,
        },
        MetricsCollector: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "log_rotation_total",
                Help: "Total number of log rotations",
            },
            []string{"reason"},
        ),
    }
    
    return createLoggerWithRotation(opt, rotateWriter)
}

type MonitoredLumberjack struct {
    *lumberjack.Logger
    MetricsCollector *prometheus.CounterVec
}

func (m *MonitoredLumberjack) Write(p []byte) (n int, err error) {
    // 检测是否发生轮转
    oldSize := m.size()
    n, err = m.Logger.Write(p)
    newSize := m.size()
    
    if newSize < oldSize {
        // 发生了轮转
        m.MetricsCollector.WithLabelValues("size").Inc()
    }
    
    return n, err
}
```

## 配置建议

### 开发环境
```go
// 开发环境：控制台 + 本地文件
opt := &option.LogOption{
    Engine:      "slog",
    Level:       "DEBUG", 
    Format:      "console",
    OutputPaths: []string{"stdout", "./logs/dev.log"},
    Development: true,
}
```

### 测试环境
```go
// 测试环境：简单轮转
rotateWriter := &lumberjack.Logger{
    Filename:   "/var/log/test/app.log",
    MaxSize:    10,    // 小文件便于测试
    MaxBackups: 3,     // 少量备份
    MaxAge:     1,     // 短期保留
    Compress:   false, // 不压缩便于查看
}
```

### 生产环境
```go
// 生产环境：完整轮转策略
rotateWriter := &lumberjack.Logger{
    Filename:   "/var/log/prod/app.log",
    MaxSize:    200,   // 根据日志量调整
    MaxBackups: 15,    // 业务需求决定
    MaxAge:     15,    // 合规要求
    Compress:   true,  // 节省磁盘空间
    LocalTime:  true,  // 使用本地时间
}
```

## 扩展实现

由于当前 kart-io/logger 不直接支持自定义写入器，您可能需要扩展工厂方法：

```go
// 扩展的工厂方法（需要在项目中实现）
func createLoggerWithRotation(opt *option.LogOption, writer io.Writer) core.Logger {
    // 根据选择的引擎创建支持自定义写入器的 logger
    // 这需要修改现有的工厂实现或创建扩展版本
}
```

建议向 kart-io/logger 项目提交 feature request，增加对自定义写入器的支持。

## 总结

kart-io/logger 专注于统一日志接口，日志轮转通过外部集成实现：

1. **推荐方案**：lumberjack + 自定义工厂方法
2. **系统方案**：logrotate + 信号处理  
3. **生产部署**：根据流量和磁盘限制选择合适策略
4. **监控集成**：添加轮转事件监控和告警

这种分离设计符合单一职责原则，让日志库专注于日志记录，轮转策略交给专门的工具处理。