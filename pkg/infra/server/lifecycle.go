// Package server provides a unified multi-protocol server framework
// supporting HTTP and gRPC with pluggable adapters.
package server

import "context"

// Lifecycle defines the lifecycle interface for servers.
type Lifecycle interface {
	// Start starts the server.
	Start(ctx context.Context) error
	// Stop stops the server gracefully.
	Stop(ctx context.Context) error
}

// Server is an alias for Lifecycle, representing a runnable server.
type Server = Lifecycle

// Runnable represents a component that can be started and stopped.
type Runnable interface {
	Lifecycle
	// Name returns the server name for identification.
	Name() string
}
