// Package router provides user-center routing.
package router

import (
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/internal/user-center/handler"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/utils/validator"
)

// Register registers the user center routes and services.
func Register(mgr *server.Manager, jwtAuth *jwt.JWT, ds *datasource.Manager) error {
	logger.Info("Registering user center routes...")

	// Initialize Store
	storeFactory, err := store.GetFactory(ds)
	if err != nil {
		return err
	}

	// Auto Migration
	if err := storeFactory.AutoMigrate(); err != nil {
		return err
	}

	// Initialize Biz and Handlers
	userBiz := biz.NewUserService(storeFactory)
	roleBiz := biz.NewRoleService(storeFactory)
	authBiz := biz.NewAuthService(jwtAuth, storeFactory)

	userHandler := handler.NewUserHandler(userBiz, roleBiz, authBiz)
	roleHandler := handler.NewRoleHandler(roleBiz)
	authHandler := handler.NewAuthHandler(authBiz)

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
				users.Handle("GET", "/:username", userHandler.Get)
				users.Handle("PUT", "/:username", userHandler.Update)
				users.Handle("DELETE", "/:username", userHandler.Delete)
				users.Handle("POST", "/:username/password", userHandler.UpdatePassword)

				// User Role Assignment
				users.Handle("POST", "/:username/roles", roleHandler.AssignUserRole)
				users.Handle("GET", "/:username/roles", roleHandler.ListUserRoles)
			}

			// Role Routes
			roles := v1.Group("/roles")
			roles.Use(middleware.Auth(middleware.AuthWithAuthenticator(jwtAuth)))
			{
				roles.Handle("POST", "", roleHandler.Create)
				roles.Handle("GET", "", roleHandler.List)
				roles.Handle("GET", "/:code", roleHandler.Get)
				roles.Handle("PUT", "/:code", roleHandler.Update)
				roles.Handle("DELETE", "/:code", roleHandler.Delete)
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
