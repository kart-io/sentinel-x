package redis

import (
	"context"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
	options "github.com/kart-io/sentinel-x/pkg/options/redis"
)

// Options is re-exported from pkg/options/redis for convenience.
type Options = options.Options

// NewOptions is re-exported from pkg/options/redis for convenience.
var NewOptions = options.NewOptions

// Factory implements the storage.Factory interface for creating Redis clients.
// It encapsulates the Redis client creation logic and configuration,
// enabling dependency injection and simplified testing.
//
// Example usage:
//
//	opts := NewOptions()
//	opts.Host = "localhost"
//	opts.Port = 6379
//
//	factory := NewFactory(opts)
//	client, err := factory.Create(context.Background())
//	if err != nil {
//	    log.Fatalf("failed to create Redis client: %v", err)
//	}
//	defer client.Close()
type Factory struct {
	opts *options.Options
}

// NewFactory creates a new Redis client factory with the provided options.
// The factory can be used to create multiple Redis client instances
// with the same configuration.
//
// Parameters:
// - opts: Redis configuration options
//
// Returns a new Factory instance.
func NewFactory(opts *options.Options) *Factory {
	return &Factory{
		opts: opts,
	}
}

// Create creates and initializes a new Redis client.
// It validates the configuration, establishes a connection,
// and verifies connectivity before returning the client.
//
// The context parameter can be used to:
// - Set timeout for the creation process
// - Cancel the operation if needed
//
// Implements storage.Factory interface.
//
// Returns:
// - storage.Client: A fully initialized Redis client
// - error: Any error that occurred during creation
func (f *Factory) Create(ctx context.Context) (storage.Client, error) {
	if f.opts == nil {
		return nil, fmt.Errorf("redis options cannot be nil")
	}

	client, err := NewWithContext(ctx, f.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	return client, nil
}

// Options returns the Redis options used by this factory.
// This is useful for inspecting or cloning the configuration.
func (f *Factory) Options() *options.Options {
	return f.opts
}

// Clone creates a new factory with a copy of the current options.
// This is useful when you need to create multiple factories with
// slightly different configurations based on the same base options.
//
// Example:
//
//	factory := NewFactory(baseOpts)
//	devFactory := factory.Clone()
//	devFactory.Options().Database = 1
func (f *Factory) Clone() *Factory {
	// Create a shallow copy of options
	optsCopy := *f.opts
	return &Factory{
		opts: &optsCopy,
	}
}

// Compile-time check that Factory implements storage.Factory.
var _ storage.Factory = (*Factory)(nil)
