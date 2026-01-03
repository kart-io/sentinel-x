// Package app provides the API server application.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kart-io/sentinel-x/cmd/api/app/options"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
)

const (
	// Name is the name of the application.
	Name = "sentinel-api"

	// commandDesc is the description of the command.
	commandDesc = `Sentinel-X API Server

The main API server for Sentinel-X platform.

This server provides:
  - RESTful HTTP API and gRPC endpoints
  - JWT authentication and RBAC authorization
  - MySQL database integration
  - Redis cache integration
  - Health checks and metrics
  - Configurable middleware stack`
)

// NewApp creates and returns a new App object with default parameters.
func NewApp() *app.App {
	opts := options.NewServerOptions()
	application := app.NewApp(
		app.WithName(Name),
		app.WithDescription(commandDesc),
		app.WithOptions(opts),
		app.WithRunFunc(run(opts)),
	)

	return application
}

// run contains the main logic for initializing and running the server.
func run(opts *options.ServerOptions) app.RunFunc {
	return func() error {
		// Load the configuration options
		cfg, err := opts.Config()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		ctx := setupSignalContext()

		// Build the server using the configuration
		server, err := cfg.NewServer(ctx)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		// Run the server with signal context for graceful shutdown
		return server.Run(ctx)
	}
}

// setupSignalContext returns a context that is cancelled on SIGINT or SIGTERM.
func setupSignalContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		<-c
		os.Exit(1)
	}()
	return ctx
}
