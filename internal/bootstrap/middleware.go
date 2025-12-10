package bootstrap

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	serveropts "github.com/kart-io/sentinel-x/pkg/infra/server"
)

// MiddlewareInitializer handles middleware configuration.
type MiddlewareInitializer struct {
	serverOpts        *serveropts.Options
	authInitializer   *AuthInitializer
	datasourceManager *datasource.Manager
	appVersion        string
}

// NewMiddlewareInitializer creates a new MiddlewareInitializer.
func NewMiddlewareInitializer(
	serverOpts *serveropts.Options,
	authInitializer *AuthInitializer,
	datasourceManager *datasource.Manager,
	appVersion string,
) *MiddlewareInitializer {
	return &MiddlewareInitializer{
		serverOpts:        serverOpts,
		authInitializer:   authInitializer,
		datasourceManager: datasourceManager,
		appVersion:        appVersion,
	}
}

// Name returns the name of the initializer.
func (mi *MiddlewareInitializer) Name() string {
	return "middleware"
}

// Initialize configures all middleware components.
func (mi *MiddlewareInitializer) Initialize(ctx context.Context) error {
	mi.configureHealth()

	if mi.authInitializer.IsEnabled() {
		mi.configureAuth()
	}

	return nil
}

// configureHealth configures the health check manager.
func (mi *MiddlewareInitializer) configureHealth() {
	healthMgr := middleware.GetHealthManager()
	healthMgr.SetVersion(mi.appVersion)

	// Register health checks for all initialized datasources
	// The datasource manager provides a unified way to check health
	healthMgr.RegisterChecker("datasources", func() error {
		if !mi.datasourceManager.IsHealthy(context.Background()) {
			return fmt.Errorf("one or more datasources are unhealthy")
		}
		return nil
	})

	logger.Info("Health checks configured")
}

// configureAuth configures authentication and authorization middleware.
func (mi *MiddlewareInitializer) configureAuth() {
	jwtAuth := mi.authInitializer.GetJWT()
	rbacAuthz := mi.authInitializer.GetRBAC()

	if jwtAuth == nil || rbacAuthz == nil {
		logger.Warn("Auth initializer not properly configured, skipping auth middleware setup")
		return
	}

	// Configure JWT authentication middleware
	mi.serverOpts.HTTP.Middleware.Auth = middleware.AuthOptions{
		Authenticator: jwtAuth,
		TokenLookup:   "header:Authorization",
		AuthScheme:    "Bearer",
		SkipPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/register",
			"/health", "/live", "/ready", "/metrics",
		},
	}
	mi.serverOpts.HTTP.Middleware.DisableAuth = false

	// Configure RBAC authorization middleware
	mi.serverOpts.HTTP.Middleware.Authz = middleware.AuthzOptions{
		Authorizer: rbacAuthz,
		SkipPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/register",
			"/api/v1/auth/refresh",
			"/api/v1/auth/logout",
			"/health", "/live", "/ready", "/metrics",
		},
	}
	mi.serverOpts.HTTP.Middleware.DisableAuthz = false

	logger.Info("Auth middleware configured")
}
