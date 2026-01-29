// Package router provides user-center routing.
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/user-center/handler"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	authmw "github.com/kart-io/sentinel-x/pkg/infra/middleware/auth"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/utils/validator"
)

// Register registers the user center routes and services.
func Register(mgr *server.Manager, jwtAuth *jwt.JWT, userHandler *handler.UserHandler, roleHandler *handler.RoleHandler, authHandler *handler.AuthHandler) error {
	logger.Info("Registering user center routes...")

	// Create auth options for all routes
	authOpts := mwopts.NewAuthOptions()

	// HTTP Server
	if httpServer := mgr.HTTPServer(); httpServer != nil {
		// 使用全局验证器，确保统一的验证规则和 i18n
		httpServer.SetValidator(validator.Global())

		//nolint:staticcheck // ST1023: 需要显式类型声明，httpServer.Engine() 返回接口类型
		var engine *gin.Engine = httpServer.Engine()

		//  Auth Routes
		auth := engine.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/captcha", authHandler.GetCaptcha)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/register", authHandler.Register)

			// Protected Auth Routes
			authProtected := auth.Group("")
			authProtected.Use(authmw.AuthWithOptions(*authOpts, jwtAuth, nil, nil))
			{
				authProtected.GET("/me", userHandler.GetProfile)
			}
		}

		// User Routes
		v1Group := engine.Group("/v1")
		{
			// Public User Routes (Registration)
			v1Group.POST("/users", userHandler.Create)

			// Protected User Routes
			users := v1Group.Group("/users")
			users.Use(authmw.AuthWithOptions(*authOpts, jwtAuth, nil, nil))
			{
				users.GET("", userHandler.List)
				users.POST("/batch-delete", userHandler.BatchDelete)
				users.GET("/detail", userHandler.Get)
				users.PUT("", userHandler.Update)
				users.DELETE("", userHandler.Delete)
				users.POST("/password", userHandler.UpdatePassword)

				// User Role Assignment
				users.POST("/roles", roleHandler.AssignUserRole)
				users.GET("/roles", roleHandler.ListUserRoles)
			}

			// Role Routes
			roles := v1Group.Group("/roles")
			roles.Use(authmw.AuthWithOptions(*authOpts, jwtAuth, nil, nil))
			{
				roles.POST("", roleHandler.Create)
				roles.GET("", roleHandler.List)
				roles.GET("/detail", roleHandler.Get)
				roles.PUT("", roleHandler.Update)
				roles.DELETE("", roleHandler.Delete)

				// Role Permissions
				roles.POST("/permissions", roleHandler.AssignPermission)
				roles.DELETE("/permissions", roleHandler.RemovePermission)
			}
		}

		logger.Info("HTTP routes registered")

	}

	// gRPC Server
	if grpcServer := mgr.GRPCServer(); grpcServer != nil {
		// Register gRPC services here
		v1.RegisterUserServiceServer(grpcServer.Server(), userHandler)
		v1.RegisterRoleServiceServer(grpcServer.Server(), roleHandler)
		logger.Info("gRPC services registered")
	}

	return nil
}
