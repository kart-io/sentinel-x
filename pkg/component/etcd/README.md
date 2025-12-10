# Etcd Storage Client

This package provides an Etcd v3 storage client implementation for the sentinel-x project. It wraps the official etcd clientv3 library and implements the `storage.Client` interface for consistent integration with other storage backends.

## Features

- **Unified Interface**: Implements `storage.Client` interface for consistent storage access
- **Authentication Support**: Username/password authentication
- **Configurable Timeouts**: Separate dial and request timeout settings
- **Health Checking**: Comprehensive health checks with latency measurement and cluster status
- **Factory Pattern**: Factory-based client creation for dependency injection
- **Raw Client Access**: Direct access to underlying etcd client for advanced operations
- **Convenience Methods**: Easy access to KV and Lease interfaces

## Installation

The etcd client dependency is already included in the project:

```bash
go get go.etcd.io/etcd/client/v3@v3.5.17
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/kart-io/sentinel-x/pkg/etcd"
    etcdopts "github.com/kart-io/sentinel-x/pkg/options/etcd"
)

func main() {
    // Create options with default values
    opts := etcdopts.NewOptions()
    opts.Endpoints = []string{"localhost:2379"}

    // Create a new etcd client
    client, err := etcd.New(opts)
    if err != nil {
        log.Fatalf("failed to create etcd client: %v", err)
    }
    defer client.Close()

    // Verify connectivity
    ctx := context.Background()
    if err := client.Ping(ctx); err != nil {
        log.Printf("ping failed: %v", err)
        return
    }

    log.Println("Successfully connected to etcd")
}
```

### Using Factory Pattern

```go
opts := etcdopts.NewOptions()
opts.Endpoints = []string{"etcd-1:2379", "etcd-2:2379", "etcd-3:2379"}

factory := etcd.NewFactory(opts)

ctx := context.Background()
client, err := factory.Create(ctx)
if err != nil {
    log.Fatalf("failed to create client: %v", err)
}
defer client.Close()
```

### Authentication

```go
opts := etcdopts.NewOptions()
opts.Endpoints = []string{"localhost:2379"}
opts.Username = "root"
opts.Password = "secret"

client, err := etcd.New(opts)
if err != nil {
    log.Fatalf("failed to create authenticated client: %v", err)
}
defer client.Close()
```

## Configuration

### Options Structure

The `Options` struct from `pkg/options/etcd` provides the following configuration fields:

```go
type Options struct {
    Endpoints      []string      // Etcd cluster endpoints
    Username       string        // Authentication username
    Password       string        // Authentication password
    DialTimeout    time.Duration // Timeout for establishing connections
    RequestTimeout time.Duration // Timeout for requests
    LeaseTTL       int64         // Default lease TTL in seconds
}
```

### Default Values

```go
opts := etcdopts.NewOptions()
// Endpoints:      ["127.0.0.1:2379"]
// Username:       ""
// Password:       ""
// DialTimeout:    5 seconds
// RequestTimeout: 2 seconds
// LeaseTTL:       60 seconds
```

### Command-Line Flags

Options can be configured via command-line flags:

```go
import (
    "github.com/spf13/cobra"
    etcdopts "github.com/kart-io/sentinel-x/pkg/options/etcd"
)

cmd := &cobra.Command{...}
opts := etcdopts.NewOptions()
opts.AddFlags(cmd.Flags())
```

Available flags:
- `--etcd.endpoints`: Etcd endpoints (default: ["127.0.0.1:2379"])
- `--etcd.username`: Etcd username
- `--etcd.password`: Etcd password
- `--etcd.dial-timeout`: Etcd dial timeout (default: 5s)
- `--etcd.request-timeout`: Etcd request timeout (default: 2s)
- `--etcd.lease-ttl`: Etcd lease TTL (default: 60)

## Health Checking

### Comprehensive Health Check

```go
status := client.CheckHealth(ctx)
if status.Healthy {
    log.Printf("etcd is healthy (latency: %v)", status.Latency)
} else {
    log.Printf("etcd is unhealthy: %v (latency: %v)", status.Error, status.Latency)
}
```

### Using Health Checker Interface

```go
// Get a HealthChecker function
checker := client.Health()

// Call it independently
if err := checker(); err != nil {
    log.Printf("health check failed: %v", err)
}
```

## Advanced Usage

### Raw Client Access

Access the underlying etcd client for etcd-specific operations:

```go
rawClient := client.Raw()

// Use etcd-specific features
resp, err := rawClient.Get(ctx, "/config/key")
if err != nil {
    log.Printf("get failed: %v", err)
}

// Use with options
resp, err = rawClient.Get(ctx, "/config/", clientv3.WithPrefix())
```

### KV Interface

Direct access to key-value operations:

```go
kv := client.KV()

// Put a key-value pair
_, err := kv.Put(ctx, "key", "value")
if err != nil {
    log.Printf("put failed: %v", err)
}

// Get a value
resp, err := kv.Get(ctx, "key")
if err != nil {
    log.Printf("get failed: %v", err)
}
```

### Lease Interface

Work with leases for time-bound keys:

```go
lease := client.Lease()

// Grant a 60-second lease
resp, err := lease.Grant(ctx, 60)
if err != nil {
    log.Printf("lease grant failed: %v", err)
}

log.Printf("Lease ID: %d, TTL: %d seconds", resp.ID, resp.TTL)

// Use the lease with KV operations
// (requires using Raw() client for WithLease option)
```

### Custom Context Timeout

```go
// Create a client with custom initialization timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

client, err := etcd.NewWithContext(ctx, opts)
if err != nil {
    log.Fatalf("failed to create client: %v", err)
}
defer client.Close()
```

## Integration with storage.Client Interface

The etcd client implements the `storage.Client` interface:

```go
var storageClient storage.Client = client

// Use storage.Client interface methods
fmt.Println(storageClient.Name())  // Output: "etcd"

err := storageClient.Ping(ctx)
if err != nil {
    log.Printf("ping failed: %v", err)
}

checker := storageClient.Health()
err = checker()
```

## Error Handling

All methods return wrapped errors with context:

```go
client, err := etcd.New(opts)
if err != nil {
    // Possible errors:
    // - "invalid etcd options: at least one endpoint is required"
    // - "invalid etcd options: dial timeout must be positive"
    // - "failed to create etcd client: ..."
    // - "failed to ping etcd cluster: ..."
    log.Fatalf("initialization failed: %v", err)
}
```

## Best Practices

1. **Always Close Clients**: Use `defer client.Close()` after creating clients
2. **Use Contexts**: Always pass contexts for timeout control
3. **Validate Options**: Options are validated automatically during client creation
4. **Health Checks**: Perform health checks before critical operations
5. **Factory for DI**: Use Factory pattern for dependency injection and testing
6. **Error Handling**: Always check and handle errors appropriately

## Testing

The package includes comprehensive example tests in `example_test.go`. To run the examples:

```bash
# Run all tests (requires running etcd instance)
go test ./pkg/etcd/...

# Run specific example
go test ./pkg/etcd/... -run ExampleNew

# Run with verbose output
go test -v ./pkg/etcd/...
```

## Architecture

### Package Structure

```
pkg/etcd/
├── client.go       # Client implementation
├── health.go       # Health checking logic
├── factory.go      # Factory implementation
├── doc.go          # Package documentation
├── example_test.go # Usage examples
└── README.md       # This file
```

### Dependencies

- `go.etcd.io/etcd/client/v3` v3.5.17 - Official etcd client library
- `github.com/kart-io/sentinel-x/pkg/options/etcd` - Configuration options
- `github.com/kart-io/sentinel-x/pkg/storage` - Storage interface

## Future Enhancements

- [ ] TLS/SSL support for secure connections
- [ ] Connection pooling optimization
- [ ] Automatic reconnection with backoff
- [ ] Metrics and monitoring integration
- [ ] Distributed lock implementation
- [ ] Watch API wrapper
- [ ] Transaction support
- [ ] Snapshot and restore utilities

## Examples

See `example_test.go` for comprehensive usage examples including:

- Basic client creation
- Factory pattern usage
- Health checking
- Authentication
- Raw client access
- KV operations
- Lease management

## Contributing

When contributing to this package:

1. Follow the existing code style and patterns
2. Add tests for new functionality
3. Update documentation and examples
4. Ensure `go vet` and `go build` pass
5. Follow the project's CLAUDE.md guidelines

## License

This package is part of the sentinel-x project and follows the same license.
