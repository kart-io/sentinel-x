// Package storage provides unified interfaces and base types for storage implementations.
// This package defines the core abstractions that all storage clients (Redis, MySQL, etc.)
// must implement, enabling consistent health checking, connection management, and
// graceful shutdown across the sentinel-x project.
package storage

import (
	"context"
	"time"
)

// Client is the base interface that all storage clients must implement.
// It provides fundamental operations for connection management, health checking,
// and graceful shutdown. Each storage type (Redis, MySQL, MongoDB, etc.) should
// implement this interface to ensure consistent behavior across the system.
//
// Example usage:
//
//	var client storage.Client
//	client = redis.NewClient(opts)
//
//	if err := client.Ping(ctx); err != nil {
//	    log.Fatalf("failed to connect: %v", err)
//	}
//	defer client.Close()
type Client interface {
	// Name returns the storage type name for identification purposes.
	// This should be a lowercase identifier like "redis", "mysql", "mongodb", etc.
	// The name is used for logging, metrics, and health check reporting.
	Name() string

	// Ping checks if the connection to the storage backend is alive.
	// It should perform a lightweight operation to verify connectivity
	// without affecting performance. The context can be used to set
	// timeouts or cancel the ping operation.
	//
	// Returns an error if the connection is unavailable or unhealthy.
	Ping(ctx context.Context) error

	// Close closes the connection gracefully, releasing all resources.
	// This method should ensure that:
	// - All pending operations are completed or cancelled
	// - Connection pools are drained
	// - File handles and network connections are released
	//
	// Close should be idempotent and safe to call multiple times.
	Close() error

	// Health returns a HealthChecker function that can be called
	// to check the storage health status. This is useful for
	// integrating with health check endpoints and monitoring systems.
	//
	// The returned HealthChecker should capture the current client
	// instance and perform health verification when invoked.
	Health() HealthChecker
}

// HealthChecker is a function type that performs health checks on storage systems.
// It encapsulates the health check logic and can be called independently
// without direct access to the storage client.
//
// Example usage:
//
//	checker := client.Health()
//	if err := checker(); err != nil {
//	    log.Printf("health check failed: %v", err)
//	}
type HealthChecker func() error

// HealthStatus represents the result of a health check operation.
// It provides comprehensive information about the health state of a
// storage backend, including timing information and error details.
type HealthStatus struct {
	// Name identifies the storage instance being checked.
	// This should match the value returned by Client.Name().
	Name string

	// Healthy indicates whether the storage is functioning properly.
	// true means the storage is accessible and responding normally.
	Healthy bool

	// Latency measures how long the health check took to complete.
	// This can be used to detect performance degradation even when
	// the service is technically healthy.
	Latency time.Duration

	// Error contains the error details if the health check failed.
	// This is nil when Healthy is true.
	Error error
}

// Factory is an interface for creating storage clients.
// It encapsulates the client creation logic and allows for
// dependency injection and testing with mock implementations.
//
// Example usage:
//
//	factory := redis.NewFactory(config)
//	client, err := factory.Create(ctx)
//	if err != nil {
//	    log.Fatalf("failed to create client: %v", err)
//	}
type Factory interface {
	// Create creates and initializes a new storage client.
	// The context can be used to set timeouts for the initialization process.
	// The returned client should be ready to use (connected and verified).
	//
	// Returns an error if client creation or initialization fails.
	Create(ctx context.Context) (Client, error)
}
