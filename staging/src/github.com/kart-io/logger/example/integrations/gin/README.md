# Gin Integration Example

This example demonstrates how to integrate the Kart Logger with the Gin web framework for structured HTTP request logging.

## Features Demonstrated

### 🔧 Core Integration Features
- **Unified Logging Interface**: Uses Kart Logger's core interface for consistent logging
- **Request/Response Logging**: Automatic logging of HTTP requests with structured data
- **Error Handling**: Proper error logging and panic recovery
- **Context Management**: Request ID and user context tracking
- **Middleware Chain**: Comprehensive middleware setup for production use

### 📊 Logging Capabilities
- **Request Metrics**: Method, path, status code, latency, body size
- **User Context**: User ID and request ID tracking across requests  
- **Error Logging**: Detailed error information with stack traces
- **Body Logging**: Optional request/response body logging (configurable)
- **Health Check Filtering**: Skip logging for health check endpoints
- **Panic Recovery**: Graceful panic recovery with detailed logging

### ⚙️ Configuration Options
- **Log Levels**: Configurable minimum log level
- **Body Size Limits**: Configurable maximum body size for logging
- **Path Filtering**: Skip logging for specific paths or patterns
- **Time Formatting**: Customizable timestamp formats
- **Client Error Handling**: Optional skipping of 4xx status codes

## Quick Start

1. **Install dependencies**:
```bash
cd example/integrations/gin
go mod download
```

2. **Run the example**:
```bash
go run main.go
```

3. **Test the endpoints**:
```bash
# Health checks (skipped in logs)
curl http://localhost:8080/ping
curl http://localhost:8080/health

# API operations (logged with full details)
curl http://localhost:8080/api/v1/users
curl http://localhost:8080/api/v1/users/1

# Create user with request body logging
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Jane Doe","email":"jane@example.com"}'

# Request with custom headers for context tracking
curl -H "X-Request-ID: my-req-123" \
     -H "X-User-ID: user-456" \
     http://localhost:8080/api/v1/users

# Error scenarios
curl http://localhost:8080/api/v1/users/404  # 404 error
curl http://localhost:8080/api/v1/error     # 500 error  
curl http://localhost:8080/api/v1/panic     # Panic recovery
```

## Code Structure

### Main Components

#### 1. **Gin Adapter Setup**
```go
// Create adapter with custom configuration
ginConfig := gin_adapter.Config{
    LogLevel:        core.InfoLevel,
    LogRequestBody:  true,
    LogResponseBody: false,
    MaxBodySize:     1024,
    SkipClientError: false,
    LogLatency:      true,
    SkipPaths:       []string{"/ping", "/metrics"},
}

ginAdapter := gin_adapter.NewGinAdapterWithConfig(coreLogger, ginConfig)
```

#### 2. **Middleware Chain Setup**
```go
// Custom middleware setup (recommended approach)
router.Use(ginAdapter.RequestIDMiddleware("X-Request-ID"))
router.Use(ginAdapter.UserContextMiddleware("X-User-ID"))  
router.Use(ginAdapter.HealthCheckSkipper("/ping", "/health"))
router.Use(ginAdapter.RequestBodyLogger(1024))
router.Use(ginAdapter.Logger())
router.Use(ginAdapter.MetricsMiddleware())
router.Use(ginAdapter.Recovery())
```

#### 3. **Alternative Middleware Setups**
```go
// Option 1: Default middleware
for _, middleware := range ginAdapter.DefaultMiddleware() {
    router.Use(middleware)
}

// Option 2: Production middleware with health check filtering
for _, middleware := range ginAdapter.ProductionMiddleware("/ping", "/health") {
    router.Use(middleware)
}

// Option 3: Debug middleware (includes request body logging)
for _, middleware := range ginAdapter.DebugMiddleware() {
    router.Use(middleware)
}
```

## Log Output Examples

### Successful Request
```json
{
  "timestamp": "2024-09-06T19:30:15Z",
  "level": "INFO",
  "message": "[GIN] 2024-09-06T19:30:15Z | 200 |     1.234ms | 127.0.0.1 | GET     /api/v1/users",
  "component": "gin",
  "operation": "http_request",
  "status": 200,
  "latency": "1.234ms",
  "client_ip": "127.0.0.1",
  "method": "GET",
  "path": "/api/v1/users",
  "body_size": 156,
  "request_id": "req-1725654615123456",
  "user_id": "user-456"
}
```

### Error Logging
```json
{
  "timestamp": "2024-09-06T19:30:20Z", 
  "level": "ERROR",
  "message": "HTTP request failed",
  "component": "gin",
  "operation": "http_error",
  "method": "GET",
  "path": "/api/v1/users/404",
  "status_code": 404,
  "error": "user not found",
  "request_id": "req-1725654620789012"
}
```

### Panic Recovery
```json
{
  "timestamp": "2024-09-06T19:30:25Z",
  "level": "ERROR", 
  "message": "Panic recovered in Gin handler",
  "component": "gin",
  "operation": "panic_recovery",
  "method": "GET",
  "path": "/api/v1/panic",
  "client_ip": "127.0.0.1",
  "user_agent": "curl/7.88.1",
  "panic": "This is a simulated panic for testing recovery middleware",
  "error": "This is a simulated panic for testing recovery middleware"
}
```

## Advanced Usage

### Custom Log Formatter
```go
customFormatter := func(params gin_adapter.LogFormatterParams, adapter *gin_adapter.GinAdapter) {
    // Custom formatting logic
    fields := []interface{}{
        "custom_field", "custom_value",
        "timestamp", params.TimeStamp,
        "status", params.StatusCode,
        "method", params.Method,
        "path", params.Path,
    }
    
    adapter.GetLogger().Infow("Custom formatted request", fields...)
}

router.Use(ginAdapter.LoggerWithFormatter(customFormatter))
```

### Request Body Validation with Logging
```go
api.POST("/users", func(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        // Log validation error with request context
        ginAdapter.LogError(err, c.Request.Method, c.Request.URL.Path, http.StatusBadRequest)
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "validation_failed",
            Message: err.Error(),
        })
        return
    }
    // Handle valid request...
})
```

### Custom Middleware Integration
```go
// Add your own middleware that works with the logger
router.Use(func(c *gin.Context) {
    start := time.Now()
    
    c.Next()
    
    // Log custom metrics
    ginAdapter.GetLogger().Infow("Custom middleware executed",
        "middleware", "my_middleware",
        "duration", time.Since(start),
        "request_id", c.GetString("request_id"),
    )
})
```

## Configuration Best Practices

### Development Environment
```go
ginConfig := gin_adapter.Config{
    LogLevel:        core.DebugLevel,
    LogRequestBody:  true,   // Enable for debugging
    LogResponseBody: true,   // Enable for debugging  
    MaxBodySize:     10240,  // 10KB for development
    SkipClientError: false,  // Log all errors in development
}
```

### Production Environment
```go  
ginConfig := gin_adapter.Config{
    LogLevel:        core.InfoLevel,
    LogRequestBody:  false,  // Disable for security/performance
    LogResponseBody: false,  // Disable for security/performance
    MaxBodySize:     1024,   // 1KB limit
    SkipClientError: true,   // Skip 4xx errors to reduce noise
    SkipPaths:       []string{"/health", "/metrics", "/ping"},
}
```

## Integration with Other Systems

### OTLP/OpenTelemetry
The logger supports OTLP integration. Configure your logger with OTLP settings for distributed tracing:

```go
logConfig := option.LogOption{
    Level:  core.InfoLevel,
    Format: core.JSONFormat,
    OTLP: &option.OTLPOption{
        Endpoint: "http://jaeger:14268/api/traces",
        Enable:   true,
    },
}
```

### Prometheus Metrics
Combine with Prometheus middleware for comprehensive monitoring:

```go
router.Use(ginAdapter.Logger())
router.Use(gin_adapter.PrometheusMiddleware()) // Your Prometheus middleware
router.Use(ginAdapter.Recovery())
```

## Troubleshooting

### Common Issues

1. **Missing Request IDs**: Ensure `RequestIDMiddleware` is added before the logger middleware
2. **Health Checks Being Logged**: Use `HealthCheckSkipper` middleware before the logger
3. **Performance Issues**: Disable body logging in production environments
4. **Large Log Volume**: Use `SkipClientError: true` and appropriate `SkipPaths` configuration

### Debug Mode
Run with debug middleware for maximum visibility:

```go
for _, middleware := range ginAdapter.DebugMiddleware() {
    router.Use(middleware)
}
```

This enables:
- Request ID tracking  
- Request body logging
- Full request/response logging
- Metrics collection
- Panic recovery

## Performance Considerations

- **Body Logging**: Disable in production for better performance and security
- **Log Level**: Use `InfoLevel` or higher in production
- **Skip Paths**: Configure health check endpoints to avoid log noise
- **Buffer Size**: Consider buffered log outputs for high-traffic applications

---

This example provides a complete foundation for integrating structured logging into Gin applications using the Kart Logger framework.