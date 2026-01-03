// Package apisvc provides the API Service server implementation.
package apisvc

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/api/handler"
	// Register adapters
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
)

// Name is the name of the application.
const Name = "sentinel-api"

// Config contains application-related configurations.
type Config struct {
	HTTPOptions      *httpopts.Options
	GRPCOptions      *grpcopts.Options
	LogOptions       *logopts.Options
	JWTOptions       *jwtopts.Options
	MySQLOptions     *mysqlopts.Options
	RedisOptions     *redisopts.Options
	RecoveryOptions  *middlewareopts.RecoveryOptions
	RequestIDOptions *middlewareopts.RequestIDOptions
	LoggerOptions    *middlewareopts.LoggerOptions
	CORSOptions      *middlewareopts.CORSOptions
	TimeoutOptions   *middlewareopts.TimeoutOptions
	HealthOptions    *middlewareopts.HealthOptions
	MetricsOptions   *middlewareopts.MetricsOptions
	PprofOptions     *middlewareopts.PprofOptions
	ShutdownTimeout  time.Duration
}

// Server represents the API server.
type Server struct {
	srv *server.Manager
}

// NewServer initializes and returns a new Server instance.
func (cfg *Config) NewServer(ctx context.Context) (*Server, error) {
	printBanner(cfg)

	// 1. 初始化日志
	cfg.LogOptions.AddInitialField("service.name", Name)
	cfg.LogOptions.AddInitialField("service.version", app.GetVersion())
	if err := cfg.LogOptions.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Info("Starting API service...")

	// 2. 初始化 JWT（如果配置了）
	// 注意：API 服务当前使用 middleware 自动处理 JWT，此处预留
	_ = cfg.JWTOptions // 忽略未使用警告

	// 3. 初始化 Handler
	demoHandler := handler.NewDemoHandler()
	logger.Info("Handlers initialized")

	// 4. 初始化服务器
	serverManager := server.NewManager(
		server.WithHTTPOptions(cfg.HTTPOptions),
		server.WithGRPCOptions(cfg.GRPCOptions),
		server.WithMiddleware(cfg.GetMiddlewareOptions()),
		server.WithShutdownTimeout(cfg.ShutdownTimeout),
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

	logger.Info("API service is ready")
	return &Server{srv: serverManager}, nil
}

// Run starts the server and listens for termination signals.
func (s *Server) Run(ctx context.Context) error {
	return s.srv.Run()
}

// GetMiddlewareOptions builds middleware options from individual configurations.
func (cfg *Config) GetMiddlewareOptions() *middlewareopts.Options {
	return &middlewareopts.Options{
		Recovery:  cfg.RecoveryOptions,
		RequestID: cfg.RequestIDOptions,
		Logger:    cfg.LoggerOptions,
		CORS:      cfg.CORSOptions,
		Timeout:   cfg.TimeoutOptions,
		Health:    cfg.HealthOptions,
		Metrics:   cfg.MetricsOptions,
		Pprof:     cfg.PprofOptions,
	}
}

func printBanner(cfg *Config) {
	mw := cfg.GetMiddlewareOptions()

	fmt.Println("===========================================")
	fmt.Println("  Sentinel-X API Server")
	fmt.Println("===========================================")
	fmt.Printf("Version: %s\n", app.GetVersion())

	if cfg.HTTPOptions != nil {
		fmt.Printf("HTTP: %s (adapter: %s)\n", cfg.HTTPOptions.Addr, cfg.HTTPOptions.Adapter)
	}
	if cfg.GRPCOptions != nil {
		fmt.Printf("gRPC: %s\n", cfg.GRPCOptions.Addr)
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Configuration:")
	fmt.Printf("  Logger: level=%s, format=%s\n", cfg.LogOptions.Level, cfg.LogOptions.Format)
	if cfg.MySQLOptions.Host != "" {
		fmt.Printf("  Database: %s:%d/%s\n", cfg.MySQLOptions.Host, cfg.MySQLOptions.Port, cfg.MySQLOptions.Database)
	}
	if cfg.RedisOptions.Host != "" {
		fmt.Printf("  Redis: %s:%d (db=%d)\n", cfg.RedisOptions.Host, cfg.RedisOptions.Port, cfg.RedisOptions.Database)
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Middleware:")
	if mw.IsEnabled(middlewareopts.MiddlewareRecovery) {
		fmt.Println("  - Recovery")
	}
	if mw.IsEnabled(middlewareopts.MiddlewareRequestID) {
		fmt.Println("  - RequestID")
	}
	if mw.IsEnabled(middlewareopts.MiddlewareLogger) {
		fmt.Println("  - Logger")
	}
	if mw.IsEnabled(middlewareopts.MiddlewareCORS) {
		fmt.Println("  - CORS")
	}
	if mw.IsEnabled(middlewareopts.MiddlewareTimeout) {
		fmt.Println("  - Timeout")
	}
	if mw.IsEnabled(middlewareopts.MiddlewareHealth) {
		fmt.Println("  - Health")
	}
	if mw.IsEnabled(middlewareopts.MiddlewareMetrics) {
		fmt.Println("  - Metrics")
	}
	if mw.IsEnabled(middlewareopts.MiddlewarePprof) {
		fmt.Println("  - Pprof")
	}
	if mw.IsEnabled(middlewareopts.MiddlewareAuth) {
		fmt.Println("  - Auth (JWT)")
	}
	if mw.IsEnabled(middlewareopts.MiddlewareAuthz) {
		fmt.Println("  - Authz (RBAC)")
	}

	if cfg.HTTPOptions != nil && mw.IsEnabled(middlewareopts.MiddlewareHealth) {
		fmt.Println("-------------------------------------------")
		fmt.Println("Endpoints:")
		fmt.Printf("  Health: http://localhost%s%s\n", cfg.HTTPOptions.Addr, mw.Health.Path)
		fmt.Printf("  Liveness: http://localhost%s%s\n", cfg.HTTPOptions.Addr, mw.Health.LivenessPath)
		fmt.Printf("  Readiness: http://localhost%s%s\n", cfg.HTTPOptions.Addr, mw.Health.ReadinessPath)
		if mw.IsEnabled(middlewareopts.MiddlewareMetrics) {
			fmt.Printf("  Metrics: http://localhost%s%s\n", cfg.HTTPOptions.Addr, mw.Metrics.Path)
		}
		if mw.IsEnabled(middlewareopts.MiddlewarePprof) {
			fmt.Printf("  Pprof: http://localhost%s%s/\n", cfg.HTTPOptions.Addr, mw.Pprof.Prefix)
		}
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Press Ctrl+C to gracefully shutdown")
	fmt.Println()
}
