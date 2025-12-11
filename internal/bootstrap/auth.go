package bootstrap

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	jwtopts "github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/security/authz/rbac"
)

// AuthInitializer handles authentication and authorization initialization.
type AuthInitializer struct {
	jwtOpts           *jwtopts.Options
	jwtAuth           *jwt.JWT
	rbac              *rbac.RBAC
	datasourceManager *datasource.Manager
}

// NewAuthInitializer creates a new AuthInitializer.
func NewAuthInitializer(jwtOpts *jwtopts.Options) *AuthInitializer {
	return &AuthInitializer{
		jwtOpts: jwtOpts,
	}
}

// Name returns the name of the initializer.
func (ai *AuthInitializer) Name() string {
	return "auth"
}

// Initialize initializes JWT authentication and RBAC authorization.
func (ai *AuthInitializer) Initialize(ctx context.Context) error {
	if ai.jwtOpts.DisableAuth {
		logger.Warn("Authentication disabled - tokens will be issued but not verified by middleware")
	}

	// Initialize JWT authenticator
	// Use Redis store if available, otherwise fallback to memory store
	var tokenStore jwt.Store
	if ai.datasourceManager != nil {
		if redisClient, err := ai.datasourceManager.GetRedis("cache"); err == nil && redisClient != nil {
			tokenStore = jwt.NewRedisStore(redisClient, "jwt:revoked:")
			logger.Info("Using Redis for JWT token revocation")
		}
	}

	if tokenStore == nil {
		tokenStore = jwt.NewMemoryStore()
		logger.Warn("Using in-memory store for JWT token revocation (not suitable for distributed deployments)")
	}

	jwtAuth, err := jwt.New(
		jwt.WithOptions(ai.jwtOpts),
		jwt.WithStore(tokenStore),
	)
	if err != nil {
		return fmt.Errorf("failed to create JWT authenticator: %w", err)
	}

	// Initialize RBAC authorizer
	rbacAuthz := rbac.New()

	// Configure default roles
	if err := ai.configureDefaultRoles(rbacAuthz); err != nil {
		return fmt.Errorf("failed to configure default roles: %w", err)
	}

	ai.jwtAuth = jwtAuth
	ai.rbac = rbacAuthz

	logger.Infow("Authentication and authorization initialized",
		"jwt_issuer", ai.jwtOpts.Issuer,
		"jwt_expired", ai.jwtOpts.Expired,
		"roles", []string{"admin", "user", "guest"},
	)

	return nil
}

// configureDefaultRoles configures the default RBAC roles.
func (ai *AuthInitializer) configureDefaultRoles(rbacAuthz *rbac.RBAC) error {
	// Admin role with full access
	if err := rbacAuthz.AddRole("admin", authz.NewPermission("*", "*")); err != nil {
		return fmt.Errorf("failed to add admin role: %w", err)
	}

	// User role with limited access
	if err := rbacAuthz.AddRole("user",
		authz.NewPermission("user", "read"),
		authz.NewPermission("user", "update"),
	); err != nil {
		return fmt.Errorf("failed to add user role: %w", err)
	}

	// Guest role with read-only access
	if err := rbacAuthz.AddRole("guest", authz.NewPermission("*", "read")); err != nil {
		return fmt.Errorf("failed to add guest role: %w", err)
	}

	return nil
}

// GetJWT returns the JWT authenticator.
func (ai *AuthInitializer) GetJWT() *jwt.JWT {
	return ai.jwtAuth
}

// GetRBAC returns the RBAC authorizer.
func (ai *AuthInitializer) GetRBAC() *rbac.RBAC {
	return ai.rbac
}

// IsEnabled returns true if authentication is enabled.
func (ai *AuthInitializer) IsEnabled() bool {
	return !ai.jwtOpts.DisableAuth
}
