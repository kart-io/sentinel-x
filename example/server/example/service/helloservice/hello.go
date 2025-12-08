// Package helloservice provides the HelloService implementation.
// This is the SERVICE LAYER where all business logic resides.
// Both HTTP and gRPC handlers call into this service - no business logic
// should exist in the transport layer.
package helloservice

import (
	"context"
	"fmt"
)

// Service implements the HelloService business logic.
// This is the single source of truth for the SayHello functionality.
type Service struct {
	// Add any dependencies here (database, cache, etc.)
}

// NewService creates a new HelloService.
func NewService() *Service {
	return &Service{}
}

// ServiceName returns the service name for registration.
func (s *Service) ServiceName() string {
	return "HelloService"
}

// SayHello is the core business logic method.
// Both HTTP and gRPC handlers should call this method.
// This ensures business logic is implemented only once.
func (s *Service) SayHello(ctx context.Context, name string) (string, error) {
	// Validate input - return structured error
	if name == "" {
		return "", ErrEmptyName
	}

	// Additional validation example
	if len(name) > 100 {
		return "", ErrNameTooLong
	}

	// This is where the actual business logic lives.
	// In a real application, this might:
	// - Query a database
	// - Call external services
	// - Perform complex computations
	message := fmt.Sprintf("Hello, %s! Welcome to Sentinel-X.", name)

	return message, nil
}

// HealthCheck performs a health check.
func (s *Service) HealthCheck(ctx context.Context) error {
	// Add real health check logic here
	return nil
}
