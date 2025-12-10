# Enhanced Logging Quick Reference

## Quick Start

```go
import applogger "github.com/kart-io/sentinel-x/pkg/infra/logger"

// Initialize
opts := applogger.NewOptions()
opts.Init()

// Add context fields
ctx = applogger.WithRequestID(ctx, "req-123")
ctx = applogger.WithUserID(ctx, "user-456")

// Get logger and log
log := applogger.GetLogger(ctx)
log.Infow("Request processed", "status", 200)
```

## Context Functions

| Function | Purpose | Example |
|----------|---------|---------|
| `WithRequestID(ctx, id)` | Add request ID | `ctx = applogger.WithRequestID(ctx, "req-123")` |
| `WithTraceID(ctx, id)` | Add trace ID | `ctx = applogger.WithTraceID(ctx, "trace-abc")` |
| `WithSpanID(ctx, id)` | Add span ID | `ctx = applogger.WithSpanID(ctx, "span-xyz")` |
| `WithUserID(ctx, id)` | Add user ID | `ctx = applogger.WithUserID(ctx, "user-456")` |
| `WithTenantID(ctx, id)` | Add tenant ID | `ctx = applogger.WithTenantID(ctx, "tenant-001")` |
| `WithError(ctx, err)` | Add error fields | `ctx = applogger.WithError(ctx, err)` |
| `WithErrorCode(ctx, code)` | Add error code | `ctx = applogger.WithErrorCode(ctx, "ERR_DB")` |
| `WithFields(ctx, kv...)` | Add multiple fields | `ctx = applogger.WithFields(ctx, "k1", "v1", "k2", "v2")` |
| `GetLogger(ctx)` | Get context logger | `log := applogger.GetLogger(ctx)` |
| `ExtractOpenTelemetryFields(ctx)` | Extract OTel traces | `ctx = applogger.ExtractOpenTelemetryFields(ctx)` |

## Logging Functions

| Function | Purpose | Example |
|----------|---------|---------|
| `LogInfo(ctx, msg, kv...)` | Log info message | `applogger.LogInfo(ctx, "User login", "user", "john")` |
| `LogDebug(ctx, msg, kv...)` | Log debug message | `applogger.LogDebug(ctx, "Query", "sql", "SELECT...")` |
| `LogWarn(ctx, msg, kv...)` | Log warning | `applogger.LogWarn(ctx, "Slow query", "ms", 1000)` |
| `LogError(ctx, msg, err, stack)` | Log error | `applogger.LogError(ctx, "Failed", err, true)` |
| `LogErrorChain(ctx, msg, err, stack)` | Log error chain | `applogger.LogErrorChain(ctx, "Failed", err, true)` |

## ContextLogger

```go
// Create
ctxLogger := applogger.NewContextLogger(ctx)

// Use
ctxLogger.Info("Message")
ctxLogger.Infow("Message", "key", "value")
ctxLogger.Debugw("Debug", "level", 3)

// Update context
newLogger := ctxLogger.WithContext(newCtx)

// Add fields
newLogger = ctxLogger.WithFields("key", "value")
```

## Middleware

```go
import "github.com/kart-io/sentinel-x/pkg/infra/middleware"

// Default configuration
router.Use(middleware.EnhancedLogger())

// Custom configuration
config := middleware.EnhancedLoggerConfig{
    EnhancedLoggerConfig: &applogger.EnhancedLoggerConfig{
        EnableTraceCorrelation: true,
        EnableResponseLogging:  true,
        SensitiveHeaders:       []string{"Authorization"},
        MaxBodyLogSize:         1024,
    },
}
router.Use(middleware.EnhancedLoggerWithConfig(config))
```

## Configuration

```go
opts := applogger.NewOptions()
opts.Enhanced = &applogger.EnhancedLoggerConfig{
    EnableTraceCorrelation:   true,   // Auto-extract OTel traces
    EnableResponseLogging:    true,   // Log response metrics
    EnableRequestLogging:     true,   // Log request details
    SensitiveHeaders:         []string{"Authorization", "Cookie"},
    MaxBodyLogSize:           1024,   // Max body bytes to log
    CaptureStackTrace:        false,  // Capture stack on errors
    ErrorStackTraceMinStatus: 500,    // Min status for stack traces
    SkipPaths:                []string{"/health"},
    LogRequestBody:           false,  // Log request body
    LogResponseBody:          false,  // Log response body
}
```

## Common Patterns

### Request Handler

```go
func HandleRequest(c transport.Context) {
    ctx := c.Request()

    // Add request context
    ctx = applogger.WithRequestID(ctx, GetRequestID(ctx))
    ctx = applogger.WithUserID(ctx, GetUserID(c))
    c.SetRequest(ctx)

    // Log with context
    log := applogger.GetLogger(ctx)
    log.Infow("Processing request", "endpoint", "/api/users")

    // ... handle request
}
```

### Error Handling

```go
func ProcessData(ctx context.Context) error {
    log := applogger.GetLogger(ctx)

    if err := doSomething(); err != nil {
        // Log error with context
        applogger.LogError(ctx, "Failed to process", err, true)
        return fmt.Errorf("processing failed: %w", err)
    }

    log.Infow("Processing complete", "items", 10)
    return nil
}
```

### Service Layer

```go
type UserService struct {
    logger applogger.ContextLogger
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    // Get context logger
    log := applogger.GetLogger(ctx)

    log.Infow("Creating user", "username", user.Username)

    if err := s.db.Create(user); err != nil {
        applogger.LogError(ctx, "Failed to create user", err, false)
        return err
    }

    log.Infow("User created", "user_id", user.ID)
    return nil
}
```

## Output Examples

### Info Log
```json
{
  "timestamp": "2025-12-10T22:07:47.432433+08:00",
  "level": "info",
  "message": "User request processed",
  "request_id": "req-12345",
  "user_id": "user-67890",
  "tenant_id": "tenant-001",
  "action": "login",
  "duration_ms": 123
}
```

### Error Log
```json
{
  "timestamp": "2025-12-10T22:07:47.432571+08:00",
  "level": "error",
  "message": "Failed to process request",
  "request_id": "req-99999",
  "error_type": "*errors.errorString",
  "error_code": "DB_CONN_ERR",
  "error_message": "database connection failed",
  "stacktrace": "..."
}
```

### Trace Correlation
```json
{
  "timestamp": "2025-12-10T22:07:47.432528+08:00",
  "level": "info",
  "message": "Database query executed",
  "trace_id": "trace-abc123",
  "span_id": "span-xyz789",
  "query": "SELECT * FROM users",
  "rows": 42
}
```

## Performance

- `WithRequestID`: **112 ns/op** - 6 allocs
- `GetLogger`: **1.2 μs/op** - 18 allocs
- `GetContextFields`: **79 ns/op** - 4 allocs

## Files

- **Implementation**: `pkg/infra/logger/fields.go`, `context.go`
- **Middleware**: `pkg/infra/middleware/logger_enhanced.go`
- **Config**: `pkg/infra/logger/options.go`
- **Tests**: `pkg/infra/logger/context_test.go`
- **Example**: `examples/enhanced_logging/main.go`
- **Docs**: `docs/ENHANCED_LOGGING.md`
