// Package redis provides Redis database implementation.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
	options "github.com/kart-io/sentinel-x/pkg/options/redis"
	goredis "github.com/redis/go-redis/v9"
)

// Client wraps Client with storage.Client interface implementation.
// It provides a unified Redis client that implements the standard storage
// interface while exposing the underlying go-redis client for advanced usage.
//
// Example usage:
//
//	opts := options.NewOptions()
//	opts.Host = "localhost"
//	opts.Port = 6379
//
//	client, err := New(opts)
//	if err != nil {
//	    log.Fatalf("failed to create Redis client: %v", err)
//	}
//	defer client.Close()
//
//	// Use redis client directly
//	rdb := client.Client()
//	err = rdb.Set(ctx, "key", "value", 0).Err()
type Client struct {
	client *goredis.Client
	opts   *options.Options
}

// Compile-time check that Client implements storage.Client.
var _ storage.Client = (*Client)(nil)

// New creates a new Redis client from the provided options.
// It validates the options, establishes a connection, and verifies connectivity.
//
// Returns an error if:
// - Options validation fails
// - Connection to Redis server fails
// - Initial ping fails
func New(opts *options.Options) (*Client, error) {
	return NewWithContext(context.Background(), opts)
}

// NewWithContext creates a new Redis client with context support.
// This allows for timeout control during the connection establishment phase.
//
// The context is used for:
// - Initial ping verification
//
// Returns an error if:
// - Options are nil
// - Options validation fails
// - Initial ping fails
func NewWithContext(ctx context.Context, opts *options.Options) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("redis options cannot be nil")
	}

	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid redis options: %w", err)
	}

	// Build Redis options
	redisOptions := &goredis.Options{
		Addr:         fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		Password:     opts.Password,
		DB:           opts.Database,
		MaxRetries:   opts.MaxRetries,
		PoolSize:     opts.PoolSize,
		MinIdleConns: opts.MinIdleConns,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		PoolTimeout:  opts.PoolTimeout,
	}

	// Create Redis client
	rdb := goredis.NewClient(redisOptions)

	// Verify connection with context
	if err := rdb.Ping(ctx).Err(); err != nil {
		// Close client on failure
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Client{
		client: rdb,
		opts:   opts,
	}, nil
}

// Name returns the storage type identifier.
// Implements storage.Client interface.
func (c *Client) Name() string {
	return "redis"
}

// Ping checks if the connection to Redis is alive.
// It performs a lightweight ping operation to verify connectivity.
// Implements storage.Client interface.
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection gracefully.
// It ensures all pending operations are completed and resources are released.
// This method is idempotent and safe to call multiple times.
// Implements storage.Client interface.
func (c *Client) Close() error {
	return c.client.Close()
}

// Health returns a HealthChecker function for Redis health monitoring.
// The returned function can be called to verify the Redis connection status.
// Implements storage.Client interface.
func (c *Client) Health() storage.HealthChecker {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return c.Ping(ctx)
	}
}

// Client returns the underlying Client instance.
// This allows direct access to go-redis functionality for advanced operations.
//
// Example:
//
//	rdb := client.Client()
//	err := rdb.Set(ctx, "key", "value", time.Hour).Err()
//	val, err := rdb.Get(ctx, "key").Result()
func (c *Client) Client() *goredis.Client {
	return c.client
}

// Options returns the Redis options used by this client.
func (c *Client) Options() *options.Options {
	return c.opts
}

// Do executes a Redis command with the given arguments.
// This is a convenience method for executing arbitrary Redis commands.
//
// Example:
//
//	result, err := client.Do(ctx, "GET", "key")
func (c *Client) Do(ctx context.Context, args ...interface{}) *goredis.Cmd {
	return c.client.Do(ctx, args...)
}

// Pipeline returns a Redis pipeline for batch operations.
// Pipelines allow multiple commands to be sent in a single round trip.
//
// Example:
//
//	pipe := client.Pipeline()
//	pipe.Set(ctx, "key1", "value1", 0)
//	pipe.Set(ctx, "key2", "value2", 0)
//	cmds, err := pipe.Exec(ctx)
func (c *Client) Pipeline() goredis.Pipeliner {
	return c.client.Pipeline()
}

// TxPipeline returns a transactional Redis pipeline.
// All commands in the pipeline will be executed atomically.
//
// Example:
//
//	pipe := client.TxPipeline()
//	pipe.Set(ctx, "key1", "value1", 0)
//	pipe.Set(ctx, "key2", "value2", 0)
//	cmds, err := pipe.Exec(ctx)
func (c *Client) TxPipeline() goredis.Pipeliner {
	return c.client.TxPipeline()
}
