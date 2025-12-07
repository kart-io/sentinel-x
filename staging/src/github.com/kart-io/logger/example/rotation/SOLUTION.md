# 日志轮转问题解决方案

## 问题描述

运行日志轮转示例时遇到错误：
```
Error: engine_error [factory]: both zap and slog engines failed 
(caused by: open sink "./logs/app.log": open ./logs/app.log: no such file or directory)
```

## 问题根因

**核心问题**：kart-io/logger 在使用 `OutputPaths` 配置文件路径时，不会自动创建目录。当指定的日志目录不存在时，引擎初始化会失败。

## 解决方案

### 方案一：预先创建目录（推荐）

在创建 logger 之前确保目录存在：

```go
import (
    "os"
    "github.com/kart-io/logger"
    "github.com/kart-io/logger/option"
)

func createLogger() (core.Logger, error) {
    // 1. 确保日志目录存在
    logDir := "./logs"  
    if err := os.MkdirAll(logDir, 0755); err != nil {
        return nil, fmt.Errorf("创建日志目录失败: %w", err)
    }

    // 2. 创建 logger 配置
    opt := &option.LogOption{
        Engine:      "zap",
        Level:       "INFO", 
        Format:      "json",
        OutputPaths: []string{"./logs/app.log"},
    }

    // 3. 创建 logger
    return logger.New(opt)
}
```

### 方案二：使用标准输出

避免文件路径问题，使用 stdout：

```go
opt := &option.LogOption{
    Engine:      "zap",
    Level:       "INFO",
    Format:      "json", 
    OutputPaths: []string{"stdout"}, // 不会出错
}
```

### 方案三：直接集成 lumberjack（完整轮转功能）

```go
import (
    "gopkg.in/natefinch/lumberjack.v2"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func createRotationLogger() *zap.Logger {
    // 创建轮转写入器
    writer := &lumberjack.Logger{
        Filename:   "./logs/app.log",
        MaxSize:    100,  // 100MB
        MaxBackups: 15,   // 保留15个文件
        MaxAge:     15,   // 15天后删除
        Compress:   true, // 压缩旧文件
        LocalTime:  true,
    }

    // 配置 zap core
    encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
    core := zapcore.NewCore(encoder, zapcore.AddSync(writer), zapcore.InfoLevel)
    
    return zap.New(core, zap.AddCaller())
}
```

## 可运行示例

我们提供了几个可运行的示例：

### 1. 简单测试（`test.go`）
```bash
go run test.go
```
- 纯 lumberjack 功能测试
- 无依赖问题
- 验证基础轮转功能

### 2. 完整示例（`simple_example.go`）  
```bash
go run simple_example.go
```
- kart-io/logger + lumberjack 集成
- 自动创建目录
- 完整错误处理

### 3. 生产环境示例（`main.go`）
```bash
go run main.go
```
- 完整的 RotationLogger 实现
- 多种轮转策略演示
- 生产级配置

## 最佳实践

### 1. 目录管理
```go
// 总是确保目录存在
func ensureLogDir(path string) error {
    dir := filepath.Dir(path)
    return os.MkdirAll(dir, 0755)
}
```

### 2. 错误处理
```go
// 优雅降级到标准输出
func createLoggerWithFallback(logPath string) core.Logger {
    // 尝试文件输出
    if err := ensureLogDir(logPath); err == nil {
        if logger, err := createFileLogger(logPath); err == nil {
            return logger
        }
    }
    
    // 降级到标准输出
    return createStdoutLogger()
}
```

### 3. 权限设置
```bash
# 生产环境目录权限
mkdir -p /var/log/myapp
chown myapp:myapp /var/log/myapp
chmod 755 /var/log/myapp
```

## 部署检查清单

- [ ] 日志目录存在且有写权限
- [ ] 磁盘空间充足
- [ ] 轮转配置合理（大小、数量、时间）
- [ ] 监控和告警已配置
- [ ] 应急清理脚本已准备

## 常见陷阱

1. **相对路径问题**：使用绝对路径避免工作目录变化
2. **权限问题**：确保应用有目录写权限
3. **磁盘空间**：监控磁盘使用避免写满
4. **并发安全**：lumberjack 是线程安全的，但要注意配置共享

## 性能考虑

- **缓冲写入**：使用 `zapcore.NewBufferedWriteSyncer` 
- **批量提交**：避免每条日志都 flush
- **合理轮转**：平衡文件数量和查询效率
- **压缩策略**：延迟压缩减少 I/O 影响

通过这些解决方案，您可以成功实现 7天分割、15天保存的日志轮转需求。