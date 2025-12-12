package etcd

import (
	"context"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
	options "github.com/kart-io/sentinel-x/pkg/options/etcd"
)

// Options is re-exported from pkg/options/etcd for convenience.
type Options = options.Options

// NewOptions is re-exported from pkg/options/etcd for convenience.
var NewOptions = options.NewOptions

// Factory implements the storage.Factory interface for creating etcd clients.
// It encapsulates the client creation logic and allows for dependency injection
// and testing with mock implementations.
type Factory struct {
	opts *options.Options
}

// NewFactory creates a new Factory for etcd clients.
// The provided options will be used to create all clients produced by this factory.
//
// Example usage:
//
//	opts := NewOptions()
//	opts.Endpoints = []string{"localhost:2379"}
//	factory := NewFactory(opts)
//
//	client, err := factory.Create(ctx)
//	if err != nil {
//	    log.Fatalf("failed to create client: %v", err)
//	}
//	defer client.Close()
func NewFactory(opts *options.Options) *Factory {
	return &Factory{
		opts: opts,
	}
}

// Create creates and initializes a new etcd client.
// The context can be used to set timeouts for the initialization process.
// The returned client is ready to use (connected and verified).
//
// This implements the storage.Factory interface.
//
// Returns an error if client creation or initialization fails.
func (f *Factory) Create(ctx context.Context) (storage.Client, error) {
	client, err := NewWithContext(ctx, f.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}
	return client, nil
}

// Options returns the etcd options used by this factory.
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
//	devFactory.Options().Endpoints = []string{"etcd-dev:2379"}
func (f *Factory) Clone() *Factory {
	// Create a shallow copy of options
	optsCopy := *f.opts
	return &Factory{
		opts: &optsCopy,
	}
}
