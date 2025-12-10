package bootstrap

import (
	"context"

	"github.com/kart-io/logger"
)

// Run initializes and runs the application components.
// It handles the lifecycle of the application, including initialization,
// execution, and graceful shutdown.
func Run(opts *BootstrapOptions) error {
	b := NewAppBootstrapper(opts)

	// Use a background context for initialization and lifecycle management
	ctx := context.Background()

	// Initialize all components
	if err := b.Initialize(ctx); err != nil {
		return err
	}

	// Ensure graceful shutdown on exit
	defer func() {
		_ = logger.Flush()
		_ = b.Shutdown(ctx)
	}()

	// Get server manager and run it
	// This will block until the server is stopped
	serverInit := b.GetServerManager()
	return serverInit.Run()
}
