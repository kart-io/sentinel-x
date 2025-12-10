// Package etcd provides an Etcd storage client implementation.
// This package wraps the etcd clientv3 library and implements the
// storage.Client interface for consistent integration with the sentinel-x project.
package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options/etcd"
	"github.com/kart-io/sentinel-x/pkg/storage"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client wraps clientv3.Client with the storage.Client interface.
// It provides a unified interface for etcd operations while maintaining
// access to the underlying etcd client for advanced usage.
type Client struct {
	// client is the underlying etcd client
	client *clientv3.Client

	// opts stores the configuration options used to create this client
	opts *etcd.Options
}

// New creates a new Etcd client from the provided options.
// It validates the options, establishes a connection to the etcd cluster,
// and verifies connectivity with a ping operation.
//
// This is a convenience wrapper around NewWithContext that uses a default
// timeout context. For more control over the initialization timeout,
// use NewWithContext directly.
//
// Example usage:
//
//	opts := etcd.NewOptions()
//	opts.Endpoints = []string{"localhost:2379"}
//	client, err := New(opts)
//	if err != nil {
//	    log.Fatalf("failed to create etcd client: %v", err)
//	}
//	defer client.Close()
func New(opts *etcd.Options) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return NewWithContext(ctx, opts)
}

// NewWithContext creates a new Etcd client with the specified context.
// The context can be used to control the timeout for the initialization process.
// This method validates options, creates the etcd client, and verifies connectivity.
//
// Returns an error if:
//   - Options validation fails (e.g., no endpoints provided)
//   - Client creation fails (e.g., TLS configuration error)
//   - Initial connection cannot be established (verified by Ping)
//
// Example usage:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	opts := etcd.NewOptions()
//	opts.Endpoints = []string{"etcd-1:2379", "etcd-2:2379", "etcd-3:2379"}
//	opts.Username = "root"
//	opts.Password = "secret"
//
//	client, err := NewWithContext(ctx, opts)
//	if err != nil {
//	    log.Fatalf("failed to create etcd client: %v", err)
//	}
//	defer client.Close()
func NewWithContext(ctx context.Context, opts *etcd.Options) (*Client, error) {
	// Validate options
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid etcd options: %w", err)
	}

	// Build etcd client configuration
	config := clientv3.Config{
		Endpoints:   opts.Endpoints,
		DialTimeout: opts.DialTimeout,
		Username:    opts.Username,
		Password:    opts.Password,
	}

	// Configure TLS if needed (placeholder for future TLS support)
	// TODO: Add TLS configuration support based on options
	if tlsConfig := buildTLSConfig(opts); tlsConfig != nil {
		config.TLS = tlsConfig
	}

	// Create the etcd client
	cli, err := clientv3.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	client := &Client{
		client: cli,
		opts:   opts,
	}

	// Verify connectivity with a ping
	if err := client.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to ping etcd cluster: %w", err)
	}

	return client, nil
}

// Name returns the storage type name.
// This implements the storage.Client interface.
func (c *Client) Name() string {
	return "etcd"
}

// Ping checks if the connection to the etcd cluster is alive.
// It performs a lightweight Get operation to verify connectivity
// without retrieving actual data. This is suitable for health checks.
//
// The context can be used to set timeouts or cancel the operation.
// Returns an error if the cluster is unreachable or unhealthy.
func (c *Client) Ping(ctx context.Context) error {
	// Use a simple Get operation with a non-existent key to check connectivity
	// We don't care about the result, only that the cluster responds
	ctx, cancel := context.WithTimeout(ctx, c.opts.RequestTimeout)
	defer cancel()

	_, err := c.client.Get(ctx, "__sentinel_x_ping__")
	if err != nil {
		return fmt.Errorf("etcd ping failed: %w", err)
	}
	return nil
}

// Close closes the etcd client connection gracefully.
// This method releases all resources including connection pools.
// It is safe to call Close multiple times.
//
// This implements the storage.Client interface.
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Health returns a HealthChecker function for this client.
// The returned function can be called to perform health checks
// without direct access to the client.
//
// This implements the storage.Client interface.
func (c *Client) Health() storage.HealthChecker {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return c.Ping(ctx)
	}
}

// Raw returns the underlying clientv3.Client for advanced operations.
// This allows direct access to etcd-specific features not exposed
// through the storage.Client interface.
//
// Example usage:
//
//	rawClient := client.Raw()
//	resp, err := rawClient.Get(ctx, "key")
//	if err != nil {
//	    log.Printf("get failed: %v", err)
//	}
func (c *Client) Raw() *clientv3.Client {
	return c.client
}

// KV returns the KV interface for key-value operations.
// This is a convenience method to access the KV API directly.
//
// Example usage:
//
//	kv := client.KV()
//	_, err := kv.Put(ctx, "key", "value")
func (c *Client) KV() clientv3.KV {
	return c.client.KV
}

// Lease returns the Lease interface for lease operations.
// This is a convenience method to access the Lease API directly.
//
// Example usage:
//
//	lease := client.Lease()
//	resp, err := lease.Grant(ctx, 60)
func (c *Client) Lease() clientv3.Lease {
	return c.client.Lease
}

// validateOptions validates the etcd options.
// Returns an error if the options are invalid.
func validateOptions(opts *etcd.Options) error {
	if opts == nil {
		return fmt.Errorf("options cannot be nil")
	}

	if len(opts.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint is required")
	}

	if opts.DialTimeout <= 0 {
		return fmt.Errorf("dial timeout must be positive")
	}

	if opts.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	return nil
}

// buildTLSConfig builds a TLS configuration from the options.
// Returns nil if TLS is not configured.
// TODO: Implement TLS configuration based on options
func buildTLSConfig(opts *etcd.Options) *tls.Config {
	// Placeholder for future TLS support
	// This would read certificate paths from options and build TLS config
	return nil
}
