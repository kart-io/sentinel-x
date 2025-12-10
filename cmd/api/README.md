# Sentinel-X API Server

## Overview

The Sentinel-X API Server is the main application server for the Sentinel-X platform. It provides both HTTP REST API and gRPC endpoints with comprehensive middleware support, authentication, and authorization.

## Features

- **Dual Protocol Support**: Both HTTP (REST) and gRPC in a single server
- **Flexible HTTP Adapters**: Support for Gin and Echo frameworks
- **JWT Authentication**: Token-based authentication with refresh tokens
- **RBAC Authorization**: Role-based access control with Casbin
- **Database Integration**: MySQL with GORM ORM
- **Caching**: Redis integration for distributed caching
- **Comprehensive Middleware**:
  - Panic recovery
  - Request ID tracking
  - Structured logging
  - CORS support
  - Request timeout
  - Health checks
  - Prometheus metrics
  - Pprof profiling (optional)
- **Configuration Management**: Multiple configuration sources (files, environment, flags)
- **Graceful Shutdown**: Proper signal handling for SIGINT/SIGTERM
- **Version Management**: Build-time version injection

## Architecture

The implementation follows a layered architecture:

```
cmd/api/
├── main.go           # Entry point
└── app/
    ├── app.go        # Application logic
    └── options.go    # Configuration options
```

The server uses:
- `pkg/app`: Application framework with Cobra/Viper integration
- `pkg/server`: Server manager for HTTP/gRPC
- `pkg/middleware`: HTTP middleware stack
- `pkg/auth`: JWT authentication
- `pkg/authz`: RBAC authorization

## Building

### Basic Build

```bash
go build -o sentinel-api ./cmd/api
```

### Build with Version Injection

```bash
go build -ldflags "
  -X 'github.com/kart-io/version.serviceName=sentinel-api'
  -X 'github.com/kart-io/version.gitVersion=$(git describe --tags --always)'
  -X 'github.com/kart-io/version.gitCommit=$(git rev-parse HEAD)'
  -X 'github.com/kart-io/version.gitBranch=$(git branch --show-current)'
  -X 'github.com/kart-io/version.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'
" -o sentinel-api ./cmd/api
```

## Running

### With Default Configuration

```bash
./sentinel-api
```

The server will:
- Start HTTP server on `:8100`
- Start gRPC server on `:9100`
- Look for config file in current directory, `./configs`, `~/.sentinel-api`, or `/etc/sentinel-api`

### With Config File

```bash
./sentinel-api -c /path/to/config.yaml
```

### With Command-Line Flags

```bash
# HTTP only mode
./sentinel-api --server.mode=http

# Custom addresses
./sentinel-api --http.addr=:8080 --grpc.addr=:9090

# Debug logging
./sentinel-api --log.level=debug

# Enable CORS
./sentinel-api --middleware.disable-cors=false

# Enable profiling
./sentinel-api --middleware.disable-pprof=false
```

### With Environment Variables

Environment variables use the prefix `SENTINEL_API_`:

```bash
export SENTINEL_API_SERVER_MODE=http
export SENTINEL_API_HTTP_ADDR=:8080
export SENTINEL_API_LOG_LEVEL=debug
export MYSQL_PASSWORD=secretpassword
export REDIS_PASSWORD=redispassword

./sentinel-api
```

## Configuration

Configuration can be provided through:
1. Command-line flags (highest priority)
2. Environment variables (prefix: `SENTINEL_API_`)
3. Configuration file (YAML)
4. Default values (lowest priority)

### Example Configuration File

See `configs/sentinel-api.yaml` for a complete example configuration file.

### Configuration Priority

The configuration system uses the following priority (highest to lowest):
1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

## Endpoints

### Health Checks

- `GET /health` - Overall health status
- `GET /live` - Liveness probe (Kubernetes)
- `GET /ready` - Readiness probe (Kubernetes)

### Metrics

- `GET /metrics` - Prometheus metrics

### Profiling (disabled by default)

- `GET /debug/pprof/` - Index page
- `GET /debug/pprof/cmdline` - Command line
- `GET /debug/pprof/profile` - CPU profile
- `GET /debug/pprof/symbol` - Symbol lookup
- `GET /debug/pprof/trace` - Execution trace

## Database Setup

### MySQL

Create the database:

```sql
CREATE DATABASE sentinel CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'sentinel'@'localhost' IDENTIFIED BY 'sentinel123';
GRANT ALL PRIVILEGES ON sentinel.* TO 'sentinel'@'localhost';
FLUSH PRIVILEGES;
```

Update configuration:

```yaml
mysql:
  host: "localhost"
  port: 3306
  username: "sentinel"
  password: "sentinel123"
  database: "sentinel"
```

Or use environment variable for password:

```bash
export MYSQL_PASSWORD=sentinel123
```

### Redis

Start Redis:

```bash
docker run -d -p 6379:6379 redis:latest
```

Or configure in config file:

```yaml
redis:
  host: "localhost"
  port: 6379
  password: ""
  database: 0
```

## Development

### Adding New Services

1. Create service in appropriate package (e.g., `internal/service/userservice`)
2. Create HTTP and gRPC handlers
3. Register service in `cmd/api/app/app.go`:

```go
// In Run() function
userSvc := userservice.NewService(db, rdb)
userHTTPHandler := handler.NewUserHTTPHandler(userSvc)
userGRPCHandler := handler.NewUserGRPCHandler(userSvc)

_ = mgr.RegisterService(
    userSvc,
    userHTTPHandler,
    &transport.GRPCServiceDesc{
        ServiceDesc: &apiv1.UserService_ServiceDesc,
        ServiceImpl: userGRPCHandler,
    },
)
```

### Adding Health Checks

In `cmd/api/app/app.go`, add health checkers in `configureHealth()`:

```go
healthMgr.RegisterChecker("custom", func() error {
    // Your health check logic
    return nil
})
```

### Configuring RBAC Roles

In `cmd/api/app/app.go`, modify `initAuth()`:

```go
// Add custom role
_ = rbacAuthz.AddRole("editor",
    authz.NewPermission("resource", "read"),
    authz.NewPermission("resource", "write"),
)
```

## Monitoring

### Health Checks

```bash
curl http://localhost:8100/health
```

### Metrics

```bash
curl http://localhost:8100/metrics
```

### Profiling

Enable profiling and access:

```bash
# Start with profiling enabled
./sentinel-api --middleware.disable-pprof=false

# Access pprof
go tool pprof http://localhost:8100/debug/pprof/heap
```

## Deployment

### Docker

Create a Dockerfile:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o sentinel-api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/sentinel-api .
COPY --from=builder /app/configs/sentinel-api.yaml /etc/sentinel-x/api.yaml
EXPOSE 8100 9100
CMD ["./sentinel-api", "-c", "/etc/sentinel-x/api.yaml"]
```

Build and run:

```bash
docker build -t sentinel-api .
docker run -p 8100:8100 -p 9100:9100 sentinel-api
```

### Kubernetes

Create deployment and service manifests with proper health check configuration using the `/live` and `/ready` endpoints.

## Troubleshooting

### Server won't start

Check logs for errors. Common issues:
- Port already in use: Change `http.addr` or `grpc.addr`
- Database connection failed: Verify MySQL credentials and connectivity
- Redis connection failed: Verify Redis is running and accessible

### Configuration not loading

Ensure config file:
- Is in a supported location (`.`, `./configs`, `~/.sentinel-api`, `/etc/sentinel-api`)
- Is named `sentinel-api.yaml`
- Has valid YAML syntax
- Or specify explicit path with `-c` flag

### Authentication issues

- Check JWT secret is set correctly
- Verify token expiration settings
- Ensure auth middleware is enabled (`disable-auth: false`)

## License

See the main project LICENSE file.
