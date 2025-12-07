# Core Package

核心接口和类型定义包，提供统一日志库的基础抽象。

## 📋 包含内容

- `Logger` 接口：统一的日志记录接口
- `Level` 类型：日志级别定义和解析
- 级别相关的常量和函数

## 🚀 快速使用

### 日志级别 (Level)

```go
package main

import (
    "fmt"
    "github.com/kart-io/logger/core"
)

func main() {
    // 使用预定义的日志级别
    fmt.Println(core.DebugLevel.String()) // "debug"
    fmt.Println(core.InfoLevel.String())  // "info"
    fmt.Println(core.WarnLevel.String())  // "warn"
    fmt.Println(core.ErrorLevel.String()) // "error"
    fmt.Println(core.FatalLevel.String()) // "fatal"

    // 从字符串解析级别
    level, err := core.ParseLevel("DEBUG")
    if err != nil {
        panic(err)
    }
    fmt.Println(level) // debug

    // 支持大小写不敏感解析
    level2, _ := core.ParseLevel("info")
    level3, _ := core.ParseLevel("INFO")
    fmt.Println(level2 == level3) // true
}
```

### Logger 接口

`Logger` 接口定义了统一的日志记录方法，所有日志引擎都必须实现此接口。

```go
type Logger interface {
    // 基础日志方法
    Debug(args ...interface{})
    Info(args ...interface{})
    Warn(args ...interface{})
    Error(args ...interface{})
    Fatal(args ...interface{})

    // 格式化日志方法
    Debugf(template string, args ...interface{})
    Infof(template string, args ...interface{})
    Warnf(template string, args ...interface{})
    Errorf(template string, args ...interface{})
    Fatalf(template string, args ...interface{})

    // 结构化日志方法
    Debugw(msg string, keysAndValues ...interface{})
    Infow(msg string, keysAndValues ...interface{})
    Warnw(msg string, keysAndValues ...interface{})
    Errorw(msg string, keysAndValues ...interface{})
    Fatalw(msg string, keysAndValues ...interface{})

    // 功能增强方法
    With(keysAndValues ...interface{}) Logger
    WithCtx(ctx context.Context) Logger
    WithCallerSkip(skip int) Logger
    SetLevel(level Level)
}
```

## 📊 日志级别

支持以下日志级别，按严重程度递增：

| 级别 | 值 | 描述 | 使用场景 |
|------|-----|------|----------|
| `DebugLevel` | -1 | 调试信息 | 开发调试，生产环境通常关闭 |
| `InfoLevel` | 0 | 一般信息 | 应用程序正常运行信息 |
| `WarnLevel` | 1 | 警告信息 | 需要注意但不影响运行的问题 |
| `ErrorLevel` | 2 | 错误信息 | 发生错误但程序可继续运行 |
| `FatalLevel` | 3 | 致命错误 | 严重错误，程序需要退出 |

## 🔧 级别解析

`ParseLevel` 函数支持以下格式：

```go
// 支持的字符串格式（大小写不敏感）
validLevels := []string{
    "DEBUG", "debug",
    "INFO", "info",
    "WARN", "warn", "WARNING", "warning",
    "ERROR", "error",
    "FATAL", "fatal",
}
```

## 💡 设计理念

### 接口隔离

核心包只定义接口，不包含具体实现，遵循依赖倒置原则：

- 高层模块（应用代码）不依赖低层模块（具体引擎）
- 都依赖于抽象（Logger接口）
- 具体实现在各自的引擎包中

### 方法分类

Logger 接口按调用风格分为三类：

1. **基础方法**: `Debug(args...)` - 类似 fmt.Print
2. **格式化方法**: `Debugf(template, args...)` - 类似 fmt.Printf
3. **结构化方法**: `Debugw(msg, keyvals...)` - 键值对结构化日志

### 级别一致性

所有级别相关功能都统一输出小写字符串：

- `Level.String()` 返回小写级别名
- 底层引擎配置确保输出一致性
- 解析函数支持大小写不敏感输入

## 🔍 错误处理

```go
// 解析错误示例
_, err := core.ParseLevel("INVALID")
if parseErr, ok := err.(*core.ParseLevelError); ok {
    fmt.Printf("无效的级别: %s\n", parseErr.Error())
}
```

## 🧪 测试支持

核心包提供了完整的测试覆盖：

```bash
go test github.com/kart-io/logger/core -v
```

## 📚 相关包

- [`engines/slog`](../engines/slog/) - Go标准库slog引擎实现
- [`engines/zap`](../engines/zap/) - Uber Zap引擎实现
- [`factory`](../factory/) - 日志器工厂，创建具体实现
- [`option`](../option/) - 配置选项管理

## ⚠️ 注意事项

1. `Fatal` 级别的日志会调用 `os.Exit(1)` 终止程序
2. 级别比较：数值越大级别越高，`FatalLevel > ErrorLevel > WarnLevel > InfoLevel > DebugLevel`
3. 接口中的 `keysAndValues` 参数必须成对出现（key-value pairs）
4. 上下文相关的方法（`WithCtx`, `WithCallerSkip`）返回新的Logger实例，不修改原实例
