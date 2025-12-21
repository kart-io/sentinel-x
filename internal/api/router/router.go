// Package router 提供 API 服务的路由注册。
package router

import (
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/api/handler"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
)

// Register 注册 API 服务的路由。
func Register(mgr *server.Manager, _ *jwt.JWT, _ *datasource.Manager) error {
	logger.Info("Registering API routes...")

	// 初始化处理器
	demoHandler := handler.NewDemoHandler()

	// HTTP Server
	if httpServer := mgr.HTTPServer(); httpServer != nil {
		router := httpServer.Router()

		// Serve static files for OpenAPI specs (proto generated)
		router.Static("/openapi", "api/openapi")

		// API v1 路由组
		v1 := router.Group("/api/v1")
		{
			// 公开路由（无需认证）
			// 这些路径已在配置文件中添加到 skip-paths
			v1.Handle("GET", "/hello", demoHandler.Hello)

			// 受保护路由（需要 JWT 认证）
			// JWT 验证由全局 Auth 中间件处理
			v1.Handle("GET", "/protected", demoHandler.Protected)
			v1.Handle("GET", "/profile", demoHandler.Profile)
		}

		logger.Info("API HTTP routes registered")
		logger.Info("Swagger UI available at: http://localhost:8100/swagger/index.html")
	}

	// gRPC Server
	if grpcServer := mgr.GRPCServer(); grpcServer != nil {
		logger.Info("API gRPC services registered (placeholder)")
	}

	return nil
}

// registerSwagger 注册 Swagger UI 路由。
// 直接在 Gin engine 上注册，避免抽象层的路径处理问题。
