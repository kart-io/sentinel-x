# 🚀 Web Framework Integration Examples

This directory demonstrates how to integrate the unified logger with popular Go web frameworks, showcasing consistent, structured logging across different frameworks.

## 🎯 Overview

The unified logger provides consistent, structured logging across different web frameworks while preserving framework-specific features and optimizations.

## 📁 Examples

### 1. Gin Framework (`../gin/`)

Demonstrates integration with the Gin web framework using our unified logger.

**Features:**
- Custom Gin middleware for request/response logging
- Panic recovery with structured logging
- User context tracking
- Performance monitoring
- Error categorization based on HTTP status codes

**Key Components:**
- `GinLoggerMiddleware`: Logs all HTTP requests with detailed information
- `GinRecoveryMiddleware`: Handles panics and logs them appropriately
- Multiple endpoint examples showing different logging scenarios

### 2. Echo Framework (`../echo/`)

Demonstrates integration with the Echo web framework using our unified logger.

**Features:**
- Custom Echo middleware for request/response logging
- Route pattern logging (Echo-specific)
- Panic recovery with structured logging
- CORS middleware integration
- RESTful API examples with comprehensive logging

**Key Components:**
- `EchoLoggerMiddleware`: Logs all HTTP requests with Echo-specific details
- `EchoRecoveryMiddleware`: Handles panics and logs them appropriately
- RESTful API examples with CRUD operations

## 🚀 Quick Start

### Automated Demo

Run the comprehensive test script to see both frameworks in action:

```bash
cd example/web-frameworks
./test-demo.sh
```

This script will:
- Start both Gin (port 8080) and Echo (port 8081) servers
- Test key endpoints automatically  
- Display log output samples
- Provide interactive URLs for testing

### Individual Framework Testing

## 🛠️ Running the Examples

### Prerequisites

First, add the required dependencies:

```bash
# For Gin example
go get github.com/gin-gonic/gin

# For Echo example  
go get github.com/labstack/echo/v4
go get github.com/labstack/echo/v4/middleware
```

### Running Gin Example

```bash
cd example/gin
go run main.go
```

The server will start on `:8080`. Test endpoints:

- `GET http://localhost:8080/` - Welcome message
- `GET http://localhost:8080/users/123` - Get user by ID
- `POST http://localhost:8080/users` - Create user
- `GET http://localhost:8080/health` - Health check
- `GET http://localhost:8080/slow` - Slow endpoint (3s delay)
- `GET http://localhost:8080/error` - Error endpoint
- `GET http://localhost:8080/panic` - Panic endpoint
- `GET http://localhost:8080/docs` - API documentation

### Running Echo Example

```bash
cd example/echo
go run main.go
```

The server will start on `:8081`. Test endpoints:

- `GET http://localhost:8081/` - Welcome message
- `GET http://localhost:8081/api/products/123` - Get product by ID
- `POST http://localhost:8081/api/products` - Create product
- `PUT http://localhost:8081/api/products/123` - Update product
- `DELETE http://localhost:8081/api/products/123` - Delete product
- `GET http://localhost:8081/health` - Health check
- `GET http://localhost:8081/metrics` - Application metrics
- `GET http://localhost:8081/slow?delay=2` - Slow endpoint (2s delay)
- `GET http://localhost:8081/error?type=bad_request` - Error endpoint
- `GET http://localhost:8081/panic` - Panic endpoint
- `GET http://localhost:8081/docs` - API documentation

## Sample cURL Commands

### Gin Examples

```bash
# Welcome
curl http://localhost:8080/

# Get user
curl http://localhost:8080/users/123

# Create user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'

# Test error handling
curl http://localhost:8080/users/invalid

# Test slow endpoint
curl http://localhost:8080/slow
```

### Echo Examples

```bash
# Welcome
curl http://localhost:8081/

# Get product
curl http://localhost:8081/api/products/123

# Create product
curl -X POST http://localhost:8081/api/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Laptop", "price": 999.99, "category": "electronics", "description": "High-performance laptop"}'

# Update product
curl -X PUT http://localhost:8081/api/products/123 \
  -H "Content-Type: application/json" \
  -d '{"price": 899.99, "in_stock": false}'

# Delete product
curl -X DELETE http://localhost:8081/api/products/123

# Test different error types
curl http://localhost:8081/error?type=not_found
curl http://localhost:8081/error?type=unauthorized

# Test slow endpoint with custom delay
curl "http://localhost:8081/slow?delay=5"
```

## Log Output Examples

Both examples produce structured JSON logs. Here are sample outputs:

### Request Logging
```json
{
  "time": "2025-08-30T10:30:45.123Z",
  "level": "INFO", 
  "msg": "GET /api/products/123",
  "component": "echo",
  "method": "GET",
  "path": "/api/products/123",
  "route": "/api/products/:id",
  "status_code": 200,
  "latency_ms": 45.2,
  "client_ip": "127.0.0.1",
  "user_agent": "curl/7.68.0",
  "bytes_in": 0,
  "bytes_out": 157
}
```

### Error Logging
```json
{
  "time": "2025-08-30T10:31:12.456Z",
  "level": "ERROR",
  "msg": "GET /error",
  "component": "gin", 
  "method": "GET",
  "path": "/error",
  "status_code": 500,
  "latency_ms": 1.2,
  "client_ip": "127.0.0.1",
  "error_type": "intentional",
  "test": true
}
```

### Panic Recovery Logging
```json
{
  "time": "2025-08-30T10:31:30.789Z",
  "level": "ERROR",
  "msg": "Panic recovered",
  "component": "echo",
  "method": "GET", 
  "path": "/panic",
  "route": "/panic",
  "client_ip": "127.0.0.1",
  "error": "This is an intentional panic for testing recovery middleware",
  "panic": true
}
```

## Key Features Demonstrated

1. **Structured Logging**: All logs use structured JSON format with consistent field names
2. **Request Tracking**: Complete HTTP request/response cycle logging
3. **Error Handling**: Proper error categorization and logging
4. **Performance Monitoring**: Latency tracking and slow request detection
5. **Panic Recovery**: Graceful panic handling with detailed logging
6. **Context Propagation**: User and request context tracking throughout the request lifecycle
7. **Framework-Specific Features**: Leverage unique features of each framework (Gin groups, Echo route patterns)

## Customization

Both examples can be easily customized:

- **Logger Configuration**: Modify the `LogOption` to use different engines (slog/zap), formats, or output destinations
- **Middleware Behavior**: Adjust logging levels, add more fields, or modify filtering logic
- **Route Handlers**: Add your own business logic while maintaining structured logging
- **Error Handling**: Customize error responses and logging based on your needs

## Integration in Your Project

To integrate similar logging in your own project:

1. Copy the middleware functions (`GinLoggerMiddleware`, `EchoLoggerMiddleware`, etc.)
2. Adapt the field names and logging levels to match your requirements
3. Add any project-specific context or metadata
4. Configure the unified logger according to your environment needs

The middleware functions are designed to be reusable and can be easily adapted for other web frameworks following similar patterns.

## 🔄 Framework Comparison

| Feature | Gin Example | Echo Example | 
|---------|-------------|--------------|
| **Logger Engine** | slog (structured) | zap (high performance) |
| **Server Port** | :8080 | :8081 |
| **Route Logging** | URL path only | URL path + route pattern |
| **Context Tracking** | User ID via c.Set() | User ID via c.Set() |
| **Middleware Style** | Gin HandlerFunc | Echo MiddlewareFunc |
| **Error Handling** | HTTP status based | Echo HTTPError + status |
| **Response Format** | gin.H{} | map[string]interface{} |

### Key Logging Differences

**Gin Implementation:**
- Uses slog engine for structured logging
- Simpler middleware signature: `gin.HandlerFunc`
- Access via `c.Writer.Status()`, `c.ClientIP()`
- Route groups for API organization

**Echo Implementation:**
- Uses zap engine for high-performance logging  
- Route pattern logging: shows both actual path and route template
- More detailed request/response metrics (bytes in/out)
- Built-in CORS middleware integration
- Access via `c.Response().Status`, `c.RealIP()`

Both implementations demonstrate the flexibility of the unified logger - the same core logging interface works seamlessly with different engines and frameworks while maintaining consistent structured output.

## 🧪 Testing Both Frameworks

You can run both servers simultaneously to compare their behavior:

```bash
# Terminal 1 - Start comparison demo
cd example/web-frameworks
go run main.go

# Terminal 2 - Test both frameworks
curl http://localhost:8080/compare  # Gin framework info
curl http://localhost:8081/compare  # Echo framework info

# Compare the same endpoint on both frameworks
curl http://localhost:8080/test/123  # Gin version
curl http://localhost:8081/test/123  # Echo version
```

This will show how the unified logger produces consistent structured logs across different frameworks while preserving each framework's unique characteristics.