# OpenTelemetry Distributed Tracing

This directory contains the OpenTelemetry distributed tracing implementation for Sentinel-X.

## Features

- W3C Trace Context propagation (traceparent header)
- Support for multiple exporters (OTLP gRPC, OTLP HTTP, stdout, noop)
- Configurable sampling strategies (always on, always off, ratio-based, parent-based)
- Batch span processing with configurable parameters
- HTTP middleware with automatic span creation and attribute injection
- gRPC interceptors for both unary and streaming RPCs
- Rich semantic conventions for HTTP and gRPC attributes
- Helper functions for span manipulation and context propagation

## Architecture

### Core Components

1. **Provider** (`pkg/infra/tracing/provider.go`)
   - Manages the OpenTelemetry tracer provider lifecycle
   - Configures exporters, samplers, and batch processors
   - Provides graceful shutdown and flush capabilities

2. **Options** (`pkg/infra/tracing/options.go`)
   - Configuration for tracing behavior
   - Exporter settings (OTLP, stdout, noop)
   - Sampling configuration
   - Batch processor tuning

3. **Context Helpers** (`pkg/infra/tracing/context.go`)
   - Convenience functions for span creation and manipulation
   - Attribute helpers with semantic conventions
   - Error recording and status management

4. **HTTP Middleware** (`pkg/infra/middleware/tracing.go`)
   - Automatic span creation for HTTP requests
   - W3C Trace Context extraction and injection
   - HTTP semantic attributes (method, URL, status code, user agent)
   - Configurable path skipping

5. **gRPC Interceptors** (`pkg/infra/middleware/grpc/tracing.go`)
   - Unary and streaming server interceptors
   - Unary and streaming client interceptors
   - gRPC metadata propagation
   - RPC semantic attributes (service, method, status code)

## Usage

### 1. Initialize Tracing Provider

```go
import (
    "context"
    "github.com/kart-io/sentinel-x/pkg/infra/tracing"
)

// Create options
opts := tracing.NewOptions()
opts.Enabled = true
opts.ServiceName = "my-service"
opts.ServiceVersion = "1.0.0"
opts.Environment = "production"
opts.ExporterType = tracing.ExporterOTLPGRPC
opts.Endpoint = "localhost:4317"
opts.SamplerType = tracing.SamplerParentBased
opts.SamplerRatio = 1.0

// Initialize provider
provider, err := tracing.NewProvider(opts)
if err != nil {
    log.Fatal(err)
}
defer provider.Shutdown(context.Background())
```

### 2. HTTP Middleware

```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
    "github.com/kart-io/sentinel-x/pkg/infra/server/transport/http"
)

// Create HTTP server with tracing middleware
server := http.NewServer(
    http.WithAddr(":8080"),
    http.WithMiddleware(
        middleware.Tracing(
            middleware.WithTracingSkipPaths([]string{"/health", "/metrics"}),
            middleware.WithTracingSkipPathPrefixes([]string{"/debug", "/internal"}),
        ),
    ),
)
```

### 3. gRPC Interceptors

```go
import (
    grpcmw "github.com/kart-io/sentinel-x/pkg/infra/middleware/grpc"
    "google.golang.org/grpc"
)

// Create gRPC server with tracing interceptors
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcmw.ChainUnaryInterceptors(
        grpcmw.UnaryTracingInterceptor(),
        // other interceptors...
    )),
    grpc.StreamInterceptor(grpcmw.ChainStreamInterceptors(
        grpcmw.StreamTracingInterceptor(),
        // other interceptors...
    )),
)

// Create gRPC client with tracing interceptors
conn, err := grpc.Dial(
    "localhost:9090",
    grpc.WithUnaryInterceptor(grpcmw.UnaryClientTracingInterceptor()),
    grpc.WithStreamInterceptor(grpcmw.StreamClientTracingInterceptor()),
)
```

### 4. Manual Span Creation

```go
import (
    "context"
    "github.com/kart-io/sentinel-x/pkg/infra/tracing"
)

func doWork(ctx context.Context) error {
    // Start a new span
    ctx, span := tracing.StartSpan(ctx, "my-service", "do-work")
    defer span.End()

    // Add attributes
    tracing.AddSpanAttributes(ctx,
        tracing.String("user.id", "123"),
        tracing.String("operation", "process"),
    )

    // Do work...
    if err := performTask(); err != nil {
        // Record error
        tracing.RecordError(ctx, err)
        return err
    }

    // Mark as successful
    tracing.SetSpanOK(ctx)
    return nil
}
```

### 5. Extract Trace Information

```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
)

// In an HTTP handler
func handler(ctx transport.Context) {
    // Extract trace ID for logging or response headers
    traceID := middleware.ExtractTraceID(ctx)
    spanID := middleware.ExtractSpanID(ctx)

    // Add to response headers
    ctx.SetHeader("X-Trace-ID", traceID)
    ctx.SetHeader("X-Span-ID", spanID)
}
```

## Configuration

### Exporter Types

- `otlp_grpc`: Export to OTLP collector via gRPC (recommended for production)
- `otlp_http`: Export to OTLP collector via HTTP
- `stdout`: Print spans to stdout (useful for development)
- `noop`: No-op exporter (disable tracing)

### Sampling Strategies

- `always_on`: Sample all traces
- `always_off`: Never sample traces
- `ratio`: Sample based on a ratio (0.0 to 1.0)
- `parent_based`: Use parent span's sampling decision

### Batch Processor Settings

- `batch-timeout`: Maximum time to wait before exporting (default: 5s)
- `batch-max-size`: Maximum spans per batch (default: 512)
- `export-timeout`: Maximum time for export operation (default: 30s)
- `max-queue-size`: Maximum queue size (default: 2048)

## Command-Line Flags

All configuration can be set via command-line flags:

```bash
--tracing.enabled=true
--tracing.service-name=my-service
--tracing.service-version=1.0.0
--tracing.environment=production
--tracing.exporter-type=otlp_grpc
--tracing.endpoint=localhost:4317
--tracing.insecure=true
--tracing.sampler-type=parent_based
--tracing.sampler-ratio=1.0
--tracing.batch-timeout=5s
--tracing.batch-max-size=512
--tracing.export-timeout=30s
--tracing.max-queue-size=2048
```

## Semantic Attributes

### HTTP Attributes

- `http.method`: HTTP method (GET, POST, etc.)
- `http.url`: Full URL
- `http.target`: Path and query string
- `http.scheme`: URL scheme (http, https)
- `http.status_code`: HTTP status code
- `user_agent.original`: User-Agent header
- `http.client_ip`: Client IP address
- `http.request_id`: Request ID from header

### gRPC Attributes

- `rpc.system`: "grpc"
- `rpc.service`: gRPC service name
- `rpc.method`: gRPC method name
- `rpc.grpc.status_code`: gRPC status code
- `rpc.grpc.stream`: Whether the RPC is a stream
- `rpc.grpc.stream_type`: Stream type (client, server, bidi)

### Custom Attributes

```go
// Common patterns
tracing.AddSpanAttributes(ctx,
    tracing.String("user.id", userID),
    tracing.String("user.email", email),
    tracing.String("tenant.id", tenantID),
    tracing.String("correlation.id", correlationID),
)
```

## Integration with Observability Backends

### Jaeger

```bash
# Start Jaeger
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest

# Configure Sentinel-X
--tracing.enabled=true
--tracing.exporter-type=otlp_grpc
--tracing.endpoint=localhost:4317
--tracing.insecure=true
```

### Grafana Tempo

```bash
# Configure Sentinel-X
--tracing.enabled=true
--tracing.exporter-type=otlp_grpc
--tracing.endpoint=tempo:4317
```

### New Relic

```bash
# Configure Sentinel-X
--tracing.enabled=true
--tracing.exporter-type=otlp_grpc
--tracing.endpoint=otlp.nr-data.net:4317
```

## Testing

```bash
# Run all tracing tests
go test ./pkg/infra/tracing/... -v

# Run middleware tests
go test ./pkg/infra/middleware/tracing_test.go -v

# Run with coverage
go test ./pkg/infra/tracing/... -cover
```

## Performance Considerations

1. **Sampling**: Use parent-based sampling with a ratio for high-throughput services
2. **Batch Processing**: Tune batch size and timeout based on your workload
3. **Skip Paths**: Configure middleware to skip tracing for health checks and metrics endpoints
4. **Resource Attributes**: Minimize custom resource attributes to reduce overhead

## Best Practices

1. **Always use defer**: `defer span.End()` immediately after creating a span
2. **Add meaningful attributes**: Include user ID, request ID, and operation-specific data
3. **Record errors**: Always record errors in spans for debugging
4. **Use semantic conventions**: Prefer standard attributes over custom ones
5. **Propagate context**: Always pass `context.Context` through your call stack
6. **Test with stdout exporter**: Use stdout exporter during development
7. **Monitor exporter health**: Watch for export errors in production

## Troubleshooting

### Spans not appearing

1. Check that tracing is enabled: `--tracing.enabled=true`
2. Verify exporter endpoint is reachable
3. Check sampling configuration (might be sampling out all traces)
4. Look for export errors in logs

### High memory usage

1. Reduce batch size: `--tracing.batch-max-size=256`
2. Reduce queue size: `--tracing.max-queue-size=1024`
3. Increase batch timeout: `--tracing.batch-timeout=2s`
4. Lower sampling ratio: `--tracing.sampler-ratio=0.1`

### Missing trace context

1. Ensure middleware is installed early in the chain
2. Verify W3C Trace Context headers are being propagated
3. Check that context is being passed through all function calls
4. Use context helpers to ensure proper propagation

## References

- [OpenTelemetry Specification](https://opentelemetry.io/docs/specs/otel/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry Go SDK](https://pkg.go.dev/go.opentelemetry.io/otel)
- [Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
