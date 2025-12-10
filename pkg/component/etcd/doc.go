// Package etcd provides an Etcd v3 storage client implementation for the sentinel-x project.
//
// This package wraps the official etcd clientv3 library and implements the storage.Client
// interface, enabling consistent integration with other storage backends in the system.
//
// # Features
//
//   - Implements storage.Client interface for unified storage access
//   - Supports authentication with username/password
//   - Configurable timeouts for dial and request operations
//   - Comprehensive health checking with latency measurement
//   - Factory pattern for client creation
//   - Direct access to underlying etcd client for advanced operations
//
// # Basic Usage
//
// Creating a client with default options:
//
//	import (
//	)
//
//	func main() {
//	    opts := etcdopts.NewOptions()
//	    opts.Endpoints = []string{"localhost:2379"}
//
//	    client, err := New(opts)
//	    if err != nil {
//	        log.Fatalf("failed to create etcd client: %v", err)
//	    }
//	    defer client.Close()
//
//	    // Use the client
//	    ctx := context.Background()
//	    if err := client.Ping(ctx); err != nil {
//	        log.Printf("ping failed: %v", err)
//	    }
//	}
//
// # Using with Factory
//
// For dependency injection and testing:
//
//	opts := etcdopts.NewOptions()
//	opts.Endpoints = []string{"etcd-1:2379", "etcd-2:2379", "etcd-3:2379"}
//
//	factory := NewFactory(opts)
//
//	ctx := context.Background()
//	client, err := factory.Create(ctx)
//	if err != nil {
//	    log.Fatalf("failed to create client: %v", err)
//	}
//	defer client.Close()
//
// # Health Checking
//
// Comprehensive health checks with detailed status:
//
//	status := client.CheckHealth(ctx)
//	if !status.Healthy {
//	    log.Printf("etcd unhealthy: %v (latency: %v)", status.Error, status.Latency)
//	} else {
//	    log.Printf("etcd healthy (latency: %v)", status.Latency)
//	}
//
// Simple boolean health check:
//
//	if !client.IsHealthy(ctx) {
//	    log.Println("etcd cluster is unhealthy")
//	}
//
// # Advanced Usage
//
// Access the underlying etcd client for etcd-specific operations:
//
//	rawClient := client.Raw()
//
//	// Use etcd-specific features
//	resp, err := rawClient.Get(ctx, "/config/key")
//	if err != nil {
//	    log.Printf("get failed: %v", err)
//	}
//
//	// Work with KV interface
//	kv := client.KV()
//	_, err = kv.Put(ctx, "key", "value")
//
//	// Work with Lease interface
//	lease := client.Lease()
//	resp, err := lease.Grant(ctx, 60)
//
// # Authentication
//
// Configure authentication in options:
//
//	opts := etcdopts.NewOptions()
//	opts.Endpoints = []string{"localhost:2379"}
//	opts.Username = "root"
//	opts.Password = "secret"
//
//	client, err := New(opts)
//	if err != nil {
//	    log.Fatalf("failed to create authenticated client: %v", err)
//	}
//	defer client.Close()
//
// # Configuration
//
// Options can be configured programmatically or via command-line flags:
//
//	opts := etcdopts.NewOptions()
//
//	// Programmatic configuration
//	opts.Endpoints = []string{"etcd-1:2379", "etcd-2:2379"}
//	opts.DialTimeout = 10 * time.Second
//	opts.RequestTimeout = 5 * time.Second
//	opts.LeaseTTL = 60
//
//	// Or use with cobra/pflag
//	cmd := &cobra.Command{...}
//	opts.AddFlags(cmd.Flags())
//
// # Integration with storage.Client
//
// The client implements the storage.Client interface:
//
//	var storageClient storage.Client = client
//
//	// Use storage.Client interface methods
//	fmt.Println(storageClient.Name())  // Output: "etcd"
//	err := storageClient.Ping(ctx)
//	checker := storageClient.Health()
//
// # Error Handling
//
// All methods return wrapped errors for better error context:
//
//	client, err := New(opts)
//	if err != nil {
//	    // Error will be wrapped with context like:
//	    // "invalid etcd options: at least one endpoint is required"
//	    // "failed to create etcd client: ..."
//	    // "failed to ping etcd cluster: ..."
//	    log.Fatalf("initialization failed: %v", err)
//	}
package etcd
