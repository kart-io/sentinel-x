# Enhanced Structured Logging with Context Propagation

This implementation provides context-aware structured logging capabilities for the Sentinel-X framework, with automatic trace correlation, error handling, and multi-tenant support.

## Features

### 1. Context-Aware Logger (`pkg/infra/logger/fields.go`)

Extract and attach logger fields from context with automatic propagation:

- **Request ID propagation**: Track requests across service boundaries
- **OpenTelemetry integration**: Automatic trace_id and span_id extraction
- **Multi-tenant support**: user_id and tenant_id fields for tenant isolation
- **Thread-safe field management**: Immutable context updates
- **Error context**: Structured error logging with type information

### 2. Enhanced Logger Middleware (`pkg/infra/middleware/logger_enhanced.go`)

HTTP middleware with comprehensive request/response logging:

- **Automatic trace correlation**: Extracts OpenTelemetry span context
- **Response metrics**: Status code, response size, latency_ms
- **Error categorization**: Distinguishes 4xx (client) vs 5xx (server) errors
- **Header redaction**: Configurable sensitive header filtering
- **Body capture**: Optional request/response body logging with size limits
- **Performance optimized**: Uses sync.Pool for field allocations

### 3. Context Logger (`pkg/infra/logger/context.go`)

High-level API for context-aware logging:

- **ContextLogger**: Wraps logger with automatic context field injection
- **Error logging**: Structured error fields with optional stack traces
- **Error chain unwrapping**: Logs complete error chains for debugging
- **Convenience functions**: Simple API for common logging patterns

### 4. Enhanced Configuration (`pkg/infra/logger/options.go`)

Extended configuration for advanced logging features:

```go
type EnhancedLoggerConfig struct {
    EnableTraceCorrelation   bool      // Auto-extract OpenTelemetry traces
    EnableResponseLogging    bool      // Log response metrics
    EnableRequestLogging     bool      // Log request details
    SensitiveHeaders         []string  // Headers to redact
    MaxBodyLogSize           int       // Max body size to log
    CaptureStackTrace        bool      // Capture stack traces for errors
    ErrorStackTraceMinStatus int       // Min status for stack traces (default: 500)
    SkipPaths                []string  // Paths to skip logging
    LogRequestBody           bool      // Enable request body logging
    LogResponseBody          bool      // Enable response body logging
}
```

## Usage Examples

### Basic Context Fields

```go
import applogger "github.com/kart-io/sentinel-x/pkg/infra/logger"

// Add request tracking
ctx := applogger.WithRequestID(ctx, "req-12345")
ctx = applogger.WithUserID(ctx, "user-67890")
ctx = applogger.WithTenantID(ctx, "tenant-001")

log := applogger.GetLogger(ctx)
log.Infow("User request processed", "action", "login", "duration_ms", 123)
```

Output:
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

### Trace Correlation

```go
// Manual trace ID injection
ctx = applogger.WithTraceID(ctx, "trace-abc123")
ctx = applogger.WithSpanID(ctx, "span-xyz789")

// Or automatic extraction from OpenTelemetry
ctx = applogger.ExtractOpenTelemetryFields(ctx)

log := applogger.GetLogger(ctx)
log.Infow("Database query executed", "query", "SELECT * FROM users", "rows", 42)
```

### Error Logging

```go
// Simple error context
err := fmt.Errorf("database connection failed")
ctx = applogger.WithError(ctx, err)
ctx = applogger.WithErrorCode(ctx, "DB_CONN_ERR")

log := applogger.GetLogger(ctx)
log.Errorw("Failed to process request")

// Error chain logging
baseErr := fmt.Errorf("connection timeout")
wrappedErr := fmt.Errorf("failed to fetch data: %w", baseErr)
applogger.LogErrorChain(ctx, "Request failed", wrappedErr, true) // captures stack trace
```

### Multiple Fields at Once

```go
ctx = applogger.WithFields(ctx,
    "request_id", "req-11111",
    "user_id", "user-22222",
    "api_version", "v2",
    "region", "us-east-1",
)

log := applogger.GetLogger(ctx)
log.Infow("API request completed", "status", 200, "response_time_ms", 45)
```

### Context Logger

```go
ctxLogger := applogger.NewContextLogger(ctx)
ctxLogger.Infow("Using context logger", "feature", "enhanced_logging")
ctxLogger.Debugw("Debug information", "debug_level", 3)

// Update context
newCtx := applogger.WithUserID(ctx, "user-999")
ctxLogger = ctxLogger.WithContext(newCtx)
```

### Middleware Usage

```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
    applogger "github.com/kart-io/sentinel-x/pkg/infra/logger"
)

// Initialize logger
opts := applogger.NewOptions()
opts.Enhanced.EnableTraceCorrelation = true
opts.Enhanced.EnableResponseLogging = true
opts.Enhanced.SensitiveHeaders = []string{"Authorization", "Cookie"}
opts.Enhanced.MaxBodyLogSize = 1024

// Use middleware
router.Use(middleware.EnhancedLogger())

// Or with custom config
config := middleware.EnhancedLoggerConfig{
    EnhancedLoggerConfig: opts.Enhanced,
}
router.Use(middleware.EnhancedLoggerWithConfig(config))
```

## API Reference

### Context Field Functions

- `WithRequestID(ctx, requestID string)` - Add request_id to context
- `WithTraceID(ctx, traceID string)` - Add trace_id to context
- `WithSpanID(ctx, spanID string)` - Add span_id to context
- `WithUserID(ctx, userID string)` - Add user_id to context
- `WithTenantID(ctx, tenantID string)` - Add tenant_id to context
- `WithError(ctx, err error)` - Add structured error fields
- `WithErrorCode(ctx, code string)` - Add error_code field
- `WithFields(ctx, keysAndValues...)` - Add multiple fields at once
- `ExtractOpenTelemetryFields(ctx)` - Extract trace fields from OpenTelemetry span

### Logger Functions

- `GetLogger(ctx)` - Get logger with context fields
- `NewContextLogger(ctx)` - Create context-aware logger wrapper
- `LogError(ctx, msg, err, captureStack bool)` - Log error with optional stack trace
- `LogErrorChain(ctx, msg, err, captureStack bool)` - Log error with unwrapped chain
- `LogInfo(ctx, msg, keysAndValues...)` - Convenience info logging
- `LogDebug(ctx, msg, keysAndValues...)` - Convenience debug logging
- `LogWarn(ctx, msg, keysAndValues...)` - Convenience warning logging

### Configuration Functions

- `NewOptions()` - Create options with enhanced config defaults
- `DefaultEnhancedLoggerConfig()` - Get default enhanced config

## Architecture

### Thread Safety

All context operations are thread-safe through immutable updates:

1. Context fields are stored in immutable `loggerFields` structs
2. Each `With*` function creates a new context with cloned fields
3. Field maps are never modified after creation
4. Sync.Pool is used for temporary field slices in middleware

### Performance Optimization

- **Field pooling**: Reuses field slices to reduce allocations
- **Lazy evaluation**: Logger fields only extracted when needed
- **Efficient lookups**: Sensitive headers and skip paths use maps (O(1))
- **Zero-copy operations**: Context fields passed by reference

### OpenTelemetry Integration

The implementation automatically extracts trace context from OpenTelemetry:

1. Checks if span is recording
2. Validates span context
3. Extracts trace_id and span_id
4. Includes trace_sampled flag if applicable

This works seamlessly with OpenTelemetry auto-instrumentation.

## Testing

Comprehensive test suites are provided:

- **`pkg/infra/logger/context_test.go`**: Context logger tests
- **`pkg/infra/middleware/logger_enhanced_test.go`**: Middleware tests

Run tests:

```bash
# Run all logger tests
go test ./pkg/infra/logger/... -v

# Run middleware tests
go test ./pkg/infra/middleware/... -run TestEnhancedLogger -v

# Run benchmarks
go test ./pkg/infra/logger/... -bench=. -benchmem
```

## Example Application

See `examples/enhanced_logging/main.go` for a complete demonstration:

```bash
go run examples/enhanced_logging/main.go
```

## Files Created

1. `/Users/costalong/code/go/src/github.com/kart/sentinel-x/pkg/infra/logger/fields.go`
   - Context field management
   - OpenTelemetry integration
   - Helper functions

2. `/Users/costalong/code/go/src/github.com/kart/sentinel-x/pkg/infra/logger/context.go`
   - ContextLogger implementation
   - Error logging utilities
   - Stack trace capture

3. `/Users/costalong/code/go/src/github.com/kart/sentinel-x/pkg/infra/logger/options.go` (updated)
   - EnhancedLoggerConfig struct
   - Configuration flags
   - Default values

4. `/Users/costalong/code/go/src/github.com/kart/sentinel-x/pkg/infra/middleware/logger_enhanced.go`
   - Enhanced logging middleware
   - Response writer wrapper
   - Request/response body capture

5. `/Users/costalong/code/go/src/github.com/kart/sentinel-x/pkg/infra/logger/context_test.go`
   - Comprehensive test suite
   - Benchmark tests

6. `/Users/costalong/code/go/src/github.com/kart/sentinel-x/pkg/infra/middleware/logger_enhanced_test.go`
   - Middleware test suite
   - Mock transport.Context implementation

7. `/Users/costalong/code/go/src/github.com/kart/sentinel-x/examples/enhanced_logging/main.go`
   - Complete usage examples
   - Demo application

## Best Practices

1. **Always use context**: Pass context through your application to maintain field continuity
2. **Add fields early**: Set request_id, user_id, tenant_id at request entry points
3. **Use GetLogger**: Always retrieve logger via `GetLogger(ctx)` to include context fields
4. **Configure sensitive headers**: Update `SensitiveHeaders` for your application's security needs
5. **Limit body logging**: Set appropriate `MaxBodyLogSize` to avoid log bloat
6. **Enable trace correlation**: Use OpenTelemetry for distributed tracing
7. **Test with race detector**: Run `go test -race` to verify thread safety

## Migration Guide

### From Standard Logger

```go
// Before
logger.Info("User logged in")

// After
log := applogger.GetLogger(ctx)
log.Info("User logged in")
```

### From Logger Middleware

```go
// Before
router.Use(middleware.Logger())

// After
router.Use(middleware.EnhancedLogger())
```

### Adding Context Fields

```go
// In middleware or handler
func MyHandler(c transport.Context) {
    ctx := c.Request()
    ctx = applogger.WithRequestID(ctx, GetRequestID(ctx))
    ctx = applogger.WithUserID(ctx, GetCurrentUserID(c))
    c.SetRequest(ctx)

    // Now all downstream logging includes these fields
    log := applogger.GetLogger(ctx)
    log.Info("Processing request")
}
```

## Future Enhancements

Potential improvements for future iterations:

1. **Sampling**: Add configurable sampling for high-volume endpoints
2. **Metrics export**: Export log metrics to Prometheus
3. **Log aggregation hints**: Add fields to optimize log aggregation systems
4. **Dynamic filtering**: Runtime-configurable log level per user/tenant
5. **Correlation ID propagation**: Automatic propagation across microservices
6. **Performance profiling**: Built-in latency profiling integration
