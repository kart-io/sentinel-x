package mysql

import (
	"context"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// Factory implements the storage.Factory interface for creating MySQL clients.
// It encapsulates the MySQL client creation logic and configuration,
// enabling dependency injection and simplified testing.
//
// Example usage:
//
//	opts := NewOptions()
//	opts.Host = "localhost"
//	opts.Database = "mydb"
//
//	factory := NewFactory(opts)
//	client, err := factory.Create(context.Background())
//	if err != nil {
//	    log.Fatalf("failed to create MySQL client: %v", err)
//	}
//	defer client.Close()
type Factory struct {
	opts *Options
}

// NewFactory creates a new MySQL client factory with the provided options.
// The factory can be used to create multiple MySQL client instances
// with the same configuration.
//
// Parameters:
// - opts: MySQL configuration options
//
// Returns a new Factory instance.
func NewFactory(opts *Options) *Factory {
	return &Factory{
		opts: opts,
	}
}

// Create creates and initializes a new MySQL client.
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
// - storage.Client: A fully initialized MySQL client
// - error: Any error that occurred during creation
func (f *Factory) Create(ctx context.Context) (storage.Client, error) {
	if f.opts == nil {
		return nil, fmt.Errorf("mysql options cannot be nil")
	}

	client, err := NewWithContext(ctx, f.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql client: %w", err)
	}

	return client, nil
}

// Options returns the MySQL options used by this factory.
// This is useful for inspecting or cloning the configuration.
func (f *Factory) Options() *Options {
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
//	devFactory.Options().Database = "dev_db"
func (f *Factory) Clone() *Factory {
	// Create a shallow copy of options
	optsCopy := *f.opts
	return &Factory{
		opts: &optsCopy,
	}
}
