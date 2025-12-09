// Package handler provides HTTP and gRPC handlers for the example server.
package handler

import (
	"time"

	"github.com/kart-io/sentinel-x/pkg/auth"
	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/response"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// User represents a user in the system.
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Password string   `json:"-"`
	Roles    []string `json:"roles"`
}

// LoginRequest is the login request body.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthHTTPHandler handles auth HTTP requests.
type AuthHTTPHandler struct {
	authenticator auth.Authenticator
	users         map[string]*User
}

// NewAuthHTTPHandler creates a new auth HTTP handler.
func NewAuthHTTPHandler(authenticator auth.Authenticator) *AuthHTTPHandler {
	// Demo users (in production, use database)
	users := map[string]*User{
		"admin": {
			ID:       "user-001",
			Username: "admin",
			Password: "admin123",
			Roles:    []string{"admin"},
		},
		"editor": {
			ID:       "user-002",
			Username: "editor",
			Password: "editor123",
			Roles:    []string{"editor"},
		},
		"viewer": {
			ID:       "user-003",
			Username: "viewer",
			Password: "viewer123",
			Roles:    []string{"viewer"},
		},
	}

	return &AuthHTTPHandler{
		authenticator: authenticator,
		users:         users,
	}
}

// RegisterRoutes registers auth routes to the router.
func (h *AuthHTTPHandler) RegisterRoutes(router transport.Router) {
	auth := router.Group("/api/v1/auth")
	auth.Handle("POST", "/login", h.Login)
	auth.Handle("POST", "/refresh", h.Refresh)
	auth.Handle("POST", "/logout", h.Logout)
	auth.Handle("GET", "/me", h.Me)
}

// Login handles user login.
func (h *AuthHTTPHandler) Login(ctx transport.Context) {
	var req LoginRequest
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(400, response.Err(errors.ErrInvalidParam.WithMessage("invalid request body")))
		return
	}

	// Find user
	user, ok := h.users[req.Username]
	if !ok || user.Password != req.Password {
		ctx.JSON(401, response.Err(errors.ErrInvalidCredentials))
		return
	}

	// Generate token with extra claims
	token, err := h.authenticator.Sign(ctx.Request(), user.ID,
		auth.WithExtra(map[string]interface{}{
			"username": user.Username,
			"roles":    user.Roles,
		}),
	)
	if err != nil {
		ctx.JSON(500, response.Err(errors.ErrInternal.WithCause(err)))
		return
	}

	ctx.JSON(200, response.Success(map[string]interface{}{
		"access_token": token.GetAccessToken(),
		"token_type":   token.GetTokenType(),
		"expires_in":   token.GetExpiresIn(),
		"expires_at":   token.GetExpiresAt(),
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"roles":    user.Roles,
		},
	}))
}

// Refresh handles token refresh.
func (h *AuthHTTPHandler) Refresh(ctx transport.Context) {
	// Get current token from context
	tokenString := auth.TokenFromContext(ctx.Request())
	if tokenString == "" {
		ctx.JSON(401, response.Err(errors.ErrUnauthorized.WithMessage("no token in context")))
		return
	}

	// Refresh token
	token, err := h.authenticator.Refresh(ctx.Request(), tokenString)
	if err != nil {
		errno := errors.FromError(err)
		ctx.JSON(errno.HTTPStatus(), response.Err(errno))
		return
	}

	ctx.JSON(200, response.Success(map[string]interface{}{
		"access_token": token.GetAccessToken(),
		"token_type":   token.GetTokenType(),
		"expires_in":   token.GetExpiresIn(),
		"expires_at":   token.GetExpiresAt(),
	}))
}

// Logout handles user logout (token revocation).
func (h *AuthHTTPHandler) Logout(ctx transport.Context) {
	// Get current token from context
	tokenString := auth.TokenFromContext(ctx.Request())
	if tokenString == "" {
		ctx.JSON(401, response.Err(errors.ErrUnauthorized.WithMessage("no token in context")))
		return
	}

	// Revoke token
	if err := h.authenticator.Revoke(ctx.Request(), tokenString); err != nil {
		errno := errors.FromError(err)
		ctx.JSON(errno.HTTPStatus(), response.Err(errno))
		return
	}

	ctx.JSON(200, response.Success(map[string]string{
		"message": "logged out successfully",
	}))
}

// Me returns current user info.
func (h *AuthHTTPHandler) Me(ctx transport.Context) {
	claims := auth.ClaimsFromContext(ctx.Request())
	if claims == nil {
		ctx.JSON(401, response.Err(errors.ErrUnauthorized.WithMessage("no claims in context")))
		return
	}

	ctx.JSON(200, response.Success(map[string]interface{}{
		"id":       claims.Subject,
		"username": claims.GetExtraString("username"),
		"roles":    claims.Extra["roles"],
		"issued":   time.Unix(claims.IssuedAt, 0).Format(time.RFC3339),
		"expires":  time.Unix(claims.ExpiresAt, 0).Format(time.RFC3339),
	}))
}

// GetUsers returns all demo users (for RBAC demo).
func (h *AuthHTTPHandler) GetUsers() map[string]*User {
	return h.users
}
