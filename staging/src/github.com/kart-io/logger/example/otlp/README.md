# OTLP Integration Example

This example demonstrates OpenTelemetry Protocol (OTLP) integration with the unified logger library, specifically testing with endpoint `127.0.0.1:4327`.

## Overview

The OTLP example shows how to configure and use the logger with OpenTelemetry collectors for distributed tracing and observability. It includes both gRPC and HTTP protocol configurations, error handling, and tracing context integration.

## Features Demonstrated

### 1. OTLP Protocol Support
- **gRPC Protocol**: Standard OTLP over gRPC
- **HTTP Protocol**: OTLP over HTTP/protobuf
- **Endpoint Configuration**: Testing with `127.0.0.1:4327`
- **Timeout Configuration**: Custom timeout settings

### 2. Error Handling
- **Invalid Endpoint**: Graceful fallback to stdout
- **Connection Failures**: Timeout handling
- **Collector Unavailable**: Automatic fallback behavior

### 3. Distributed Tracing
- **Trace Context**: Integration with OpenTelemetry trace IDs
- **Span Integration**: Parent-child span relationships
- **Service Metadata**: Service name and version tracking

## Configuration Options

```go
&option.LogOption{
    Engine:      "zap",           // or "slog"
    Level:       "INFO",
    Format:      "json",
    OutputPaths: []string{"stdout"},
    OTLP: &option.OTLPOption{
        Endpoint: "127.0.0.1:4327",    // Target OTLP collector
        Protocol: "grpc",              // or "http"
        Timeout:  10 * time.Second,    // Connection timeout
    },
}
```

## Running the Example

```bash
cd example/otlp
go run main.go
```

## Example Output

### OTLP with gRPC Protocol
```json
{
  "level": "INFO",
  "timestamp": "2025-08-29T16:03:47.09498053+08:00",
  "caller": "otlp/main.go:60",
  "message": "OTLP gRPC test message",
  "engine": "zap",
  "protocol": "grpc",
  "endpoint": "127.0.0.1:4327",
  "test_id": "grpc_001"
}
```

### OTLP with HTTP Protocol
```json
{
  "time": "2025-08-29T16:03:47.095069038+08:00",
  "level": "INFO",
  "msg": "OTLP HTTP test message",
  "engine": "slog",
  "protocol": "http",
  "endpoint": "http://127.0.0.1:4327",
  "test_id": "http_001"
}
```

### Distributed Tracing Context
```json
{
  "level": "INFO",
  "timestamp": "2025-08-29T16:03:47.09519419+08:00",
  "caller": "otlp/main.go:260",
  "message": "OTLP traced operation started",
  "engine": "zap",
  "trace_id": "trace_abc123def456789",
  "span_id": "span_012345678901234",
  "parent_span_id": "span_parent123456",
  "service_name": "otlp-test-service",
  "service_version": "1.0.0",
  "operation": "process_user_request"
}
```

## Test Scenarios

### 1. gRPC Protocol Test
- **Endpoint**: `127.0.0.1:4327`
- **Protocol**: `grpc`
- **Purpose**: Test standard OTLP gRPC integration
- **Features**: Metrics logging, application telemetry

### 2. HTTP Protocol Test
- **Endpoint**: `http://127.0.0.1:4327`
- **Protocol**: `http`
- **Purpose**: Test OTLP HTTP/protobuf integration
- **Features**: Warning logs, rate limiting scenarios

### 3. Custom Timeout Test
- **Timeout**: `2 seconds`
- **Purpose**: Test timeout behavior
- **Features**: Debug logs, error scenarios

### 4. Error Handling Tests
- **Invalid Endpoint**: Test fallback behavior
- **No Collector**: Test connection failure handling
- **Network Issues**: Test timeout scenarios

### 5. Distributed Tracing Test
- **Trace ID**: `trace_abc123def456789`
- **Span ID**: `span_012345678901234`
- **Service**: `otlp-test-service v1.0.0`
- **Purpose**: Test OpenTelemetry trace context integration

## Expected Behavior

### With OTLP Collector Running
When an OTLP collector is running at `127.0.0.1:4327`:
- Logs are sent to both stdout and the collector
- Trace context is properly propagated
- Metrics are exported to the observability backend

### Without OTLP Collector
When no collector is available:
- Logger creation succeeds
- Logs continue to appear on stdout
- OTLP export attempts may timeout gracefully
- Application continues to function normally

## Integration with Observability Stack

This example is designed to work with:
- **Jaeger**: Distributed tracing
- **OpenTelemetry Collector**: Log aggregation
- **Prometheus**: Metrics collection
- **Grafana**: Visualization

To set up a complete observability stack:

1. **Run OTLP Collector**:
```bash
# Docker example
docker run -p 4327:4317 \
  otel/opentelemetry-collector:latest \
  --config=/etc/otel-collector-config.yaml
```

2. **Configure Collector** to forward to your observability backend

3. **Run the Example**:
```bash
go run main.go
```

## Key Features

- ✅ **Dual Protocol Support**: Both gRPC and HTTP
- ✅ **Flexible Configuration**: Timeout and endpoint customization
- ✅ **Error Resilience**: Graceful fallback on failures
- ✅ **Trace Integration**: Full OpenTelemetry compatibility
- ✅ **Engine Agnostic**: Works with both Zap and Slog
- ✅ **Production Ready**: Handles real-world scenarios

## Notes

- The example will work without an OTLP collector running
- Logs will always appear on stdout regardless of OTLP status
- Real production deployments should use proper TLS configuration
- Consider using environment variables for endpoint configuration
- Monitor OTLP export success rates in production environments