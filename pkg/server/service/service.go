// Package service provides the service layer interface definitions.
// Business logic should only exist in the service layer, not in handlers.
package service

import "context"

// Service is a marker interface for all business services.
// All business logic must be implemented in types that implement this interface.
type Service interface {
	// ServiceName returns the service name for registration.
	ServiceName() string
}

// Registrable represents a service that can register itself to transports.
type Registrable interface {
	Service
}

// HealthChecker represents a service that can report health status.
type HealthChecker interface {
	// HealthCheck performs a health check and returns an error if unhealthy.
	HealthCheck(ctx context.Context) error
}

// Initializable represents a service that requires initialization.
type Initializable interface {
	// Init initializes the service.
	Init(ctx context.Context) error
}

// Closeable represents a service that requires cleanup.
type Closeable interface {
	// Close releases resources held by the service.
	Close(ctx context.Context) error
}
