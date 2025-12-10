// Package app provides the API server application.
package app

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/app"
	"github.com/kart-io/sentinel-x/pkg/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/authz"
	"github.com/kart-io/sentinel-x/pkg/authz/rbac"
	// Import bridges to register them
	_ "github.com/kart-io/sentinel-x/pkg/bridge/echo"
	_ "github.com/kart-io/sentinel-x/pkg/bridge/gin"
	"github.com/kart-io/sentinel-x/pkg/middleware"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	"github.com/kart-io/sentinel-x/pkg/server"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

	// Initialize database connections
	ctx := context.Background()
	db, err := initDatabase(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	logger.Info("Database initialized successfully")

	// Initialize Redis
	rdb, err := initRedis(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}
	logger.Info("Redis initialized successfully")

	// Configure health checks
	configureHealth(opts, db, rdb)

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
		serveropts.WithMode(opts.Server.Mode),
		serveropts.WithHTTPOptions(opts.Server.HTTP),
		serveropts.WithGRPCOptions(opts.Server.GRPC),
		serveropts.WithShutdownTimeout(opts.Server.ShutdownTimeout),
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

// initDatabase initializes the MySQL database connection.
func initDatabase(ctx context.Context, opts *Options) (*gorm.DB, error) {
	if opts.MySQL.Host == "" {
		logger.Warn("MySQL host not configured, skipping database initialization")
		return nil, nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		opts.MySQL.Username,
		opts.MySQL.Password,
		opts.MySQL.Host,
		opts.MySQL.Port,
		opts.MySQL.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(opts.MySQL.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(opts.MySQL.MaxOpenConnections)
	sqlDB.SetConnMaxLifetime(opts.MySQL.MaxConnectionLifeTime)

	// Test connection
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Infow("Database connection established",
		"host", opts.MySQL.Host,
		"database", opts.MySQL.Database,
		"max_idle_conns", opts.MySQL.MaxIdleConnections,
		"max_open_conns", opts.MySQL.MaxOpenConnections,
	)

	return db, nil
}

// initRedis initializes the Redis connection.
func initRedis(ctx context.Context, opts *Options) (*redis.Client, error) {
	if opts.Redis.Host == "" {
		logger.Warn("Redis host not configured, skipping Redis initialization")
		return nil, nil
	}

	addr := fmt.Sprintf("%s:%d", opts.Redis.Host, opts.Redis.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     opts.Redis.Password,
		DB:           opts.Redis.Database,
		MaxRetries:   opts.Redis.MaxRetries,
		PoolSize:     opts.Redis.PoolSize,
		MinIdleConns: opts.Redis.MinIdleConns,
	})

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger.Infow("Redis connection established",
		"addr", addr,
		"db", opts.Redis.Database,
		"pool_size", opts.Redis.PoolSize,
	)

	return rdb, nil
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
func configureHealth(opts *Options, db *gorm.DB, rdb *redis.Client) {
	healthMgr := middleware.GetHealthManager()
	healthMgr.SetVersion(app.GetVersion())

	// Database health check
	if db != nil {
		healthMgr.RegisterChecker("database", func() error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Ping()
		})
	}

	// Redis health check
	if rdb != nil {
		healthMgr.RegisterChecker("redis", func() error {
			return rdb.Ping(context.Background()).Err()
		})
	}

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
