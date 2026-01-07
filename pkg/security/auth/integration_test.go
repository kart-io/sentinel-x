//go:build integration

package auth_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	authmw "github.com/kart-io/sentinel-x/pkg/infra/middleware/auth"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/security/authz/rbac"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

const (
	testKey     = "test-secret-key-at-least-64-chars-long-for-hmac-algorithms!!!!!!"
	testSubject = "user-123"
	testIssuer  = "sentinel-x-test"
)

// TestAuthFlow_CompleteTokenLifecycle tests the complete token lifecycle:
// Sign -> Verify -> Refresh -> Revoke
func TestAuthFlow_CompleteTokenLifecycle(t *testing.T) {
	ctx := context.Background()

	// Create JWT authenticator with token store
	store := jwt.NewMemoryStore()
	defer store.Close()

	jwtAuth, err := jwt.New(
		jwt.WithKey(testKey),
		jwt.WithSigningMethod("HS256"),
		jwt.WithExpired(time.Hour),
		jwt.WithIssuer(testIssuer),
		jwt.WithStore(store),
	)
	if err != nil {
		t.Fatalf("Failed to create JWT authenticator: %v", err)
	}

	// Step 1: Sign a new token
	t.Run("Sign", func(t *testing.T) {
		token, err := jwtAuth.Sign(ctx, testSubject, auth.WithExtra(map[string]interface{}{
			"username": "testuser",
			"role":     "admin",
		}))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		if token.GetAccessToken() == "" {
			t.Error("Expected access token to be non-empty")
		}
		if token.GetTokenType() != "Bearer" {
			t.Errorf("Expected token type Bearer, got %s", token.GetTokenType())
		}
		if token.GetExpiresAt() == 0 {
			t.Error("Expected expiration to be set")
		}
		if token.GetExpiresIn() <= 0 {
			t.Error("Expected expires_in to be positive")
		}
	})

	// Step 2: Verify the token
	var tokenString string
	var claims *auth.Claims
	t.Run("Verify", func(t *testing.T) {
		token, err := jwtAuth.Sign(ctx, testSubject, auth.WithExtra(map[string]interface{}{
			"username": "testuser",
			"role":     "admin",
		}))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}
		tokenString = token.GetAccessToken()

		claims, err = jwtAuth.Verify(ctx, tokenString)
		if err != nil {
			t.Fatalf("Failed to verify token: %v", err)
		}

		if claims.Subject != testSubject {
			t.Errorf("Expected subject %s, got %s", testSubject, claims.Subject)
		}
		if claims.Issuer != testIssuer {
			t.Errorf("Expected issuer %s, got %s", testIssuer, claims.Issuer)
		}
		if username := claims.GetExtraString("username"); username != "testuser" {
			t.Errorf("Expected username testuser, got %s", username)
		}
		if role := claims.GetExtraString("role"); role != "admin" {
			t.Errorf("Expected role admin, got %s", role)
		}
		if !claims.Valid() {
			t.Error("Expected claims to be valid")
		}
	})

	// Step 3: Refresh the token
	t.Run("Refresh", func(t *testing.T) {
		newToken, err := jwtAuth.Refresh(ctx, tokenString)
		if err != nil {
			t.Fatalf("Failed to refresh token: %v", err)
		}

		if newToken.GetAccessToken() == "" {
			t.Error("Expected refreshed token to be non-empty")
		}
		if newToken.GetAccessToken() == tokenString {
			t.Error("Expected refreshed token to be different from original")
		}

		// Verify the refreshed token has the same claims
		newClaims, err := jwtAuth.Verify(ctx, newToken.GetAccessToken())
		if err != nil {
			t.Fatalf("Failed to verify refreshed token: %v", err)
		}
		if newClaims.Subject != claims.Subject {
			t.Error("Expected same subject after refresh")
		}
		if newClaims.GetExtraString("username") != claims.GetExtraString("username") {
			t.Error("Expected same username after refresh")
		}
	})

	// Step 4: Revoke the token
	t.Run("Revoke", func(t *testing.T) {
		// Create a new token for revocation test
		token, err := jwtAuth.Sign(ctx, testSubject)
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}
		revokeToken := token.GetAccessToken()

		// Verify token works before revocation
		_, err = jwtAuth.Verify(ctx, revokeToken)
		if err != nil {
			t.Fatalf("Token should be valid before revocation: %v", err)
		}

		// Revoke the token
		err = jwtAuth.Revoke(ctx, revokeToken)
		if err != nil {
			t.Fatalf("Failed to revoke token: %v", err)
		}

		// Verify token fails after revocation
		_, err = jwtAuth.Verify(ctx, revokeToken)
		if err == nil {
			t.Error("Expected error when verifying revoked token")
		}
		if !errors.IsCode(err, errors.ErrTokenRevoked.Code) {
			t.Errorf("Expected ErrTokenRevoked, got %v", err)
		}
	})
}

// TestAuthFlow_ErrorScenarios tests various error scenarios
func TestAuthFlow_ErrorScenarios(t *testing.T) {
	ctx := context.Background()

	store := jwt.NewMemoryStore()
	defer store.Close()

	jwtAuth, err := jwt.New(
		jwt.WithKey(testKey),
		jwt.WithSigningMethod("HS256"),
		jwt.WithExpired(100*time.Millisecond), // Short expiration for testing
		jwt.WithIssuer(testIssuer),
		jwt.WithStore(store),
	)
	if err != nil {
		t.Fatalf("Failed to create JWT authenticator: %v", err)
	}

	t.Run("ExpiredToken", func(t *testing.T) {
		// Create token with short expiration
		token, err := jwtAuth.Sign(ctx, testSubject)
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		// Wait for token to expire
		time.Sleep(150 * time.Millisecond)

		// Verify should fail
		_, err = jwtAuth.Verify(ctx, token.GetAccessToken())
		if err == nil {
			t.Error("Expected error when verifying expired token")
		}
		if !errors.IsCode(err, errors.ErrTokenExpired.Code) {
			t.Errorf("Expected ErrTokenExpired, got %v", err)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		tests := []struct {
			name  string
			token string
		}{
			{"Empty", ""},
			{"Malformed", "not.a.valid.jwt"},
			{"Invalid Signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.invalid"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := jwtAuth.Verify(ctx, tt.token)
				if err == nil {
					t.Error("Expected error for invalid token")
				}
				if !errors.IsCode(err, errors.ErrInvalidToken.Code) {
					t.Errorf("Expected ErrInvalidToken, got %v", err)
				}
			})
		}
	})

	t.Run("RevokedToken", func(t *testing.T) {
		// Create a JWT with longer expiration for this test
		longAuth, err := jwt.New(
			jwt.WithKey(testKey),
			jwt.WithSigningMethod("HS256"),
			jwt.WithExpired(time.Hour), // Long expiration
			jwt.WithIssuer(testIssuer),
			jwt.WithStore(store),
		)
		if err != nil {
			t.Fatalf("Failed to create JWT authenticator: %v", err)
		}

		// Create a token for revocation test
		token, err := longAuth.Sign(ctx, testSubject)
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}
		revokeToken := token.GetAccessToken()

		// Verify token works before revocation
		_, err = longAuth.Verify(ctx, revokeToken)
		if err != nil {
			t.Fatalf("Token should be valid before revocation: %v", err)
		}

		// Revoke the token
		err = longAuth.Revoke(ctx, revokeToken)
		if err != nil {
			t.Fatalf("Failed to revoke token: %v", err)
		}

		// Verify token fails after revocation
		_, err = longAuth.Verify(ctx, revokeToken)
		if err == nil {
			t.Error("Expected error when verifying revoked token")
		}
		if !errors.IsCode(err, errors.ErrTokenRevoked.Code) {
			t.Errorf("Expected ErrTokenRevoked, got %v", err)
		}

		// Refresh should also fail
		_, err = longAuth.Refresh(ctx, revokeToken)
		if err == nil {
			t.Error("Expected error when refreshing revoked token")
		}
	})

	t.Run("RefreshExpiredBeyondMaxRefresh", func(t *testing.T) {
		// Create authenticator with very short max refresh window
		shortAuth, err := jwt.New(
			jwt.WithKey(testKey),
			jwt.WithSigningMethod("HS256"),
			jwt.WithExpired(100*time.Millisecond),
			jwt.WithOptions(&jwtopts.Options{
				Key:           testKey,
				SigningMethod: "HS256",
				Expired:       100 * time.Millisecond,
				MaxRefresh:    200 * time.Millisecond,
				Issuer:        testIssuer,
			}),
		)
		if err != nil {
			t.Fatalf("Failed to create JWT authenticator: %v", err)
		}

		token, err := shortAuth.Sign(ctx, testSubject)
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		// Wait beyond max refresh window
		time.Sleep(250 * time.Millisecond)

		// Refresh should fail
		_, err = shortAuth.Refresh(ctx, token.GetAccessToken())
		if err == nil {
			t.Error("Expected error when refreshing token beyond max refresh window")
		}
		if !errors.IsCode(err, errors.ErrSessionExpired.Code) {
			t.Errorf("Expected ErrSessionExpired, got %v", err)
		}
	})
}

// TestAuthFlow_WithRBAC tests the integration of authentication and authorization
func TestAuthFlow_WithRBAC(t *testing.T) {
	ctx := context.Background()

	// Create RBAC authorizer
	rbacAuthz := rbac.New()

	// Define roles and permissions
	rbacAuthz.AddRole("admin",
		authz.NewPermission("*", "*"),
	)
	rbacAuthz.AddRole("editor",
		authz.NewPermission("posts", "read"),
		authz.NewPermission("posts", "create"),
		authz.NewPermission("posts", "update"),
	)
	rbacAuthz.AddRole("viewer",
		authz.NewPermission("posts", "read"),
	)

	// Assign roles to users
	rbacAuthz.AssignRole("user-admin", "admin")
	rbacAuthz.AssignRole("user-editor", "editor")
	rbacAuthz.AssignRole("user-viewer", "viewer")

	tests := []struct {
		name      string
		subject   string
		resource  string
		action    string
		allowed   bool
		wantError bool
	}{
		{
			name:      "Admin can do everything",
			subject:   "user-admin",
			resource:  "posts",
			action:    "delete",
			allowed:   true,
			wantError: false,
		},
		{
			name:      "Editor can update posts",
			subject:   "user-editor",
			resource:  "posts",
			action:    "update",
			allowed:   true,
			wantError: false,
		},
		{
			name:      "Editor cannot delete posts",
			subject:   "user-editor",
			resource:  "posts",
			action:    "delete",
			allowed:   false,
			wantError: false,
		},
		{
			name:      "Viewer can read posts",
			subject:   "user-viewer",
			resource:  "posts",
			action:    "read",
			allowed:   true,
			wantError: false,
		},
		{
			name:      "Viewer cannot create posts",
			subject:   "user-viewer",
			resource:  "posts",
			action:    "create",
			allowed:   false,
			wantError: false,
		},
		{
			name:      "Unknown user has no permissions",
			subject:   "user-unknown",
			resource:  "posts",
			action:    "read",
			allowed:   false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := rbacAuthz.Authorize(ctx, tt.subject, tt.resource, tt.action)
			if (err != nil) != tt.wantError {
				t.Errorf("Authorize() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if allowed != tt.allowed {
				t.Errorf("Authorize() = %v, want %v", allowed, tt.allowed)
			}
		})
	}
}

// TestMiddleware_AuthIntegration tests the auth middleware with JWT
func TestMiddleware_AuthIntegration(t *testing.T) {
	// Create JWT authenticator
	jwtAuth, err := jwt.New(
		jwt.WithKey(testKey),
		jwt.WithSigningMethod("HS256"),
		jwt.WithExpired(time.Hour),
		jwt.WithIssuer(testIssuer),
	)
	if err != nil {
		t.Fatalf("Failed to create JWT authenticator: %v", err)
	}

	// Create auth middleware
	authOpts := mwopts.NewAuthOptions()
	authOpts.SkipPaths = []string{"/public", "/health"}
	authMiddleware := authmw.AuthWithOptions(*authOpts, jwtAuth, nil, nil)

	handlerCalled := false
	// Create a test handler that checks for authentication
	testHandler := func(ctx transport.Context) {
		handlerCalled = true

		// For non-skip paths, verify claims are in context
		path := ctx.HTTPRequest().URL.Path
		if path != "/public" && path != "/health" {
			// Verify claims are in context
			subject := auth.SubjectFromContext(ctx.Request())
			if subject == "" {
				t.Error("Expected subject to be in context")
			}
			if subject != testSubject {
				t.Errorf("Expected subject %s, got %s", testSubject, subject)
			}

			claims := auth.ClaimsFromContext(ctx.Request())
			if claims == nil {
				t.Error("Expected claims to be in context")
			}
		}

		ctx.JSON(200, map[string]string{"status": "ok"})
	}

	// Create valid token
	ctx := context.Background()
	token, err := jwtAuth.Sign(ctx, testSubject)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
		shouldCallNext bool
	}{
		{
			name:           "Valid token",
			path:           "/api/users",
			authHeader:     "Bearer " + token.GetAccessToken(),
			expectedStatus: 200,
			shouldCallNext: true,
		},
		{
			name:           "Missing token",
			path:           "/api/users",
			authHeader:     "",
			expectedStatus: 401,
			shouldCallNext: false,
		},
		{
			name:           "Invalid token",
			path:           "/api/users",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: 401,
			shouldCallNext: false,
		},
		{
			name:           "Skip path - public",
			path:           "/public",
			authHeader:     "",
			expectedStatus: 200,
			shouldCallNext: true,
		},
		{
			name:           "Skip path - health",
			path:           "/health",
			authHeader:     "",
			expectedStatus: 200,
			shouldCallNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			mockCtx := newMockContext(req, w)
			handler := authMiddleware(testHandler)
			handler(mockCtx)

			if tt.shouldCallNext != handlerCalled {
				t.Errorf("Expected handler called = %v, got %v", tt.shouldCallNext, handlerCalled)
			}

			mockCtx.mu.RLock()
			if mockCtx.jsonCalled && mockCtx.jsonCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, mockCtx.jsonCode)
			}
			mockCtx.mu.RUnlock()
		})
	}
}

// TestMiddleware_AuthzIntegration tests the authz middleware with RBAC
func TestMiddleware_AuthzIntegration(t *testing.T) {
	// Create RBAC authorizer
	rbacAuthz := rbac.New()
	rbacAuthz.AddRole("admin", authz.NewPermission("*", "*"))
	rbacAuthz.AddRole("editor", authz.NewPermission("posts", "read"), authz.NewPermission("posts", "create"))
	rbacAuthz.AssignRole("user-admin", "admin")
	rbacAuthz.AssignRole("user-editor", "editor")

	// Create authz middleware
	authzOpts := mwopts.NewAuthzOptions()
	authzOpts.SkipPaths = []string{"/public"}
	authzMiddleware := authmw.AuthzWithOptions(*authzOpts, rbacAuthz, nil, nil, nil, nil)

	handlerCalled := false
	testHandler := func(ctx transport.Context) {
		handlerCalled = true
		ctx.JSON(200, map[string]string{"status": "ok"})
	}

	tests := []struct {
		name           string
		path           string
		method         string
		subject        string
		expectedStatus int
		shouldCallNext bool
	}{
		{
			name:           "Admin can delete posts",
			path:           "/api/v1/posts",
			method:         http.MethodDelete,
			subject:        "user-admin",
			expectedStatus: 200,
			shouldCallNext: true,
		},
		{
			name:           "Editor can read posts",
			path:           "/api/v1/posts",
			method:         http.MethodGet,
			subject:        "user-editor",
			expectedStatus: 200,
			shouldCallNext: true,
		},
		{
			name:           "Editor cannot delete posts",
			path:           "/api/v1/posts",
			method:         http.MethodDelete,
			subject:        "user-editor",
			expectedStatus: 403,
			shouldCallNext: false,
		},
		{
			name:           "Unknown user denied",
			path:           "/api/v1/posts",
			method:         http.MethodGet,
			subject:        "user-unknown",
			expectedStatus: 403,
			shouldCallNext: false,
		},
		{
			name:           "Skip path allowed without permission",
			path:           "/public",
			method:         http.MethodGet,
			subject:        "user-unknown",
			expectedStatus: 200,
			shouldCallNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Inject subject into context
			ctx := auth.ContextWithSubject(req.Context(), tt.subject)
			req = req.WithContext(ctx)

			mockCtx := newMockContext(req, w)
			handler := authzMiddleware(testHandler)
			handler(mockCtx)

			if tt.shouldCallNext != handlerCalled {
				t.Errorf("Expected handler called = %v, got %v", tt.shouldCallNext, handlerCalled)
			}

			mockCtx.mu.RLock()
			if mockCtx.jsonCalled && mockCtx.jsonCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, mockCtx.jsonCode)
			}
			mockCtx.mu.RUnlock()
		})
	}
}

// TestMiddleware_FullAuthFlow tests the complete middleware chain: Auth -> Authz -> Handler
func TestMiddleware_FullAuthFlow(t *testing.T) {
	ctx := context.Background()

	// Create JWT authenticator
	jwtAuth, err := jwt.New(
		jwt.WithKey(testKey),
		jwt.WithSigningMethod("HS256"),
		jwt.WithExpired(time.Hour),
		jwt.WithIssuer(testIssuer),
	)
	if err != nil {
		t.Fatalf("Failed to create JWT authenticator: %v", err)
	}

	// Create RBAC authorizer
	rbacAuthz := rbac.New()
	rbacAuthz.AddRole("admin", authz.NewPermission("*", "*"))
	rbacAuthz.AddRole("viewer", authz.NewPermission("posts", "read"))
	rbacAuthz.AssignRole("user-admin", "admin")
	rbacAuthz.AssignRole("user-viewer", "viewer")

	// Create middlewares
	authOpts := mwopts.NewAuthOptions()
	authMiddleware := authmw.AuthWithOptions(*authOpts, jwtAuth, nil, nil)

	authzOpts := mwopts.NewAuthzOptions()
	authzMiddleware := authmw.AuthzWithOptions(*authzOpts, rbacAuthz, nil, nil, nil, nil)

	handlerCalled := false
	testHandler := func(ctx transport.Context) {
		handlerCalled = true
		ctx.JSON(200, map[string]string{"status": "ok"})
	}

	// Chain middlewares: Auth -> Authz -> Handler
	handler := authMiddleware(authzMiddleware(testHandler))

	tests := []struct {
		name           string
		subject        string
		method         string
		path           string
		shouldCallNext bool
		description    string
	}{
		{
			name:           "Admin can delete posts",
			subject:        "user-admin",
			method:         http.MethodDelete,
			path:           "/api/v1/posts",
			shouldCallNext: true,
			description:    "Admin should pass both auth and authz",
		},
		{
			name:           "Viewer can read posts",
			subject:        "user-viewer",
			method:         http.MethodGet,
			path:           "/api/v1/posts",
			shouldCallNext: true,
			description:    "Viewer should pass both auth and authz for read",
		},
		{
			name:           "Viewer cannot delete posts",
			subject:        "user-viewer",
			method:         http.MethodDelete,
			path:           "/api/v1/posts",
			shouldCallNext: false,
			description:    "Viewer should pass auth but fail authz for delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false

			// Create token for user
			token, err := jwtAuth.Sign(ctx, tt.subject)
			if err != nil {
				t.Fatalf("Failed to sign token: %v", err)
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+token.GetAccessToken())
			w := httptest.NewRecorder()

			mockCtx := newMockContext(req, w)
			handler(mockCtx)

			if tt.shouldCallNext != handlerCalled {
				t.Errorf("%s: Expected handler called = %v, got %v",
					tt.description, tt.shouldCallNext, handlerCalled)
			}
		})
	}
}

// TestConcurrentAccess tests concurrent token operations
func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	store := jwt.NewMemoryStore()
	defer store.Close()

	jwtAuth, err := jwt.New(
		jwt.WithKey(testKey),
		jwt.WithSigningMethod("HS256"),
		jwt.WithExpired(time.Hour),
		jwt.WithIssuer(testIssuer),
		jwt.WithStore(store),
	)
	if err != nil {
		t.Fatalf("Failed to create JWT authenticator: %v", err)
	}

	rbacAuthz := rbac.New()
	rbacAuthz.AddRole("user", authz.NewPermission("posts", "read"))

	const numGoroutines = 50
	const numOperations = 10

	t.Run("ConcurrentTokenGeneration", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					subject := "user-" + string(rune(id))
					_, err := jwtAuth.Sign(ctx, subject)
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent sign error: %v", err)
		}
	})

	t.Run("ConcurrentVerification", func(t *testing.T) {
		// Create a token to verify
		token, err := jwtAuth.Sign(ctx, testSubject)
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					_, err := jwtAuth.Verify(ctx, token.GetAccessToken())
					if err != nil {
						errors <- err
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent verify error: %v", err)
		}
	})

	t.Run("ConcurrentRBACOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				subject := "user-" + string(rune(id))

				// Assign role
				err := rbacAuthz.AssignRole(subject, "user")
				if err != nil {
					errors <- err
					return
				}

				// Check authorization
				for j := 0; j < numOperations; j++ {
					_, err := rbacAuthz.Authorize(ctx, subject, "posts", "read")
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent RBAC error: %v", err)
		}
	})

	t.Run("ConcurrentRevocation", func(t *testing.T) {
		// Create multiple tokens
		tokens := make([]string, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			token, err := jwtAuth.Sign(ctx, "user-"+string(rune(i)))
			if err != nil {
				t.Fatalf("Failed to sign token: %v", err)
			}
			tokens[i] = token.GetAccessToken()
		}

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		// Concurrently revoke all tokens
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(tokenStr string) {
				defer wg.Done()
				err := jwtAuth.Revoke(ctx, tokenStr)
				if err != nil {
					errors <- err
				}
			}(tokens[i])
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent revoke error: %v", err)
		}

		// Verify all tokens are revoked
		for i, tokenStr := range tokens {
			_, err := jwtAuth.Verify(ctx, tokenStr)
			if err == nil {
				t.Errorf("Token %d should be revoked", i)
			}
		}
	})
}

// mockContext is a minimal implementation of transport.Context for testing
type mockContext struct {
	req        *http.Request
	writer     http.ResponseWriter
	headers    map[string]string
	jsonCalled bool
	jsonCode   int
	jsonData   interface{}
	mu         sync.RWMutex
}

func newMockContext(req *http.Request, w http.ResponseWriter) *mockContext {
	return &mockContext{
		req:     req,
		writer:  w,
		headers: make(map[string]string),
	}
}

func (m *mockContext) Request() context.Context {
	return m.req.Context()
}

func (m *mockContext) SetRequest(ctx context.Context) {
	m.req = m.req.WithContext(ctx)
}

func (m *mockContext) HTTPRequest() *http.Request {
	return m.req
}

func (m *mockContext) ResponseWriter() http.ResponseWriter {
	return m.writer
}

func (m *mockContext) Body() io.ReadCloser {
	return m.req.Body
}

func (m *mockContext) Param(key string) string {
	return ""
}

func (m *mockContext) Query(key string) string {
	return m.req.URL.Query().Get(key)
}

func (m *mockContext) Header(key string) string {
	return m.req.Header.Get(key)
}

func (m *mockContext) SetHeader(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headers[key] = value
	if m.writer != nil {
		m.writer.Header().Set(key, value)
	}
}

func (m *mockContext) Bind(v interface{}) error {
	return nil
}

func (m *mockContext) Validate(v interface{}) error {
	return nil
}

func (m *mockContext) ShouldBindAndValidate(v interface{}) error {
	return nil
}

func (m *mockContext) MustBindAndValidate(v interface{}) (string, bool) {
	return "", true
}

func (m *mockContext) JSON(code int, v interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jsonCalled = true
	m.jsonCode = code
	m.jsonData = v
}

func (m *mockContext) String(code int, s string) {}

func (m *mockContext) Error(code int, err error) {
	m.JSON(code, map[string]string{"error": err.Error()})
}

func (m *mockContext) GetRawContext() interface{} {
	return nil
}

func (m *mockContext) Lang() string {
	return "en"
}

func (m *mockContext) SetLang(lang string) {}

var _ transport.Context = (*mockContext)(nil)
