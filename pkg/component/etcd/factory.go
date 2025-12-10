package etcd

import (
	"context"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// Factory implements the storage.Factory interface for creating etcd clients.
// It encapsulates the client creation logic and allows for dependency injection
// and testing with mock implementations.
type Factory struct {
	opts *Options
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
func NewFactory(opts *Options) *Factory {
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

// CreateWithOptions creates a new etcd client with specific options.
// This allows creating clients with different configurations from the same factory.
//
// Example usage:
//
//	factory := NewFactory(defaultOpts)
//
//	// Create a client with custom options
//	customOpts := NewOptions()
//	customOpts.Endpoints = []string{"etcd-prod:2379"}
//	client, err := factory.CreateWithOptions(ctx, customOpts)
func (f *Factory) CreateWithOptions(ctx context.Context, opts *Options) (*Client, error) {
	return NewWithContext(ctx, opts)
}

// MustCreate creates a new etcd client and panics if creation fails.
// This is useful for initialization code where failure should stop the program.
//
// Example usage:
//
//	factory := NewFactory(opts)
//	client := factory.MustCreate(context.Background())
//	defer client.Close()
func (f *Factory) MustCreate(ctx context.Context) *Client {
	client, err := NewWithContext(ctx, f.opts)
	if err != nil {
		panic(fmt.Sprintf("failed to create etcd client: %v", err))
	}
	return client
}
