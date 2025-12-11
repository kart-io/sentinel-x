// Package bootstrap provides modular initialization components for the API server.
package bootstrap

import "context"

// Initializer defines the interface for initialization components.
// Each initializer is responsible for setting up a specific subsystem.
type Initializer interface {
	// Name returns the name of the initializer for logging purposes.
	Name() string

	// Dependencies returns the names of initializers this one depends on.
	// The bootstrapper will ensure dependencies are initialized first.
	// Return nil or empty slice if no dependencies.
	Dependencies() []string

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

// BaseInitializer provides default implementations for Initializer interface.
// Embed this in your initializer structs to get default Dependencies() behavior.
type BaseInitializer struct {
	name string
	deps []string
}

// NewBaseInitializer creates a new BaseInitializer with the given name and dependencies.
func NewBaseInitializer(name string, deps ...string) BaseInitializer {
	return BaseInitializer{
		name: name,
		deps: deps,
	}
}

// Name returns the name of the initializer.
func (b BaseInitializer) Name() string {
	return b.name
}

// Dependencies returns the names of initializers this one depends on.
func (b BaseInitializer) Dependencies() []string {
	return b.deps
}
