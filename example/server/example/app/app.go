// Package app provides the example server application.
package app

import (
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/example/server/example/handler"
	"github.com/kart-io/sentinel-x/example/server/example/service/authservice"
	"github.com/kart-io/sentinel-x/example/server/example/service/helloservice"
	v1 "github.com/kart-io/sentinel-x/pkg/api/hello/v1"
	// Import bridges to register them
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/security/authz/rbac"
	"github.com/spf13/pflag"
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
	// Initialize logger first
	if err := opts.Log.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() { _ = logger.Flush() }()

	// Print startup banner
	printBanner(opts)

	// Log server startup
	logger.Infow("Starting server",
		"app", appName,
		"version", app.GetVersion(),
		"mode", opts.Server.Mode.String(),
	)

	// Configure health manager
	configureHealth(opts)

	// Initialize JWT authenticator if enabled
	var jwtAuth *jwt.JWT
	var rbacAuthz *rbac.RBAC
	var tokenStore *jwt.MemoryStore

	if !opts.JWT.DisableAuth {
		// Create token store
		tokenStore = jwt.NewMemoryStore()

		// Create JWT authenticator
		var err error
		jwtAuth, err = jwt.New(
			jwt.WithOptions(opts.JWT),
			jwt.WithStore(tokenStore),
		)
		if err != nil {
			return fmt.Errorf("failed to create JWT authenticator: %w", err)
		}

		// Create RBAC authorizer
		rbacAuthz = rbac.New()

		// Define roles with permissions
		_ = rbacAuthz.AddRole("admin",
			authz.NewPermission("*", "*"),
		)
		_ = rbacAuthz.AddRole("editor",
			authz.NewPermission("hello", "read"),
			authz.NewPermission("hello", "create"),
			authz.NewPermission("hello", "update"),
		)
		_ = rbacAuthz.AddRole("viewer",
			authz.NewPermission("hello", "read"),
		)

		logger.Infow("Auth initialized",
			"jwt_issuer", opts.JWT.Issuer,
			"jwt_expired", opts.JWT.Expired,
		)
	}

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
	_ = mgr.RegisterService(
		helloSvc,
		httpHandler,
		&transport.GRPCServiceDesc{
			ServiceDesc: &v1.HelloService_ServiceDesc,
			ServiceImpl: grpcHandler,
		},
	)

	// Register auth routes if enabled
	if jwtAuth != nil {
		authSvc := authservice.NewService()
		authHandler := handler.NewAuthHTTPHandler(jwtAuth)
		_ = mgr.RegisterHTTP(authSvc, authHandler)

		// Assign roles to demo users
		for _, user := range authHandler.GetUsers() {
			for _, role := range user.Roles {
				_ = rbacAuthz.AssignRole(user.ID, role)
			}
		}

		// Configure auth middleware
		opts.Server.HTTP.Middleware.Auth = mwopts.AuthOptions{
			Authenticator: jwtAuth,
			TokenLookup:   "header:Authorization",
			AuthScheme:    "Bearer",
			SkipPaths: []string{
				"/api/v1/auth/login",
				"/health", "/live", "/ready", "/metrics",
			},
		}
		opts.Server.HTTP.Middleware.DisableAuth = false

		// Configure authz middleware
		opts.Server.HTTP.Middleware.Authz = mwopts.AuthzOptions{
			Authorizer: rbacAuthz,
			SkipPaths: []string{
				"/api/v1/auth/login",
				"/api/v1/auth/refresh",
				"/api/v1/auth/logout",
				"/api/v1/auth/me",
				"/health", "/live", "/ready", "/metrics",
			},
		}
		opts.Server.HTTP.Middleware.DisableAuthz = false
	}

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
	if !opts.JWT.DisableAuth {
		fmt.Println("  - Auth (enabled)")
		fmt.Println("  - Authz (enabled)")
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Endpoints:")

	if opts.Server.EnableHTTP() {
		fmt.Println("  API:")
		fmt.Printf("    GET  http://localhost%s/api/v1/hello?name=World\n", opts.Server.HTTP.Addr)
		fmt.Printf("    POST http://localhost%s/api/v1/hello\n", opts.Server.HTTP.Addr)

		if !opts.JWT.DisableAuth {
			fmt.Println("  Auth:")
			fmt.Printf("    POST http://localhost%s/api/v1/auth/login\n", opts.Server.HTTP.Addr)
			fmt.Printf("    POST http://localhost%s/api/v1/auth/refresh\n", opts.Server.HTTP.Addr)
			fmt.Printf("    POST http://localhost%s/api/v1/auth/logout\n", opts.Server.HTTP.Addr)
			fmt.Printf("    GET  http://localhost%s/api/v1/auth/me\n", opts.Server.HTTP.Addr)
			fmt.Println("  Demo Users:")
			fmt.Println("    admin/admin123   (role: admin - full access)")
			fmt.Println("    editor/editor123 (role: editor - read/create/update)")
			fmt.Println("    viewer/viewer123 (role: viewer - read only)")
		}

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
	Log    *logopts.Options    `json:"log" mapstructure:"log"`
	JWT    *jwtopts.Options    `json:"jwt" mapstructure:"jwt"`
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		Server: serveropts.NewOptions(),
		Log:    logopts.NewOptions(),
		JWT:    jwtopts.NewOptions(),
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.Server.AddFlags(fs)
	o.Log.AddFlags(fs)
	o.JWT.AddFlags(fs)
}

// Validate validates the options.
func (o *Options) Validate() error {
	if err := o.Log.Validate(); err != nil {
		return err
	}
	if err := o.JWT.Validate(); err != nil {
		return err
	}
	return o.Server.Validate()
}

// Complete completes the options.
func (o *Options) Complete() error {
	return o.Server.Complete()
}
