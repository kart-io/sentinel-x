// Package router 提供 API 服务的路由注册。
package router

import (
	"github.com/kart-io/logger"
	_ "github.com/kart-io/sentinel-x/docs/swagger" // swagger docs
	"github.com/kart-io/sentinel-x/internal/api/handler"
	"github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	httpserver "github.com/kart-io/sentinel-x/pkg/infra/server/transport/http"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

		// Swagger UI - 访问地址: http://localhost:8100/swagger/index.html
		registerSwagger(httpServer)

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
func registerSwagger(httpServer *httpserver.Server) {
	adapter := httpServer.Adapter()
	if adapter == nil {
		logger.Warn("HTTP adapter is nil, skip swagger registration")
		return
	}

	// 获取底层 Gin bridge
	bridge := adapter.Bridge()
	if bridge == nil {
		logger.Warn("HTTP bridge is nil, skip swagger registration")
		return
	}

	// 类型断言获取 Gin bridge
	ginBridge, ok := bridge.(*gin.Bridge)
	if !ok {
		logger.Warn("HTTP bridge is not Gin, skip swagger registration")
		return
	}

	// 直接在 Gin engine 上注册 Swagger 路由
	ginBridge.Engine().GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	logger.Info("Swagger routes registered on Gin engine")
}
