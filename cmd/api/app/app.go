// Package app provides the API server application.
package app

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	// Import bridges to register them
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/security/authz/rbac"
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

// Bootstrapper manages the initialization and lifecycle of the API server.
type Bootstrapper struct {
	opts    *Options
	dsMgr   *datasource.Manager
	srvMgr  *server.Manager
	jwtAuth *jwt.JWT
	rbac    *rbac.RBAC
}

// NewBootstrapper creates a new Bootstrapper instance.
func NewBootstrapper(opts *Options) *Bootstrapper {
	return &Bootstrapper{opts: opts}
}

// InitializeLogging initializes the logging system.
func (b *Bootstrapper) InitializeLogging() error {
	if err := b.opts.Log.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	printBanner(b.opts)

	logger.Infow("Starting Sentinel-X API server",
		"app", appName,
		"version", app.GetVersion(),
		"mode", b.opts.Server.Mode.String(),
	)

	return nil
}

// InitializeDatasources initializes all configured datasources.
func (b *Bootstrapper) InitializeDatasources(ctx context.Context) error {
	b.dsMgr = datasource.NewManager()
	datasource.SetGlobal(b.dsMgr)

	if b.opts.MySQL.Host != "" {
		if err := b.dsMgr.RegisterMySQL("primary", b.opts.MySQL); err != nil {
			return fmt.Errorf("failed to register mysql: %w", err)
		}
	}

	if b.opts.Redis.Host != "" {
		if err := b.dsMgr.RegisterRedis("cache", b.opts.Redis); err != nil {
			return fmt.Errorf("failed to register redis: %w", err)
		}
	}

	if err := b.dsMgr.InitAll(ctx); err != nil {
		return fmt.Errorf("failed to initialize datasources: %w", err)
	}

	logger.Info("Datasources initialized successfully")
	return nil
}

// InitializeAuth initializes JWT authentication and RBAC authorization.
func (b *Bootstrapper) InitializeAuth(ctx context.Context) error {
	if b.opts.JWT.DisableAuth {
		logger.Info("Authentication disabled")
		return nil
	}

	tokenStore := jwt.NewMemoryStore()
	jwtAuth, err := jwt.New(
		jwt.WithOptions(b.opts.JWT),
		jwt.WithStore(tokenStore),
	)
	if err != nil {
		return fmt.Errorf("failed to create JWT authenticator: %w", err)
	}

	rbacAuthz := rbac.New()

	if err := rbacAuthz.AddRole("admin", authz.NewPermission("*", "*")); err != nil {
		return fmt.Errorf("failed to add admin role: %w", err)
	}

	if err := rbacAuthz.AddRole("user",
		authz.NewPermission("user", "read"),
		authz.NewPermission("user", "update"),
	); err != nil {
		return fmt.Errorf("failed to add user role: %w", err)
	}

	if err := rbacAuthz.AddRole("guest", authz.NewPermission("*", "read")); err != nil {
		return fmt.Errorf("failed to add guest role: %w", err)
	}

	b.jwtAuth = jwtAuth
	b.rbac = rbacAuthz

	logger.Infow("Authentication and authorization initialized",
		"jwt_issuer", b.opts.JWT.Issuer,
		"jwt_expired", b.opts.JWT.Expired,
		"roles", []string{"admin", "user", "guest"},
	)

	return nil
}

// ConfigureMiddleware configures all middleware components.
func (b *Bootstrapper) ConfigureMiddleware() {
	configureHealth(b.opts, b.dsMgr)

	if b.jwtAuth != nil && b.rbac != nil {
		configureAuthMiddleware(b.opts, b.jwtAuth, b.rbac)
	}
}

// CreateServerManager creates and configures the server manager.
func (b *Bootstrapper) CreateServerManager() error {
	b.srvMgr = server.NewManager(
		server.WithMode(b.opts.Server.Mode),
		server.WithHTTPOptions(b.opts.Server.HTTP),
		server.WithGRPCOptions(b.opts.Server.GRPC),
		server.WithShutdownTimeout(b.opts.Server.ShutdownTimeout),
	)

	// TODO: Register your services here
	// Example:
	// db, _ := b.dsMgr.GetMySQL("primary")
	// rdb, _ := b.dsMgr.GetRedis("cache")
	// userSvc := userservice.NewService(db.DB(), rdb.Client())
	// userHTTPHandler := handler.NewUserHTTPHandler(userSvc)
	// userGRPCHandler := handler.NewUserGRPCHandler(userSvc)
	// _ = b.srvMgr.RegisterService(userSvc, userHTTPHandler, &transport.GRPCServiceDesc{
	//     ServiceDesc: &apiv1.UserService_ServiceDesc,
	//     ServiceImpl: userGRPCHandler,
	// })

	logger.Info("All services registered successfully")
	return nil
}

// Shutdown gracefully shuts down all components.
func (b *Bootstrapper) Shutdown(ctx context.Context) error {
	if b.dsMgr != nil {
		b.dsMgr.CloseAll()
	}
	return nil
}

// Run runs the API server with the given options.
func Run(opts *Options) error {
	b := NewBootstrapper(opts)

	if err := b.InitializeLogging(); err != nil {
		return err
	}
	defer func() { _ = logger.Flush() }()

	ctx := context.Background()

	if err := b.InitializeDatasources(ctx); err != nil {
		return err
	}
	defer func() { _ = b.Shutdown(ctx) }()

	if err := b.InitializeAuth(ctx); err != nil {
		return err
	}

	b.ConfigureMiddleware()

	if err := b.CreateServerManager(); err != nil {
		return err
	}

	logger.Info("Starting server manager...")
	return b.srvMgr.Run()
}

// configureAuthMiddleware configures authentication and authorization middleware.
func configureAuthMiddleware(opts *Options, jwtAuth *jwt.JWT, rbacAuthz *rbac.RBAC) {
	// Configure JWT authentication middleware
	opts.Server.HTTP.Middleware.Auth = middleware.AuthOptions{
		Authenticator: jwtAuth,
		TokenLookup:   "header:Authorization",
		AuthScheme:    "Bearer",
		SkipPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/register",
			"/health", "/live", "/ready", "/metrics",
		},
	}
	opts.Server.HTTP.Middleware.DisableAuth = false

	// Configure RBAC authorization middleware
	opts.Server.HTTP.Middleware.Authz = middleware.AuthzOptions{
		Authorizer: rbacAuthz,
		SkipPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/register",
			"/api/v1/auth/refresh",
			"/api/v1/auth/logout",
			"/health", "/live", "/ready", "/metrics",
		},
	}
	opts.Server.HTTP.Middleware.DisableAuthz = false

	logger.Info("Auth middleware configured")
}

// configureHealth configures the health check manager.
func configureHealth(opts *Options, dsMgr *datasource.Manager) {
	healthMgr := middleware.GetHealthManager()
	healthMgr.SetVersion(app.GetVersion())

	// Register health checks for all initialized datasources
	// The datasource manager provides a unified way to check health
	healthMgr.RegisterChecker("datasources", func() error {
		if !dsMgr.IsHealthy(context.Background()) {
			return fmt.Errorf("one or more datasources are unhealthy")
		}
		return nil
	})

	// Individual checks can also be registered if needed, but IsHealthy covers all.
	// For more granular reporting in /health endpoint, we could iterate:
	//
	// if opts.MySQL.Host != "" {
	// 	healthMgr.RegisterChecker("mysql", func() error {
	// 		client, err := dsMgr.GetMySQL("primary")
	// 		if err != nil { return err }
	// 		return client.Ping(context.Background())
	// 	})
	// }

	logger.Info("Health checks configured")
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
