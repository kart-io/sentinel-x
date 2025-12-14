// Package app provides the API server application.
package app

import (
	"fmt"

	// Use the shared bootstrap package
	"github.com/kart-io/sentinel-x/internal/api/router"
	"github.com/kart-io/sentinel-x/internal/bootstrap"
	// Import bridges to register them
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
)

const (
	appName        = "sentinel-api"
	appDescription = `Sentinel-X API Server

The main API server for Sentinel-X platform.

This server provides:
  - RESTful HTTP API and gRPC endpoints
  - JWT authentication and RBAC authorization
  - MySQL database integration
  - Redis cache integration
  - Health checks and metrics
  - Configurable middleware stack

Examples:
  # Start with default configuration
  sentinel-api

  # Start HTTP only mode
  sentinel-api --server.mode=http

  # Start with custom address
  sentinel-api --http.addr=:8100 --grpc.addr=:9100

  # Use config file
  sentinel-api -c /etc/sentinel-x/api.yaml

  # Enable debug logging
  sentinel-api --log.level=debug

Configuration:
  Configuration can be provided via:
  - Command-line flags (highest priority)
  - Environment variables (prefix: SENTINEL_API_)
  - Configuration file (YAML)
  - Default values (lowest priority)`
)

// NewApp creates a new application instance.
func NewApp() *app.App {
	opts := NewOptions()

	return app.NewApp(
		app.WithName(appName),
		app.WithDescription(appDescription),
		app.WithOptions(opts),
		app.WithRunFunc(func() error {
			return Run(opts)
		}),
	)
}

// Run runs the API server with the given options.
// This is the main entry point that orchestrates the bootstrapping process.
func Run(opts *Options) error {
	// Print banner before initialization
	printBanner(opts)

	// Create bootstrap options
	bootstrapOpts := &bootstrap.Options{
		AppName:      appName,
		AppVersion:   app.GetVersion(),
		ServerMode:   opts.Server.Mode.String(),
		LogOpts:      opts.Log,
		ServerOpts:   opts.Server,
		JWTOpts:      opts.JWT,
		MySQLOpts:    opts.MySQL,
		RedisOpts:    opts.Redis,
		RegisterFunc: router.Register,
	}

	return bootstrap.Run(bootstrapOpts)
}

// printBanner prints the startup banner.
func printBanner(opts *Options) {
	mw := opts.Server.HTTP.Middleware

	fmt.Println("===========================================")
	fmt.Println("  Sentinel-X API Server")
	fmt.Println("===========================================")
	fmt.Printf("Version: %s\n", app.GetVersion())
	fmt.Printf("Mode: %s\n", opts.Server.Mode.String())

	if opts.Server.EnableHTTP() {
		fmt.Printf("HTTP: %s (adapter: %s)\n", opts.Server.HTTP.Addr, opts.Server.HTTP.Adapter)
	}
	if opts.Server.EnableGRPC() {
		fmt.Printf("gRPC: %s\n", opts.Server.GRPC.Addr)
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Configuration:")
	fmt.Printf("  Logger: level=%s, format=%s\n", opts.Log.Level, opts.Log.Format)
	if opts.MySQL.Host != "" {
		fmt.Printf("  Database: %s:%d/%s\n", opts.MySQL.Host, opts.MySQL.Port, opts.MySQL.Database)
	}
	if opts.Redis.Host != "" {
		fmt.Printf("  Redis: %s:%d (db=%d)\n", opts.Redis.Host, opts.Redis.Port, opts.Redis.Database)
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Middleware:")
	if !mw.DisableRecovery {
		fmt.Println("  - Recovery")
	}
	if !mw.DisableRequestID {
		fmt.Println("  - RequestID")
	}
	if !mw.DisableLogger {
		fmt.Println("  - Logger")
	}
	if !mw.DisableCORS {
		fmt.Println("  - CORS")
	}
	if !mw.DisableTimeout {
		fmt.Println("  - Timeout")
	}
	if !mw.DisableHealth {
		fmt.Println("  - Health")
	}
	if !mw.DisableMetrics {
		fmt.Println("  - Metrics")
	}
	if !mw.DisablePprof {
		fmt.Println("  - Pprof")
	}
	if !mw.DisableAuth {
		fmt.Println("  - Auth (JWT)")
	}
	if !mw.DisableAuthz {
		fmt.Println("  - Authz (RBAC)")
	}

	if opts.Server.EnableHTTP() && !mw.DisableHealth {
		fmt.Println("-------------------------------------------")
		fmt.Println("Endpoints:")
		fmt.Printf("  Health: http://localhost%s%s\n", opts.Server.HTTP.Addr, mw.Health.Path)
		fmt.Printf("  Liveness: http://localhost%s%s\n", opts.Server.HTTP.Addr, mw.Health.LivenessPath)
		fmt.Printf("  Readiness: http://localhost%s%s\n", opts.Server.HTTP.Addr, mw.Health.ReadinessPath)
		if !mw.DisableMetrics {
			fmt.Printf("  Metrics: http://localhost%s%s\n", opts.Server.HTTP.Addr, mw.Metrics.Path)
		}
		if !mw.DisablePprof {
			fmt.Printf("  Pprof: http://localhost%s%s/\n", opts.Server.HTTP.Addr, mw.Pprof.Prefix)
		}
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Press Ctrl+C to gracefully shutdown")
	fmt.Println()
}
