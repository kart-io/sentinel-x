// Package datasource provides unified management for all storage clients.
package datasource

import (
	"context"
	"fmt"
)

// TypedGetter provides type-safe, generic access to storage clients.
// It eliminates boilerplate code by using Go 1.18+ generics to handle
// different client types uniformly.
//
// Usage:
//
//	mysqlGetter := mgr.MySQL()
//	client, err := mysqlGetter.Get("primary")
//	if err != nil {
//	    log.Fatal(err)
//	}
type TypedGetter[T any] struct {
	mgr         *Manager
	storageType StorageType
}

// NewTypedGetter creates a new type-safe getter for a specific storage type.
// The generic parameter T must match the actual client type for the storage.
//
// Example:
//
//	mysqlGetter := NewTypedGetter[*mysql.Client](mgr, TypeMySQL)
func NewTypedGetter[T any](mgr *Manager, st StorageType) *TypedGetter[T] {
	return &TypedGetter[T]{
		mgr:         mgr,
		storageType: st,
	}
}

// Get retrieves a storage client by name with lazy initialization.
// If the client is not yet initialized, it will be initialized using context.Background().
//
// Returns an error if:
//   - The instance is not registered
//   - Initialization fails
//   - Type assertion fails (should not happen if TypedGetter is correctly instantiated)
func (g *TypedGetter[T]) Get(name string) (T, error) {
	var zero T
	client, err := g.mgr.getClient(g.storageType, name)
	if err != nil {
		return zero, err
	}
	typed, ok := client.(T)
	if !ok {
		return zero, fmt.Errorf("type assertion failed for %s '%s': expected %T, got %T",
			g.storageType, name, zero, client)
	}
	return typed, nil
}

// GetWithContext retrieves a storage client by name with context support.
// This allows the caller to control timeouts and cancellation during initialization.
//
// If the client is not yet initialized, it will be initialized using the provided context.
func (g *TypedGetter[T]) GetWithContext(ctx context.Context, name string) (T, error) {
	var zero T
	client, err := g.mgr.getClientWithContext(ctx, g.storageType, name)
	if err != nil {
		return zero, err
	}
	typed, ok := client.(T)
	if !ok {
		return zero, fmt.Errorf("type assertion failed for %s '%s': expected %T, got %T",
			g.storageType, name, zero, client)
	}
	return typed, nil
}

// MustGet retrieves a storage client by name, panicking on error.
// This is useful in initialization code where failure is unrecoverable.
//
// Warning: This method will panic if the instance is not registered or initialization fails.
// Use Get() or GetWithContext() in production code where you need proper error handling.
func (g *TypedGetter[T]) MustGet(name string) T {
	client, err := g.Get(name)
	if err != nil {
		panic(fmt.Sprintf("failed to get %s instance '%s': %v", g.storageType, name, err))
	}
	return client
}
