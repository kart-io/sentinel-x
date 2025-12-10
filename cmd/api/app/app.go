// Package app provides the API server application.
package app

import (
	"context"
	"fmt"

	"gorm.io/gorm"

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
	goredis "github.com/redis/go-redis/v9"
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
func Run(opts *Options) error {
	// Initialize logger first
	if err := opts.Log.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() { _ = logger.Flush() }()

	// Print startup banner
	printBanner(opts)

	// Log server startup
	logger.Infow("Starting Sentinel-X API server",
		"app", appName,
		"version", app.GetVersion(),
		"mode", opts.Server.Mode.String(),
	)

	// Initialize datasource manager
	dsMgr := datasource.NewManager()
	datasource.SetGlobal(dsMgr) // Set as global if needed by other components
	defer dsMgr.CloseAll()

	// Register datasources
	if opts.MySQL.Host != "" {
		if err := dsMgr.RegisterMySQL("primary", opts.MySQL); err != nil {
			return fmt.Errorf("failed to register mysql: %w", err)
		}
	}

	if opts.Redis.Host != "" {
		if err := dsMgr.RegisterRedis("cache", opts.Redis); err != nil {
			return fmt.Errorf("failed to register redis: %w", err)
		}
	}

	// Initialize all datasources
	ctx := context.Background()
	if err := dsMgr.InitAll(ctx); err != nil {
		return fmt.Errorf("failed to initialize datasources: %w", err)
	}
	logger.Info("Datasources initialized successfully")

	// Get clients for dependency injection
	var db *gorm.DB
	if opts.MySQL.Host != "" {
		mysqlClient, err := dsMgr.GetMySQL("primary")
		if err != nil {
			return fmt.Errorf("failed to get mysql client: %w", err)
		}
		db = mysqlClient.DB()
	}

	var rdb *goredis.Client
	if opts.Redis.Host != "" {
		redisClient, err := dsMgr.GetRedis("cache")
		if err != nil {
			return fmt.Errorf("failed to get redis client: %w", err)
		}
		rdb = redisClient.Client()
	}

	// Prevent unused variable error until services are registered
	_ = db
	_ = rdb

	// Configure health checks
	configureHealth(opts, dsMgr)

	// Initialize authentication and authorization
	jwtAuth, rbacAuthz, err := initAuth(opts)
	if err != nil {
		return fmt.Errorf("failed to initialize auth: %w", err)
	}

	// Configure auth middleware if enabled
	if jwtAuth != nil && rbacAuthz != nil {
		configureAuthMiddleware(opts, jwtAuth, rbacAuthz)
	}

	// Create server manager
	mgr := server.NewManager(
		server.WithMode(opts.Server.Mode),
		server.WithHTTPOptions(opts.Server.HTTP),
		server.WithGRPCOptions(opts.Server.GRPC),
		server.WithShutdownTimeout(opts.Server.ShutdownTimeout),
	)

	// TODO: Register your services here
	// Example:
	// userSvc := userservice.NewService(db, rdb)
	// userHTTPHandler := handler.NewUserHTTPHandler(userSvc)
	// userGRPCHandler := handler.NewUserGRPCHandler(userSvc)
	// _ = mgr.RegisterService(userSvc, userHTTPHandler, &transport.GRPCServiceDesc{
	//     ServiceDesc: &apiv1.UserService_ServiceDesc,
	//     ServiceImpl: userGRPCHandler,
	// })

	logger.Info("All services registered successfully")

	// Run server with graceful shutdown
	logger.Info("Starting server manager...")
	return mgr.Run()
}

// initAuth initializes JWT authentication and RBAC authorization.
func initAuth(opts *Options) (*jwt.JWT, *rbac.RBAC, error) {
	if opts.JWT.DisableAuth {
		logger.Info("Authentication disabled")
		return nil, nil, nil
	}

	// Create JWT authenticator
	tokenStore := jwt.NewMemoryStore()
	jwtAuth, err := jwt.New(
		jwt.WithOptions(opts.JWT),
		jwt.WithStore(tokenStore),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create JWT authenticator: %w", err)
	}

	// Create RBAC authorizer
	rbacAuthz := rbac.New()

	// Define default roles with permissions
	// Admin role - full access
	_ = rbacAuthz.AddRole("admin",
		authz.NewPermission("*", "*"),
	)

	// User role - basic access
	_ = rbacAuthz.AddRole("user",
		authz.NewPermission("user", "read"),
		authz.NewPermission("user", "update"),
	)

	// Guest role - read-only access
	_ = rbacAuthz.AddRole("guest",
		authz.NewPermission("*", "read"),
	)

	logger.Infow("Authentication and authorization initialized",
		"jwt_issuer", opts.JWT.Issuer,
		"jwt_expired", opts.JWT.Expired,
		"roles", []string{"admin", "user", "guest"},
	)

	return jwtAuth, rbacAuthz, nil
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
