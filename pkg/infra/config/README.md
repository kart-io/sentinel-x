# Configuration Hot Reload

This package provides configuration hot reload capabilities for the Sentinel-X application, allowing dynamic configuration changes without service restart.

## Overview

The configuration watcher monitors the application's YAML configuration file and notifies registered components when changes occur. Components implementing the `Reloadable` interface can react to configuration changes and update their behavior at runtime.

## Features

- **Thread-Safe**: All operations are safe for concurrent use
- **Flexible Subscription**: Components can subscribe/unsubscribe from config change notifications
- **Error Isolation**: Errors in one handler don't affect others
- **Type-Safe**: Configuration is validated before being applied
- **Atomic Updates**: Components maintain valid state if config update fails
- **Zero Downtime**: No service restart required for configuration changes

## Architecture

```
┌─────────────────────┐
│   Config File       │
│  (sentinel-api.yaml)│
└──────────┬──────────┘
           │ fsnotify
           ▼
┌─────────────────────┐
│   Config Watcher    │
│   (viper-based)     │
└──────────┬──────────┘
           │
           ├──────────┐
           │          │
           ▼          ▼
    ┌──────────┐ ┌──────────┐
    │  Logger  │ │Middleware│
    │ Reloader │ │ Reloader │
    └──────────┘ └──────────┘
```

## Components

### Core Types

#### `Reloadable` Interface
```go
type Reloadable interface {
    OnConfigChange(newConfig interface{}) error
}
```

#### `Watcher`
Manages configuration file watching and handler notifications.

#### `ReloadableSubscriber`
Helper that bridges the watcher to `Reloadable` components.

## Supported Hot-Reloadable Configurations

### Logger Configuration
- Log level (debug, info, warn, error, fatal)
- Log format (json, text)
- Output paths
- Development mode
- Caller and stacktrace settings

### Middleware Configuration
- **CORS**: Origins, methods, headers, credentials, max age
- **Timeout**: Duration and skip paths
- **Request ID**: Header name
- **Logger Middleware**: Skip paths, structured logging
- **Recovery**: Stack trace enablement
- **Health**: Endpoint paths
- **Metrics**: Path, namespace, subsystem
- **Pprof**: Profile rates

### Not Hot-Reloadable
These require service restart:
- Server listen addresses
- Enable/disable flags for middleware
- TLS certificates
- Database connection strings
- JWT signing keys

## Usage

### Basic Setup

```go
package main

import (
    "github.com/kart-io/sentinel-x/pkg/infra/config"
    "github.com/kart-io/sentinel-x/pkg/infra/logger"
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
    "github.com/spf13/viper"
)

func main() {
    // 1. Initialize viper and load config
    v := viper.New()
    v.SetConfigFile("configs/sentinel-api.yaml")
    if err := v.ReadInConfig(); err != nil {
        panic(err)
    }

    // 2. Parse configuration
    logOpts := logger.NewOptions()
    v.UnmarshalKey("log", logOpts)

    mwOpts := middleware.NewOptions()
    v.UnmarshalKey("server.http.middleware", mwOpts)

    // 3. Create reloadable components
    reloadableLogger := logger.NewReloadableLogger(logOpts)
    reloadableMiddleware := middleware.NewReloadableMiddleware(mwOpts)

    // 4. Create config watcher
    watcher := config.NewWatcher(v)

    // 5. Register components
    reloadableLogger.RegisterWithWatcher(watcher, "logger", "log")
    reloadableMiddleware.RegisterWithWatcher(watcher, "middleware", "server.http.middleware")

    // 6. Start watching
    watcher.Start()

    // Your application continues to run
    // Config changes are handled automatically
}
```

### Custom Reloadable Component

```go
type MyService struct {
    config MyConfig
    mu     sync.RWMutex
}

type MyConfig struct {
    MaxRetries int    `mapstructure:"max_retries"`
    Timeout    string `mapstructure:"timeout"`
}

func (s *MyService) OnConfigChange(newConfig interface{}) error {
    cfg, ok := newConfig.(*MyConfig)
    if !ok {
        return fmt.Errorf("invalid config type")
    }

    // Validate
    if cfg.MaxRetries < 0 {
        return fmt.Errorf("max_retries must be non-negative")
    }

    // Apply atomically
    s.mu.Lock()
    defer s.mu.Unlock()

    oldConfig := s.config
    s.config = *cfg

    logger.Infof("MyService config reloaded: retries %d -> %d",
        oldConfig.MaxRetries, cfg.MaxRetries)

    return nil
}

// Register with watcher
service := &MyService{}
target := &MyConfig{}
subscriber := config.NewReloadableSubscriber(service, "myservice", target)
watcher.Subscribe("myservice-handler", subscriber.Handler())
```

### Manual Handler Registration

For more control, register a custom handler:

```go
watcher.Subscribe("custom", func(v *viper.Viper) error {
    newValue := v.GetString("some.config.key")
    logger.Infof("Config changed: %s", newValue)

    // Apply your changes here
    return nil
})
```

## Configuration File Example

```yaml
# configs/sentinel-api.yaml
log:
  level: info           # Can be changed at runtime
  format: json          # Can be changed at runtime
  development: false

server:
  http:
    middleware:
      timeout:
        timeout: 30s    # Can be changed at runtime
        skip-paths:
          - /health
          - /metrics

      cors:
        allow-origins:  # Can be changed at runtime
          - "*"
        max-age: 86400  # Can be changed at runtime

      request-id:
        header: X-Request-ID  # Can be changed at runtime
```

## Testing

### Unit Tests
```bash
go test -v ./pkg/infra/config -run TestWatcher
```

### Integration Tests
```bash
go test -v ./pkg/infra/config -run TestIntegration
```

### Watch Tests
```bash
# Tests that require actual file system watching
go test -v ./pkg/infra/config -run TestConfigFileChange
```

## Best Practices

### 1. Validate Before Applying
Always validate new configuration before applying:
```go
func (c *Component) OnConfigChange(newConfig interface{}) error {
    cfg := newConfig.(*Config)

    if err := cfg.Validate(); err != nil {
        return err
    }

    // Apply changes...
    return nil
}
```

### 2. Support Rollback
Store previous state to rollback on failure:
```go
old := c.config
c.config = new

if err := c.applyConfig(); err != nil {
    c.config = old  // Rollback
    return err
}
```

### 3. Log Changes
Always log configuration changes for observability:
```go
logger.Infof("Config changed: timeout %v -> %v", old.Timeout, new.Timeout)
```

### 4. Thread Safety
Use RWMutex for safe concurrent access:
```go
type Component struct {
    config Config
    mu     sync.RWMutex
}

func (c *Component) GetConfig() Config {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.config
}
```

## Performance Considerations

- **Handler Execution**: Handlers are called sequentially, not concurrently
- **File Watch Overhead**: fsnotify uses OS-level file system events (minimal overhead)
- **Lock Contention**: Use RWMutex to allow multiple readers
- **Memory**: Config copies are created to prevent external modification

## Limitations

1. **Sequential Handlers**: Handlers execute one at a time, in no guaranteed order
2. **No Transaction Support**: Each handler applies changes independently
3. **File System Dependency**: Requires file system change notification support
4. **Config Size**: Very large config files may have parsing overhead

## Troubleshooting

### Config Changes Not Detected

**Problem**: File changes aren't triggering handlers

**Solutions**:
- Verify file watcher is started: `watcher.Start()`
- Check file permissions
- Some editors use atomic writes (rename) which may not trigger events
- Check OS file system notification limits (inotify on Linux)

### Handler Errors

**Problem**: Handler returns error on config change

**Solutions**:
- Check validation logic
- Ensure config structure matches mapstructure tags
- Review error logs for specific failure reasons
- Test handler with various config values

### Race Conditions

**Problem**: Concurrent access to config causes panics

**Solutions**:
- Use RWMutex for config access
- Create defensive copies of config
- Don't share config pointers across goroutines

## API Reference

### Watcher Methods

- `NewWatcher(v *viper.Viper) *Watcher` - Create new watcher
- `Subscribe(id string, handler ChangeHandler)` - Register handler
- `Unsubscribe(id string)` - Remove handler
- `Start()` - Begin watching config file
- `Stop()` - Stop watching
- `IsWatching() bool` - Check watch status
- `HandlerCount() int` - Get registered handler count

### ReloadableSubscriber Methods

- `NewReloadableSubscriber(component Reloadable, configKey string, target interface{}) *ReloadableSubscriber`
- `Handler() ChangeHandler` - Get the change handler function

## Examples

See `integration_test.go` for complete working examples including:
- Full application integration
- Logger reload
- Middleware reload
- Unsubscribe behavior

## Contributing

When adding new reloadable components:

1. Implement the `Reloadable` interface
2. Use `sync.RWMutex` for thread safety
3. Validate configuration before applying
4. Support rollback on failure
5. Log configuration changes
6. Add integration tests
7. Document hot-reloadable vs restart-required settings

## See Also

- [Viper Documentation](https://github.com/spf13/viper)
- [fsnotify Documentation](https://github.com/fsnotify/fsnotify)
- Logger Package: `pkg/infra/logger`
- Middleware Package: `pkg/infra/middleware`
