package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kart-io/sentinel-x/cmd/scheduler/app/options"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
)

const (
	Name = "sentinel-scheduler"
	commandDesc = "Sentinel-X Distributed Scheduler Service"
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
		cfg, err := opts.Config()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		ctx := setupSignalContext()

		server, err := cfg.NewServer(ctx)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		return server.Run(ctx)
	}
}

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
