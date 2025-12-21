// Package router provides the auth service router.
package router

import (
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/auth/biz"
	"github.com/kart-io/sentinel-x/internal/auth/handler"
	"github.com/kart-io/sentinel-x/internal/auth/store"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
)

// Register registers the auth service routes.
func Register(mgr *server.Manager, jwtAuth *jwt.JWT, ds *datasource.Manager) error {
	logger.Info("Registering auth routes...")
	// Initialize Store
	storeFactory, err := store.GetFactory(ds)
	if err != nil {
		logger.Errorf("failed to get store factory: %s", err.Error())
		return err
	}


	// Initialize Biz and Handlers
	authBiz := biz.NewAuthService(jwtAuth, storeFactory)
	authHandler := handler.NewAuthHandler(authBiz)
	// HTTP Server
	if httpServer := mgr.HTTPServer(); httpServer != nil {
		router := httpServer.Router()

		auth := router.Group("/auth")
		{
			auth.Handle("POST", "/login", authHandler.Login)
			auth.Handle("POST", "/logout", authHandler.Logout)
			auth.Handle("POST", "/register", authHandler.Register)
		}

		logger.Info("Auth HTTP routes registered")
	}

	// gRPC Server
	if grpcServer := mgr.GRPCServer(); grpcServer != nil {
		logger.Info("Auth gRPC services registered (placeholder)")
	}

	return nil
}
