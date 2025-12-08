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
  sentinel-example --pprof

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

	// Configure middleware
	configureMiddleware(opts)

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

// configureMiddleware configures middleware based on options.
func configureMiddleware(opts *Options) {
	mw := opts.Server.HTTP.Middleware

	// Configure recovery
	mw.Recovery.EnableStackTrace = opts.EnableStackTrace

	// Configure CORS
	mw.DisableCORS = !opts.EnableCORS

	// Configure metrics
	mw.DisableMetrics = !opts.EnableMetrics
	if opts.EnableMetrics {
		mw.Metrics.Namespace = "sentinel"
		mw.Metrics.Subsystem = "example"
	}

	// Configure pprof
	mw.DisablePprof = !opts.EnablePprof

	// Configure health
	mw.DisableHealth = false
	middleware.GetHealthManager().SetVersion(app.GetVersion())
	middleware.GetHealthManager().RegisterChecker("service", func() error {
		return nil
	})
}

// printBanner prints the startup banner.
func printBanner(opts *Options) {
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
	fmt.Println("  - Recovery (enabled)")
	fmt.Println("  - RequestID (enabled)")
	fmt.Println("  - Logger (enabled)")

	if opts.EnableCORS {
		fmt.Println("  - CORS (enabled)")
	}
	if opts.EnableMetrics {
		fmt.Println("  - Metrics (enabled)")
	}
	if opts.EnablePprof {
		fmt.Println("  - Pprof (enabled)")
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Endpoints:")

	if opts.Server.EnableHTTP() {
		fmt.Println("  API:")
		fmt.Printf("    GET  http://localhost%s/api/v1/hello?name=World\n", opts.Server.HTTP.Addr)
		fmt.Printf("    POST http://localhost%s/api/v1/hello\n", opts.Server.HTTP.Addr)
		fmt.Println("  Health:")
		fmt.Printf("    GET  http://localhost%s/health\n", opts.Server.HTTP.Addr)
		fmt.Printf("    GET  http://localhost%s/live\n", opts.Server.HTTP.Addr)
		fmt.Printf("    GET  http://localhost%s/ready\n", opts.Server.HTTP.Addr)

		if opts.EnableMetrics {
			fmt.Printf("  Metrics: http://localhost%s/metrics\n", opts.Server.HTTP.Addr)
		}
		if opts.EnablePprof {
			fmt.Printf("  Pprof: http://localhost%s/debug/pprof/\n", opts.Server.HTTP.Addr)
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

	// Feature flags
	EnableCORS       bool `json:"enable-cors" mapstructure:"enable-cors"`
	EnableMetrics    bool `json:"enable-metrics" mapstructure:"enable-metrics"`
	EnablePprof      bool `json:"enable-pprof" mapstructure:"enable-pprof"`
	EnableStackTrace bool `json:"enable-stack-trace" mapstructure:"enable-stack-trace"`
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		Server:           serveropts.NewOptions(),
		EnableCORS:       false,
		EnableMetrics:    true,
		EnablePprof:      false,
		EnableStackTrace: false,
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.Server.AddFlags(fs)

	fs.BoolVar(&o.EnableCORS, "cors", o.EnableCORS, "Enable CORS middleware")
	fs.BoolVar(&o.EnableMetrics, "metrics", o.EnableMetrics, "Enable metrics endpoint")
	fs.BoolVar(&o.EnablePprof, "pprof", o.EnablePprof, "Enable pprof endpoints")
	fs.BoolVar(&o.EnableStackTrace, "stack-trace", o.EnableStackTrace, "Enable stack trace in error responses")
}

// Validate validates the options.
func (o *Options) Validate() error {
	return o.Server.Validate()
}

// Complete completes the options.
func (o *Options) Complete() error {
	return o.Server.Complete()
}
