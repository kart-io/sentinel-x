// Package app provides the example server application.
package app

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/kart-io/sentinel-x/pkg/app"
	"github.com/kart-io/sentinel-x/pkg/middleware"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	"github.com/kart-io/sentinel-x/pkg/server"
	v1 "github.com/kart-io/sentinel-x/example/server/example/api/hello/v1"
	"github.com/kart-io/sentinel-x/example/server/example/handler"
	"github.com/kart-io/sentinel-x/example/server/example/service/helloservice"
	"github.com/kart-io/sentinel-x/pkg/server/transport"

	// Import bridges to register them
	_ "github.com/kart-io/sentinel-x/pkg/bridge/echo"
	_ "github.com/kart-io/sentinel-x/pkg/bridge/gin"
)

const (
	appName        = "sentinel-example"
	appDescription = `Sentinel-X Example Server

This example demonstrates:
  - HTTP and gRPC server with unified service layer
  - Middleware configuration (recovery, logger, CORS, health, metrics, pprof)
  - Configuration via flags, environment variables, and config file

Examples:
  # Start with default configuration
  sentinel-example

  # Start HTTP only mode
  sentinel-example --server.mode=http

  # Start with custom address
  sentinel-example --http.addr=:8081 --grpc.addr=:9091

  # Enable pprof for debugging
  sentinel-example --middleware.disable-pprof=false

  # Enable CORS
  sentinel-example --middleware.disable-cors=false

  # Use config file
  sentinel-example -c config.yaml`
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

// Run runs the server with the given options.
func Run(opts *Options) error {
	// Print startup banner
	printBanner(opts)

	// Configure health manager
	configureHealth(opts)

	// Create server manager
	mgr := server.NewManager(
		serveropts.WithMode(opts.Server.Mode),
		serveropts.WithHTTPOptions(opts.Server.HTTP),
		serveropts.WithGRPCOptions(opts.Server.GRPC),
		serveropts.WithShutdownTimeout(opts.Server.ShutdownTimeout),
	)

	// Create service and handlers
	helloSvc := helloservice.NewService()
	httpHandler := handler.NewHelloHTTPHandler(helloSvc)
	grpcHandler := handler.NewHelloGRPCHandler(helloSvc)

	// Register service
	mgr.RegisterService(
		helloSvc,
		httpHandler,
		&transport.GRPCServiceDesc{
			ServiceDesc: &v1.HelloService_ServiceDesc,
			ServiceImpl: grpcHandler,
		},
	)

	// Run server
	return mgr.Run()
}

// configureHealth configures the health manager.
func configureHealth(opts *Options) {
	middleware.GetHealthManager().SetVersion(app.GetVersion())
	middleware.GetHealthManager().RegisterChecker("service", func() error {
		return nil
	})
}

// printBanner prints the startup banner.
func printBanner(opts *Options) {
	mw := opts.Server.HTTP.Middleware

	fmt.Println("===========================================")
	fmt.Println("  Sentinel-X Example Server")
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
	fmt.Println("Middleware:")
	if !mw.DisableRecovery {
		fmt.Println("  - Recovery (enabled)")
	}
	if !mw.DisableRequestID {
		fmt.Println("  - RequestID (enabled)")
	}
	if !mw.DisableLogger {
		fmt.Println("  - Logger (enabled)")
	}
	if !mw.DisableCORS {
		fmt.Println("  - CORS (enabled)")
	}
	if !mw.DisableTimeout {
		fmt.Println("  - Timeout (enabled)")
	}
	if !mw.DisableHealth {
		fmt.Println("  - Health (enabled)")
	}
	if !mw.DisableMetrics {
		fmt.Println("  - Metrics (enabled)")
	}
	if !mw.DisablePprof {
		fmt.Println("  - Pprof (enabled)")
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Endpoints:")

	if opts.Server.EnableHTTP() {
		fmt.Println("  API:")
		fmt.Printf("    GET  http://localhost%s/api/v1/hello?name=World\n", opts.Server.HTTP.Addr)
		fmt.Printf("    POST http://localhost%s/api/v1/hello\n", opts.Server.HTTP.Addr)

		if !mw.DisableHealth {
			fmt.Println("  Health:")
			fmt.Printf("    GET  http://localhost%s%s\n", opts.Server.HTTP.Addr, mw.Health.Path)
			fmt.Printf("    GET  http://localhost%s%s\n", opts.Server.HTTP.Addr, mw.Health.LivenessPath)
			fmt.Printf("    GET  http://localhost%s%s\n", opts.Server.HTTP.Addr, mw.Health.ReadinessPath)
		}

		if !mw.DisableMetrics {
			fmt.Printf("  Metrics: http://localhost%s%s\n", opts.Server.HTTP.Addr, mw.Metrics.Path)
		}
		if !mw.DisablePprof {
			fmt.Printf("  Pprof: http://localhost%s%s/\n", opts.Server.HTTP.Addr, mw.Pprof.Prefix)
		}
	}

	if opts.Server.EnableGRPC() {
		fmt.Println("  gRPC:")
		fmt.Printf("    grpcurl -plaintext localhost%s api.hello.v1.HelloService/SayHello\n", opts.Server.GRPC.Addr)
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()
}

// Options contains all server options.
type Options struct {
	Server *serveropts.Options `json:"server" mapstructure:"server"`
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		Server: serveropts.NewOptions(),
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.Server.AddFlags(fs)
}

// Validate validates the options.
func (o *Options) Validate() error {
	return o.Server.Validate()
}

// Complete completes the options.
func (o *Options) Complete() error {
	return o.Server.Complete()
}
