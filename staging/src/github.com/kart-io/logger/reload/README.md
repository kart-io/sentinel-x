# Reload Package

动态配置重载包，提供运行时配置变更功能，支持文件监控、信号处理和 API 触发等多种重载方式。

## 概述

`reload` 包实现了完整的动态配置管理功能：

- **多触发方式**：文件监控、信号处理、API 调用
- **配置验证**：重载前的配置验证和回滚机制
- **备份管理**：自动备份历史配置，支持回滚
- **优雅处理**：异步处理，不阻塞主进程
- **错误恢复**：配置失败时的自动恢复机制
- **生产就绪**：完整的错误处理和日志记录

## 核心组件

### ConfigReloader

```go
type ConfigReloader struct {
    // 内部状态管理
    config        *ReloadConfig
    currentConfig *config.Config
    factory       *factory.LoggerFactory
    
    // 监控组件
    watcher    *fsnotify.Watcher
    signalChan chan os.Signal
    reloadChan chan *config.Config
    
    // 备份和错误处理
    backupConfigs []*config.Config
    errorHandler  *errors.ErrorHandler
}
```

### ReloadConfig

```go
type ReloadConfig struct {
    ConfigFile           string          // 配置文件路径
    Triggers            ReloadTrigger   // 触发方式
    Signals             []os.Signal     // 监听的信号
    ValidateBeforeReload bool           // 重载前验证
    ValidationFunc      ValidationFunc  // 自定义验证函数
    ReloadTimeout       time.Duration   // 重载超时时间
    BackupOnReload      bool           // 是否备份旧配置
    BackupRetention     int            // 备份保留数量
    Callback            ReloadCallback  // 重载完成回调
    Logger              core.Logger     // 内部日志记录器
}
```

## 触发方式

### ReloadTrigger 类型

```go
const (
    TriggerNone      ReloadTrigger = 0
    TriggerSignal    ReloadTrigger = 1 << iota  // 信号触发
    TriggerFileWatch                           // 文件监控触发  
    TriggerAPI                                 // API 调用触发
    TriggerAll = TriggerSignal | TriggerFileWatch | TriggerAPI
)
```

## 使用方式

### 1. 基本文件监控重载

```go
package main

import (
    "github.com/kart-io/logger/reload"
    "github.com/kart-io/logger/config"
    "github.com/kart-io/logger/factory"
    "github.com/kart-io/logger/option"
)

func main() {
    // 初始配置
    initialConfig := config.DefaultConfig()
    
    // 创建工厂
    opt := option.DefaultLogOption()
    factory := factory.NewLoggerFactory(opt)
    
    // 配置重载器
    reloadConfig := &reload.ReloadConfig{
        ConfigFile:           "logger.yaml",
        Triggers:            reload.TriggerFileWatch,
        ValidateBeforeReload: true,
        BackupOnReload:      true,
        BackupRetention:     5,
    }
    
    // 创建重载器
    reloader, err := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
    if err != nil {
        panic(err)
    }
    
    // 启动重载器
    if err := reloader.Start(); err != nil {
        panic(err)
    }
    defer reloader.Stop()
    
    // 应用继续运行，配置文件变更时会自动重载
    select {}
}
```

### 2. 信号处理重载

```go
func signalReloadExample() {
    reloadConfig := &reload.ReloadConfig{
        ConfigFile: "logger.yaml",
        Triggers:   reload.TriggerSignal,
        Signals:    []os.Signal{syscall.SIGUSR1, syscall.SIGHUP},
    }
    
    reloader, _ := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
    reloader.Start()
    
    // 现在可以通过信号触发重载：
    // kill -USR1 <pid>  # 触发配置重载
    // kill -HUP <pid>   # 触发配置重载
}
```

### 3. API 触发重载

```go
func apiReloadExample() {
    reloadConfig := &reload.ReloadConfig{
        Triggers: reload.TriggerAPI,
    }
    
    reloader, _ := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
    reloader.Start()
    
    // HTTP API 端点
    http.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
        // 解析新配置
        var newConfig config.Config
        if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
            http.Error(w, "Invalid config", http.StatusBadRequest)
            return
        }
        
        // 触发重载
        if err := reloader.TriggerReload(&newConfig); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"})
    })
}
```

### 4. 综合使用（多触发方式）

```go
func comprehensiveReloadExample() {
    // 创建日志器用于重载器内部日志
    logger, _ := logger.NewWithDefaults()
    
    reloadConfig := &reload.ReloadConfig{
        ConfigFile:           "logger.yaml",
        Triggers:            reload.TriggerAll, // 启用所有触发方式
        Signals:             []os.Signal{syscall.SIGUSR1, syscall.SIGHUP},
        ValidateBeforeReload: true,
        ValidationFunc:      customValidationFunc,
        ReloadTimeout:       30 * time.Second,
        BackupOnReload:      true,
        BackupRetention:     10,
        Callback:            reloadCallback,
        Logger:              logger,
    }
    
    reloader, err := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
    if err != nil {
        panic(err)
    }
    
    if err := reloader.Start(); err != nil {
        panic(err)
    }
    
    // 现在支持：
    // 1. 文件变更自动重载
    // 2. 信号触发重载 (kill -USR1 <pid>)
    // 3. API 调用重载
    
    defer reloader.Stop()
}

// 自定义验证函数
func customValidationFunc(cfg *config.Config) error {
    if cfg.Engine != "zap" && cfg.Engine != "slog" {
        return fmt.Errorf("unsupported engine: %s", cfg.Engine)
    }
    
    if cfg.Level == "" {
        return fmt.Errorf("log level cannot be empty")
    }
    
    return nil
}

// 重载完成回调
func reloadCallback(oldConfig, newConfig *config.Config) error {
    fmt.Printf("配置重载完成: %s -> %s\n", oldConfig.Engine, newConfig.Engine)
    
    // 可以执行额外的重载逻辑
    // 比如通知其他组件、更新监控指标等
    
    return nil
}
```

## 配置文件格式

### YAML 配置示例

```yaml
# logger.yaml
engine: "zap"
level: "info"
format: "json"
output-paths: 
  - "stdout"
  - "/var/log/app.log"

# OTLP 配置
otlp-endpoint: "http://localhost:4317"
otlp:
  protocol: "grpc"
  timeout: "10s"
  headers:
    Authorization: "Bearer token123"

# 功能开关
development: false
disable-caller: false
disable-stacktrace: false
```

### JSON 配置示例

```json
{
  "engine": "slog",
  "level": "debug",
  "format": "console",
  "output_paths": ["stdout"],
  "otlp_endpoint": "http://localhost:4317",
  "development": true
}
```

## 备份和回滚

### 查看备份配置

```go
func backupManagementExample(reloader *reload.ConfigReloader) {
    // 获取当前配置
    currentConfig := reloader.GetCurrentConfig()
    fmt.Printf("当前引擎: %s, 级别: %s\n", currentConfig.Engine, currentConfig.Level)
    
    // 获取所有备份配置
    backups := reloader.GetBackupConfigs()
    fmt.Printf("备份配置数量: %d\n", len(backups))
    
    for i, backup := range backups {
        fmt.Printf("备份 %d: 引擎=%s, 级别=%s\n", i, backup.Engine, backup.Level)
    }
}
```

### 回滚到上一个配置

```go
func rollbackExample(reloader *reload.ConfigReloader) {
    // 回滚到上一个配置
    if err := reloader.RollbackToPrevious(); err != nil {
        fmt.Printf("回滚失败: %v\n", err)
    } else {
        fmt.Println("成功回滚到上一个配置")
    }
    
    // 查看回滚后的配置
    currentConfig := reloader.GetCurrentConfig()
    fmt.Printf("回滚后引擎: %s\n", currentConfig.Engine)
}
```

## 错误处理和监控

### 错误处理策略

```go
func errorHandlingExample() {
    reloadConfig := &reload.ReloadConfig{
        ConfigFile:           "logger.yaml",
        Triggers:            reload.TriggerFileWatch,
        ValidateBeforeReload: true,
        ValidationFunc: func(cfg *config.Config) error {
            // 严格验证
            if cfg.Engine == "" {
                return fmt.Errorf("engine cannot be empty")
            }
            
            // 检查 OTLP 配置
            if cfg.IsOTLPEnabled() && cfg.OTLP.Endpoint == "" {
                return fmt.Errorf("OTLP enabled but endpoint is empty")
            }
            
            return nil
        },
        ReloadTimeout:   15 * time.Second,
        BackupOnReload:  true,
        BackupRetention: 3,
        
        // 重载完成回调，用于错误通知
        Callback: func(oldConfig, newConfig *config.Config) error {
            // 发送重载成功通知
            notifyConfigReload(oldConfig, newConfig)
            return nil
        },
    }
    
    reloader, err := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
    if err != nil {
        handleReloaderInitError(err)
        return
    }
    
    if err := reloader.Start(); err != nil {
        handleReloaderStartError(err)
        return
    }
}

func handleReloaderInitError(err error) {
    // 记录初始化错误
    log.Errorf("重载器初始化失败: %v", err)
    
    // 可能需要降级处理，比如禁用动态配置功能
    runWithStaticConfig()
}

func handleReloaderStartError(err error) {
    // 记录启动错误
    log.Errorf("重载器启动失败: %v", err)
    
    // 尝试重新启动或降级
    retryReloaderStart()
}
```

### 监控集成

```go
func monitoringIntegrationExample(reloader *reload.ConfigReloader) {
    // 设置监控指标
    reloadConfig := &reload.ReloadConfig{
        Callback: func(oldConfig, newConfig *config.Config) error {
            // 更新监控指标
            metrics.IncrementCounter("config_reloads_total", 1)
            metrics.RecordHistogram("config_reload_duration", time.Since(start))
            
            // 记录配置变更
            if oldConfig.Engine != newConfig.Engine {
                metrics.IncrementCounter("config_engine_changes", 1, 
                    map[string]string{
                        "from": oldConfig.Engine,
                        "to":   newConfig.Engine,
                    })
            }
            
            if oldConfig.Level != newConfig.Level {
                metrics.IncrementCounter("config_level_changes", 1,
                    map[string]string{
                        "from": oldConfig.Level,
                        "to":   newConfig.Level,
                    })
            }
            
            return nil
        },
        
        ValidationFunc: func(cfg *config.Config) error {
            start := time.Now()
            defer func() {
                metrics.RecordHistogram("config_validation_duration", time.Since(start))
            }()
            
            if err := cfg.Validate(); err != nil {
                metrics.IncrementCounter("config_validation_errors", 1)
                return err
            }
            
            metrics.IncrementCounter("config_validations_success", 1)
            return nil
        },
    }
}
```

## HTTP API 集成

### 完整的 Web 管理接口

```go
func setupReloadAPI(reloader *reload.ConfigReloader) {
    // 获取当前配置
    http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
            config := reloader.GetCurrentConfig()
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(config)
            
        case "PUT":
            var newConfig config.Config
            if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
                http.Error(w, "Invalid JSON", http.StatusBadRequest)
                return
            }
            
            if err := reloader.TriggerReload(&newConfig); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"})
        }
    })
    
    // 获取备份配置
    http.HandleFunc("/config/backups", func(w http.ResponseWriter, r *http.Request) {
        backups := reloader.GetBackupConfigs()
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "count":   len(backups),
            "backups": backups,
        })
    })
    
    // 回滚配置
    http.HandleFunc("/config/rollback", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        
        if err := reloader.RollbackToPrevious(); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        currentConfig := reloader.GetCurrentConfig()
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "rolled_back",
            "config": currentConfig,
        })
    })
    
    // 健康检查
    http.HandleFunc("/config/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status":        "healthy",
            "backup_count":  len(reloader.GetBackupConfigs()),
            "current_engine": reloader.GetCurrentConfig().Engine,
        })
    })
}
```

## 最佳实践

### 1. 生产环境配置

```go
func productionReloadSetup() *reload.ConfigReloader {
    // 生产环境推荐配置
    reloadConfig := &reload.ReloadConfig{
        ConfigFile:           "/etc/myapp/logger.yaml",
        Triggers:            reload.TriggerSignal | reload.TriggerAPI, // 不用文件监控避免意外重载
        Signals:             []os.Signal{syscall.SIGUSR1},            // 只监听 USR1
        ValidateBeforeReload: true,                                   // 必须验证
        ValidationFunc:      strictValidationFunc,                   // 严格验证
        ReloadTimeout:       10 * time.Second,                       // 较短超时
        BackupOnReload:      true,                                   // 必须备份
        BackupRetention:     3,                                      // 保留最近3个
        Callback:           productionCallback,                     // 生产回调
        Logger:             productionLogger,                       // 生产日志器
    }
    
    reloader, err := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
    if err != nil {
        log.Fatalf("Failed to create config reloader: %v", err)
    }
    
    return reloader
}

func strictValidationFunc(cfg *config.Config) error {
    // 生产环境严格验证
    if cfg.Development {
        return fmt.Errorf("development mode not allowed in production")
    }
    
    if cfg.Level == "DEBUG" {
        return fmt.Errorf("debug level not allowed in production")
    }
    
    // 确保有适当的输出路径
    hasFile := false
    for _, path := range cfg.OutputPaths {
        if path != "stdout" && path != "stderr" {
            hasFile = true
            break
        }
    }
    if !hasFile {
        return fmt.Errorf("production requires file output")
    }
    
    return nil
}
```

### 2. 开发环境配置

```go
func developmentReloadSetup() *reload.ConfigReloader {
    // 开发环境配置
    reloadConfig := &reload.ReloadConfig{
        ConfigFile:           "logger-dev.yaml",
        Triggers:            reload.TriggerAll,           // 启用所有触发方式
        ValidateBeforeReload: false,                     // 宽松验证
        ReloadTimeout:       30 * time.Second,           // 长超时
        BackupOnReload:      true,
        BackupRetention:     10,                         // 更多备份
        Logger:              developmentLogger,
    }
    
    reloader, _ := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
    return reloader
}
```

### 3. 容器化部署

```go
func containerReloadSetup() *reload.ConfigReloader {
    reloadConfig := &reload.ReloadConfig{
        Triggers: reload.TriggerSignal | reload.TriggerAPI,
        Signals:  []os.Signal{syscall.SIGUSR1},
        // 容器中通常不用文件监控，而是通过信号或API
        ValidateBeforeReload: true,
        BackupOnReload:      true,
        BackupRetention:     5,
    }
    
    reloader, _ := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
    return reloader
}
```

## 故障处理

### 常见问题及解决

1. **配置文件不存在**
```go
// 检查文件是否存在
if _, err := os.Stat(configFile); os.IsNotExist(err) {
    log.Warnf("Config file %s does not exist, using defaults", configFile)
    useDefaultConfig()
}
```

2. **配置验证失败**
```go
ValidationFunc: func(cfg *config.Config) error {
    if err := cfg.Validate(); err != nil {
        // 记录验证错误但不阻止启动
        log.Errorf("Config validation failed: %v, using previous config", err)
        return err
    }
    return nil
}
```

3. **重载超时处理**
```go
reloadConfig.ReloadTimeout = 30 * time.Second
reloadConfig.Callback = func(oldConfig, newConfig *config.Config) error {
    // 设置超时上下文
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    return applyConfigWithTimeout(ctx, newConfig)
}
```

## API 参考

### 主要方法

| 方法 | 描述 |
|------|------|
| `NewConfigReloader(config, initial, factory)` | 创建配置重载器 |
| `Start()` | 启动重载器 |
| `Stop()` | 停止重载器 |
| `TriggerReload(config)` | 手动触发重载 |
| `GetCurrentConfig()` | 获取当前配置 |
| `GetBackupConfigs()` | 获取备份配置 |
| `RollbackToPrevious()` | 回滚到上一个配置 |

### 配置选项

| 选项 | 类型 | 描述 |
|------|------|------|
| `ConfigFile` | `string` | 配置文件路径 |
| `Triggers` | `ReloadTrigger` | 触发方式位掩码 |
| `Signals` | `[]os.Signal` | 监听的信号列表 |
| `ValidateBeforeReload` | `bool` | 重载前是否验证 |
| `ValidationFunc` | `ValidationFunc` | 自定义验证函数 |
| `ReloadTimeout` | `time.Duration` | 重载超时时间 |
| `BackupOnReload` | `bool` | 是否备份旧配置 |
| `BackupRetention` | `int` | 备份保留数量 |
| `Callback` | `ReloadCallback` | 重载完成回调 |
| `Logger` | `core.Logger` | 内部日志记录器 |

## 注意事项

1. **线程安全**：所有公共方法都是线程安全的
2. **资源清理**：确保调用 `Stop()` 清理资源
3. **文件权限**：确保有读取配置文件的权限
4. **信号处理**：避免信号冲突，选择合适的信号
5. **备份管理**：合理设置备份保留数量，避免内存泄漏
6. **错误处理**：实现适当的错误处理和降级策略

## 相关包

- [`config`](../config/) - 配置结构定义
- [`factory`](../factory/) - 日志器工厂
- [`option`](../option/) - 配置选项
- [`errors`](../errors/) - 错误处理机制