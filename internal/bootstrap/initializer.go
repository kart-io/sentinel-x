// Package bootstrap provides modular initialization components for the API server.
package bootstrap

import "context"

// Initializer defines the interface for initialization components.
// Each initializer is responsible for setting up a specific subsystem.
type Initializer interface {
	// Name returns the name of the initializer for logging purposes.
	Name() string

	// Initialize performs the initialization logic.
	// Returns an error if initialization fails.
	Initialize(ctx context.Context) error
}

// Shutdowner defines the interface for components that need graceful shutdown.
type Shutdowner interface {
	// Shutdown performs graceful shutdown of the component.
	// The context may contain a deadline for shutdown timeout.
	Shutdown(ctx context.Context) error
}
