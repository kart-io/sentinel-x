package postgres

import (
	"context"
	"fmt"
	"sync"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// SimpleFactory implements the storage.Factory interface for creating PostgreSQL clients.
// It encapsulates the PostgreSQL client creation logic and configuration,
// enabling dependency injection and simplified testing.
//
// Example usage:
//
//	opts := options.NewOptions()
//	opts.Host = "localhost"
//	opts.Database = "mydb"
//
//	factory := NewSimpleFactory(opts)
//	client, err := factory.Create(context.Background())
//	if err != nil {
//	    log.Fatalf("failed to create PostgreSQL client: %v", err)
//	}
//	defer client.Close()
type SimpleFactory struct {
	opts *Options
}

// NewSimpleFactory creates a new PostgreSQL client factory with the provided options.
// The factory can be used to create multiple PostgreSQL client instances
// with the same configuration.
func NewSimpleFactory(opts *Options) *SimpleFactory {
	return &SimpleFactory{
		opts: opts,
	}
}

// Create creates and initializes a new PostgreSQL client.
// It validates the configuration, establishes a connection,
// and verifies connectivity before returning the client.
//
// Implements storage.Factory interface.
func (f *SimpleFactory) Create(ctx context.Context) (storage.Client, error) {
	if f.opts == nil {
		return nil, fmt.Errorf("postgres options cannot be nil")
	}

	client, err := NewWithContext(ctx, f.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client: %w", err)
	}

	return client, nil
}

// Options returns the PostgreSQL options used by this factory.
func (f *SimpleFactory) Options() *Options {
	return f.opts
}

// Clone creates a new factory with a copy of the current options.
func (f *SimpleFactory) Clone() *SimpleFactory {
	optsCopy := *f.opts
	return &SimpleFactory{
		opts: &optsCopy,
	}
}

// Compile-time check that SimpleFactory implements storage.Factory.
var _ storage.Factory = (*SimpleFactory)(nil)

// Factory provides a way to create and manage multiple PostgreSQL clients.
// This is useful for applications that need to connect to multiple databases.
type Factory struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewFactory creates a new PostgreSQL client factory.
func NewFactory() *Factory {
	return &Factory{
		clients: make(map[string]*Client),
	}
}

// Create creates a new PostgreSQL client with the given name and options.
// If a client with the same name already exists, it returns an error.
func (f *Factory) Create(name string, opts *Options) (*Client, error) {
	return f.CreateWithContext(context.Background(), name, opts)
}

// CreateWithContext creates a new PostgreSQL client with the given context, name, and options.
// The context allows for timeout and cancellation during connection establishment.
func (f *Factory) CreateWithContext(ctx context.Context, name string, opts *Options) (*Client, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if client already exists
	if _, exists := f.clients[name]; exists {
		return nil, fmt.Errorf("postgres client '%s' already exists", name)
	}

	// Create new client
	client, err := NewWithContext(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client '%s': %w", name, err)
	}

	// Store client
	f.clients[name] = client

	return client, nil
}

// Get retrieves a PostgreSQL client by name.
// Returns nil if the client doesn't exist.
func (f *Factory) Get(name string) *Client {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.clients[name]
}

// GetOrCreate retrieves an existing client or creates a new one if it doesn't exist.
func (f *Factory) GetOrCreate(name string, opts *Options) (*Client, error) {
	return f.GetOrCreateWithContext(context.Background(), name, opts)
}

// GetOrCreateWithContext retrieves an existing client or creates a new one with context.
func (f *Factory) GetOrCreateWithContext(ctx context.Context, name string, opts *Options) (*Client, error) {
	// First, try to get existing client (read lock)
	f.mu.RLock()
	if client, exists := f.clients[name]; exists {
		f.mu.RUnlock()
		return client, nil
	}
	f.mu.RUnlock()

	// Client doesn't exist, create it (write lock)
	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := f.clients[name]; exists {
		return client, nil
	}

	// Create new client
	client, err := NewWithContext(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client '%s': %w", name, err)
	}

	// Store client
	f.clients[name] = client

	return client, nil
}

// Remove removes a client by name and closes its connection.
// Returns true if the client was found and removed, false otherwise.
func (f *Factory) Remove(name string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	client, exists := f.clients[name]
	if !exists {
		return false, nil
	}

	// Close the client connection
	if err := client.Close(); err != nil {
		return true, fmt.Errorf("failed to close postgres client '%s': %w", name, err)
	}

	// Remove from map
	delete(f.clients, name)

	return true, nil
}

// CloseAll closes all managed clients and clears the factory.
func (f *Factory) CloseAll() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errs []error

	for name, client := range f.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close client '%s': %w", name, err))
		}
	}

	// Clear the map
	f.clients = make(map[string]*Client)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing postgres clients: %v", errs)
	}

	return nil
}

// List returns the names of all managed clients.
func (f *Factory) List() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.clients))
	for name := range f.clients {
		names = append(names, name)
	}

	return names
}

// Count returns the number of managed clients.
func (f *Factory) Count() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.clients)
}
