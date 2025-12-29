// Package router provides user-center routing.
package router

import (
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/user-center/handler"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/utils/validator"
)

// Register registers the user center routes and services.
func Register(mgr *server.Manager, jwtAuth *jwt.JWT, userHandler *handler.UserHandler, roleHandler *handler.RoleHandler, authHandler *handler.AuthHandler) error {
	logger.Info("Registering user center routes...")

	// HTTP Server
	if httpServer := mgr.HTTPServer(); httpServer != nil {
		// 使用全局验证器，确保统一的验证规则和 i18n
		httpServer.SetValidator(validator.Global())

		router := httpServer.Router()

		//  Auth Routes
		auth := router.Group("/auth")
		{
			auth.Handle("POST", "/login", authHandler.Login)
			auth.Handle("POST", "/logout", authHandler.Logout)
			auth.Handle("POST", "/register", authHandler.Register)

			// Protected Auth Routes
			authProtected := auth.Group("")
			authProtected.Use(middleware.Auth(middleware.AuthWithAuthenticator(jwtAuth)))
			{
				authProtected.Handle("GET", "/me", userHandler.GetProfile)
			}
		}

		// User Routes
		v1 := router.Group("/v1")
		{
			// Public User Routes (Registration)
			v1.Handle("POST", "/users", userHandler.Create)

			// Protected User Routes
			users := v1.Group("/users")
			users.Use(middleware.Auth(middleware.AuthWithAuthenticator(jwtAuth)))
			{
				users.Handle("GET", "", userHandler.List)
				users.Handle("POST", "/batch-delete", userHandler.BatchDelete)
				users.Handle("GET", "/detail", userHandler.Get)
				users.Handle("PUT", "/update", userHandler.Update)
				users.Handle("DELETE", "/delete", userHandler.Delete)
				users.Handle("POST", "/password", userHandler.UpdatePassword)

				// User Role Assignment
				users.Handle("POST", "/roles", roleHandler.AssignUserRole)
				users.Handle("GET", "/roles", roleHandler.ListUserRoles)
			}

			// Role Routes
			roles := v1.Group("/roles")
			roles.Use(middleware.Auth(middleware.AuthWithAuthenticator(jwtAuth)))
			{
				roles.Handle("POST", "", roleHandler.Create)
				roles.Handle("GET", "", roleHandler.List)
				roles.Handle("GET", "/detail", roleHandler.Get)
				roles.Handle("PUT", "/update", roleHandler.Update)
				roles.Handle("DELETE", "/delete", roleHandler.Delete)
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
