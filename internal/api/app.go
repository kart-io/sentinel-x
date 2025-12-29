// Package app provides the API server application.
package app

import (
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/api/handler"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/server"

	// Import adapters
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
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
	printBanner(opts)

	// 1. 初始化日志
	opts.Log.AddInitialField("service.name", appName)
	opts.Log.AddInitialField("service.version", app.GetVersion())
	if err := opts.Log.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Info("Starting API service...")

	// 2. 初始化 JWT（如果配置了）
	// 注意：API 服务当前使用 middleware 自动处理 JWT，此处预留
	_ = opts.JWT // 忽略未使用警告

	// 3. 初始化 Handler
	demoHandler := handler.NewDemoHandler()
	logger.Info("Handlers initialized")

	// 4. 初始化服务器
	serverManager := server.NewManager(
		server.WithMode(opts.Server.Mode),
		server.WithHTTPOptions(opts.Server.HTTP),
		server.WithGRPCOptions(opts.Server.GRPC),
		server.WithShutdownTimeout(opts.Server.ShutdownTimeout),
	)

	// 5. 注册路由
	if httpServer := serverManager.HTTPServer(); httpServer != nil {
		router := httpServer.Router()

		// Serve static files for OpenAPI specs
		router.Static("/openapi", "api/openapi")

		// API v1 路由组
		v1 := router.Group("/api/v1")
		{
			// 公开路由（无需认证）
			v1.Handle("GET", "/hello", demoHandler.Hello)

			// 受保护路由（需要 JWT 认证）
			v1.Handle("GET", "/protected", demoHandler.Protected)
			v1.Handle("GET", "/profile", demoHandler.Profile)
		}

		logger.Info("HTTP routes registered")
		logger.Info("Swagger UI available at: http://localhost:8100/swagger/index.html")
	}

	// gRPC Server
	if serverManager.GRPCServer() != nil {
		logger.Info("gRPC services registered (placeholder)")
	}

	// 6. 启动服务器
	logger.Info("API service is ready")
	return serverManager.Run()
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
