package router

import (
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/internal/user-center/handler"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
)

// Register registers the user center routes and services.
func Register(mgr *server.Manager, jwtAuth *jwt.JWT, ds *datasource.Manager) error {
	logger.Info("Registering user center routes...")

	// Initialize Store
	storeFactory, err := store.GetFactory(ds)
	if err != nil {
		return err
	}

	// Initialize Biz and Handlers
	userBiz := biz.NewUserService(storeFactory)
	authBiz := biz.NewAuthService(jwtAuth, storeFactory)

	userHandler := handler.NewUserHandler(userBiz)
	authHandler := handler.NewAuthHandler(authBiz)

	// HTTP Server
	if httpServer := mgr.HTTPServer(); httpServer != nil {
		router := httpServer.Router()

		// Auth Routes
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
				users.Handle("GET", "/:username", userHandler.Get)
				users.Handle("PUT", "/:username", userHandler.Update)
				users.Handle("DELETE", "/:username", userHandler.Delete)
				users.Handle("POST", "/:username/password", userHandler.ChangePassword)
			}
		}

		logger.Info("HTTP routes registered")
	}

	// gRPC Server
	if grpcServer := mgr.GRPCServer(); grpcServer != nil {
		// Register gRPC services here
		// example:
		// pb.RegisterUserServiceServer(grpcServer.Server, handler.NewUserService())

		// For now, let's just log
		logger.Info("gRPC services registered")
	}

	return nil
}
