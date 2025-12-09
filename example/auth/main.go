// Package main demonstrates the Auth/Authz module of Sentinel-X.
//
// This example shows:
//   - JWT-based authentication
//   - RBAC-based authorization
//   - HTTP middleware integration
//   - Token generation, verification, refresh, and revocation
//
// Run the example:
//
//	go run example/auth/main.go
//
// Test endpoints:
//
//	# Login to get token
//	curl -X POST http://localhost:8082/api/v1/auth/login \
//	  -H "Content-Type: application/json" \
//	  -d '{"username":"admin","password":"admin123"}'
//
//	# Access protected resource with token
//	curl http://localhost:8082/api/v1/users \
//	  -H "Authorization: Bearer <token>"
//
//	# Refresh token
//	curl -X POST http://localhost:8082/api/v1/auth/refresh \
//	  -H "Authorization: Bearer <token>"
//
//	# Logout (revoke token)
//	curl -X POST http://localhost:8082/api/v1/auth/logout \
//	  -H "Authorization: Bearer <token>"
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kart-io/sentinel-x/pkg/auth"
	"github.com/kart-io/sentinel-x/pkg/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/authz"
	"github.com/kart-io/sentinel-x/pkg/authz/rbac"
	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/middleware"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/jwt"
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

// Demo users (in production, use database)
var users = map[string]*User{
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

func main() {
	printBanner()

	// Initialize JWT authenticator
	jwtOpts := jwtopts.NewOptions()
	jwtOpts.Key = "sentinel-x-demo-secret-key-32char!"
	jwtOpts.Expired = 15 * time.Minute
	jwtOpts.MaxRefresh = 24 * time.Hour
	jwtOpts.Issuer = "sentinel-x-auth-demo"

	tokenStore := jwt.NewMemoryStore()
	jwtAuth, err := jwt.New(
		jwt.WithOptions(jwtOpts),
		jwt.WithStore(tokenStore),
	)
	if err != nil {
		fmt.Printf("Failed to create JWT authenticator: %v\n", err)
		os.Exit(1)
	}

	// Initialize RBAC authorizer
	rbacAuthz := rbac.New()

	// Define roles with permissions
	// Admin: full access to all resources
	rbacAuthz.AddRole("admin",
		authz.NewPermission("*", "*"),
	)

	// Editor: can read, create, and update users
	rbacAuthz.AddRole("editor",
		authz.NewPermission("users", "read"),
		authz.NewPermission("users", "create"),
		authz.NewPermission("users", "update"),
		authz.NewPermission("posts", "*"),
	)

	// Viewer: can only read users
	rbacAuthz.AddRole("viewer",
		authz.NewPermission("users", "read"),
		authz.NewPermission("posts", "read"),
	)

	// Assign roles to users
	for _, user := range users {
		for _, role := range user.Roles {
			rbacAuthz.AssignRole(user.ID, role)
		}
	}

	// Create HTTP handlers
	mux := http.NewServeMux()

	// Auth endpoints (no authentication required)
	mux.HandleFunc("/api/v1/auth/login", loginHandler(jwtAuth))

	// Protected endpoints with auth middleware
	authMiddleware := middleware.Auth(
		middleware.AuthWithAuthenticator(jwtAuth),
		middleware.AuthWithSkipPaths("/api/v1/auth/login", "/health", "/metrics"),
	)

	authzMiddleware := middleware.Authz(
		middleware.AuthzWithAuthorizer(rbacAuthz),
		middleware.AuthzWithSkipPaths("/api/v1/auth/login", "/api/v1/auth/refresh", "/api/v1/auth/logout", "/health", "/metrics"),
	)

	// Protected handlers
	mux.HandleFunc("/api/v1/auth/refresh", withMiddleware(refreshHandler(jwtAuth), authMiddleware))
	mux.HandleFunc("/api/v1/auth/logout", withMiddleware(logoutHandler(jwtAuth), authMiddleware))
	mux.HandleFunc("/api/v1/users", withMiddleware(usersHandler(), authMiddleware, authzMiddleware))
	mux.HandleFunc("/api/v1/users/", withMiddleware(userHandler(), authMiddleware, authzMiddleware))
	mux.HandleFunc("/api/v1/me", withMiddleware(meHandler(), authMiddleware))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Start server
	server := &http.Server{
		Addr:    ":8082",
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nShutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		tokenStore.Close()
	}()

	fmt.Println("[Server] Auth demo server started on :8082")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Printf("Server error: %v\n", err)
	}
}

func printBanner() {
	fmt.Println("===========================================")
	fmt.Println("  Sentinel-X Auth/Authz Demo")
	fmt.Println("===========================================")
	fmt.Println("Demo Users:")
	fmt.Println("  - admin/admin123   (role: admin - full access)")
	fmt.Println("  - editor/editor123 (role: editor - read/write)")
	fmt.Println("  - viewer/viewer123 (role: viewer - read only)")
	fmt.Println("-------------------------------------------")
	fmt.Println("Endpoints:")
	fmt.Println("  POST /api/v1/auth/login    - Login")
	fmt.Println("  POST /api/v1/auth/refresh  - Refresh token")
	fmt.Println("  POST /api/v1/auth/logout   - Logout")
	fmt.Println("  GET  /api/v1/me            - Current user info")
	fmt.Println("  GET  /api/v1/users         - List users (read)")
	fmt.Println("  POST /api/v1/users         - Create user (create)")
	fmt.Println("  GET  /api/v1/users/:id     - Get user (read)")
	fmt.Println("  PUT  /api/v1/users/:id     - Update user (update)")
	fmt.Println("  DELETE /api/v1/users/:id   - Delete user (delete)")
	fmt.Println("-------------------------------------------")
}

// withMiddleware applies middleware to a handler.
func withMiddleware(handler http.HandlerFunc, middlewares ...transport.MiddlewareFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create transport context wrapper
		ctx := &httpContext{
			req: r,
			w:   w,
		}

		// Build handler chain
		var h transport.HandlerFunc = func(c transport.Context) {
			handler(c.ResponseWriter(), c.HTTPRequest())
		}

		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}

		h(ctx)
	}
}

// loginHandler handles user login.
func loginHandler(jwtAuth auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonResponse(w, http.StatusMethodNotAllowed, response.ErrorWithCode(
				errors.ErrBadRequest.Code, "method not allowed"))
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonResponse(w, http.StatusBadRequest, response.Err(
				errors.ErrInvalidParam.WithMessage("invalid request body")))
			return
		}

		// Find user
		user, ok := users[req.Username]
		if !ok || user.Password != req.Password {
			jsonResponse(w, http.StatusUnauthorized, response.Err(
				errors.ErrInvalidCredentials))
			return
		}

		// Generate token with extra claims
		token, err := jwtAuth.Sign(r.Context(), user.ID,
			auth.WithExtra(map[string]interface{}{
				"username": user.Username,
				"roles":    user.Roles,
			}),
		)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, response.Err(
				errors.ErrInternal.WithCause(err)))
			return
		}

		jsonResponse(w, http.StatusOK, response.Success(map[string]interface{}{
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
}

// refreshHandler handles token refresh.
func refreshHandler(jwtAuth auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonResponse(w, http.StatusMethodNotAllowed, response.ErrorWithCode(
				errors.ErrBadRequest.Code, "method not allowed"))
			return
		}

		// Get current token from context
		tokenString := auth.TokenFromContext(r.Context())
		if tokenString == "" {
			jsonResponse(w, http.StatusUnauthorized, response.Err(
				errors.ErrUnauthorized.WithMessage("no token in context")))
			return
		}

		// Refresh token
		token, err := jwtAuth.Refresh(r.Context(), tokenString)
		if err != nil {
			errno := errors.FromError(err)
			jsonResponse(w, errno.HTTPStatus(), response.Err(errno))
			return
		}

		jsonResponse(w, http.StatusOK, response.Success(map[string]interface{}{
			"access_token": token.GetAccessToken(),
			"token_type":   token.GetTokenType(),
			"expires_in":   token.GetExpiresIn(),
			"expires_at":   token.GetExpiresAt(),
		}))
	}
}

// logoutHandler handles user logout (token revocation).
func logoutHandler(jwtAuth auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonResponse(w, http.StatusMethodNotAllowed, response.ErrorWithCode(
				errors.ErrBadRequest.Code, "method not allowed"))
			return
		}

		// Get current token from context
		tokenString := auth.TokenFromContext(r.Context())
		if tokenString == "" {
			jsonResponse(w, http.StatusUnauthorized, response.Err(
				errors.ErrUnauthorized.WithMessage("no token in context")))
			return
		}

		// Revoke token
		if err := jwtAuth.Revoke(r.Context(), tokenString); err != nil {
			errno := errors.FromError(err)
			jsonResponse(w, errno.HTTPStatus(), response.Err(errno))
			return
		}

		jsonResponse(w, http.StatusOK, response.Success(map[string]string{
			"message": "logged out successfully",
		}))
	}
}

// meHandler returns current user info.
func meHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jsonResponse(w, http.StatusMethodNotAllowed, response.ErrorWithCode(
				errors.ErrBadRequest.Code, "method not allowed"))
			return
		}

		claims := auth.ClaimsFromContext(r.Context())
		if claims == nil {
			jsonResponse(w, http.StatusUnauthorized, response.Err(
				errors.ErrUnauthorized.WithMessage("no claims in context")))
			return
		}

		jsonResponse(w, http.StatusOK, response.Success(map[string]interface{}{
			"id":       claims.Subject,
			"username": claims.GetExtraString("username"),
			"roles":    claims.Extra["roles"],
			"issued":   time.Unix(claims.IssuedAt, 0).Format(time.RFC3339),
			"expires":  time.Unix(claims.ExpiresAt, 0).Format(time.RFC3339),
		}))
	}
}

// usersHandler handles user list and creation.
func usersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// List users (requires read permission)
			userList := make([]map[string]interface{}, 0, len(users))
			for _, u := range users {
				userList = append(userList, map[string]interface{}{
					"id":       u.ID,
					"username": u.Username,
					"roles":    u.Roles,
				})
			}
			jsonResponse(w, http.StatusOK, response.Success(userList))

		case http.MethodPost:
			// Create user (requires create permission)
			jsonResponse(w, http.StatusOK, response.Success(map[string]string{
				"message": "user created (demo)",
			}))

		default:
			jsonResponse(w, http.StatusMethodNotAllowed, response.ErrorWithCode(
				errors.ErrBadRequest.Code, "method not allowed"))
		}
	}
}

// userHandler handles single user operations.
func userHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from path (simplified)
		userID := r.URL.Path[len("/api/v1/users/"):]
		if userID == "" {
			jsonResponse(w, http.StatusBadRequest, response.Err(
				errors.ErrInvalidParam.WithMessage("user id required")))
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Get user (requires read permission)
			for _, u := range users {
				if u.ID == userID {
					jsonResponse(w, http.StatusOK, response.Success(map[string]interface{}{
						"id":       u.ID,
						"username": u.Username,
						"roles":    u.Roles,
					}))
					return
				}
			}
			jsonResponse(w, http.StatusNotFound, response.Err(errors.ErrUserNotFound))

		case http.MethodPut:
			// Update user (requires update permission)
			jsonResponse(w, http.StatusOK, response.Success(map[string]string{
				"message": "user updated (demo)",
				"id":      userID,
			}))

		case http.MethodDelete:
			// Delete user (requires delete permission)
			jsonResponse(w, http.StatusOK, response.Success(map[string]string{
				"message": "user deleted (demo)",
				"id":      userID,
			}))

		default:
			jsonResponse(w, http.StatusMethodNotAllowed, response.ErrorWithCode(
				errors.ErrBadRequest.Code, "method not allowed"))
		}
	}
}

// jsonResponse writes JSON response.
func jsonResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// httpContext implements transport.Context for standard http.
type httpContext struct {
	req *http.Request
	w   http.ResponseWriter
}

func (c *httpContext) Request() context.Context       { return c.req.Context() }
func (c *httpContext) SetRequest(ctx context.Context) { c.req = c.req.WithContext(ctx) }
func (c *httpContext) HTTPRequest() *http.Request     { return c.req }
func (c *httpContext) ResponseWriter() http.ResponseWriter {
	return c.w
}
func (c *httpContext) Body() io.ReadCloser { return c.req.Body }
func (c *httpContext) Param(key string) string {
	return ""
}
func (c *httpContext) Query(key string) string {
	return c.req.URL.Query().Get(key)
}
func (c *httpContext) Header(key string) string {
	return c.req.Header.Get(key)
}
func (c *httpContext) SetHeader(key, value string) {
	c.w.Header().Set(key, value)
}
func (c *httpContext) Bind(v interface{}) error {
	return json.NewDecoder(c.req.Body).Decode(v)
}
func (c *httpContext) JSON(code int, v interface{}) {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(code)
	json.NewEncoder(c.w).Encode(v)
}
func (c *httpContext) String(code int, s string) {
	c.w.WriteHeader(code)
	c.w.Write([]byte(s))
}
func (c *httpContext) Error(code int, err error) {
	c.w.WriteHeader(code)
	c.w.Write([]byte(err.Error()))
}
func (c *httpContext) GetRawContext() interface{} {
	return c
}
