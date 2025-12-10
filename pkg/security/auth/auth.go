// Package auth provides authentication interfaces and implementations for Sentinel-X.
//
// This package follows the onex project authentication architecture pattern,
// providing a unified authentication interface that can be implemented by
// different authentication providers (JWT, OAuth2, API Key, etc.).
//
// The authentication flow:
//  1. Client provides credentials (username/password, API key, etc.)
//  2. Authenticator.Authenticate() validates credentials and returns a Token
//  3. Token is included in subsequent requests
//  4. Authenticator.Verify() validates the token and extracts claims
//  5. Claims are injected into request context for use by handlers
//
// Design Principles:
//   - Interface-driven: All authenticators implement the Authenticator interface
//   - Context-aware: Claims and user info are propagated via context
//   - Extensible: Easy to add new authentication providers
//   - Secure: No direct JWT parsing in business code
//
// Usage:
//
//	// Create JWT authenticator
//	auth, err := jwt.New(jwt.WithOptions(jwtOpts))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Generate token for user
//	token, err := auth.Sign(ctx, userID)
//	if err != nil {
//	    return err
//	}
//
//	// Verify token and get claims
//	claims, err := auth.Verify(ctx, tokenString)
//	if err != nil {
//	    return err
//	}
package auth

import (
	"context"
	"encoding/json"
	"time"
)

// Authenticator defines the authentication interface.
// All authentication providers must implement this interface.
type Authenticator interface {
	// Sign creates a new token for the given subject (usually user ID).
	// The returned Token contains the access token and metadata.
	Sign(ctx context.Context, subject string, opts ...SignOption) (Token, error)

	// Verify validates the token and returns the claims.
	// Returns an error if the token is invalid, expired, or revoked.
	Verify(ctx context.Context, tokenString string) (*Claims, error)

	// Refresh creates a new token using a valid existing token.
	// The original token's claims are preserved, but expiration is extended.
	// Returns an error if the token cannot be refreshed (e.g., max refresh exceeded).
	Refresh(ctx context.Context, tokenString string) (Token, error)

	// Revoke invalidates the given token.
	// After revocation, the token will fail verification.
	Revoke(ctx context.Context, tokenString string) error

	// Type returns the authenticator type (e.g., "jwt", "oauth2", "apikey").
	Type() string
}

// Token represents an authentication token with metadata.
type Token interface {
	// GetAccessToken returns the access token string.
	GetAccessToken() string

	// GetTokenType returns the token type (e.g., "Bearer").
	GetTokenType() string

	// GetExpiresAt returns the token expiration timestamp (Unix seconds).
	GetExpiresAt() int64

	// GetExpiresIn returns the duration until expiration in seconds.
	GetExpiresIn() int64

	// GetRefreshToken returns the refresh token (optional, may be empty).
	GetRefreshToken() string

	// JSON returns the token as JSON bytes.
	JSON() ([]byte, error)

	// Map returns the token as a map.
	Map() map[string]interface{}
}

// Claims represents the authentication claims extracted from a token.
type Claims struct {
	// Subject is the principal that is the subject of the token (user ID).
	Subject string `json:"sub"`

	// Issuer is the token issuer.
	Issuer string `json:"iss,omitempty"`

	// Audience is the intended audience.
	Audience []string `json:"aud,omitempty"`

	// ExpiresAt is the expiration time (Unix timestamp).
	ExpiresAt int64 `json:"exp,omitempty"`

	// IssuedAt is the time when the token was issued (Unix timestamp).
	IssuedAt int64 `json:"iat,omitempty"`

	// NotBefore is the time before which the token is not valid (Unix timestamp).
	NotBefore int64 `json:"nbf,omitempty"`

	// ID is the unique identifier for the token.
	ID string `json:"jti,omitempty"`

	// Extra contains additional custom claims.
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// Valid checks if the claims are valid (not expired, etc.).
func (c *Claims) Valid() bool {
	now := time.Now().Unix()
	if c.ExpiresAt > 0 && now > c.ExpiresAt {
		return false
	}
	if c.NotBefore > 0 && now < c.NotBefore {
		return false
	}
	return true
}

// IsExpired returns true if the token has expired.
func (c *Claims) IsExpired() bool {
	if c.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > c.ExpiresAt
}

// GetExtra returns a custom claim value.
func (c *Claims) GetExtra(key string) (interface{}, bool) {
	if c.Extra == nil {
		return nil, false
	}
	v, ok := c.Extra[key]
	return v, ok
}

// GetExtraString returns a custom claim as string.
func (c *Claims) GetExtraString(key string) string {
	if v, ok := c.GetExtra(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SignOption is a functional option for signing tokens.
type SignOption func(*SignOptions)

// SignOptions contains options for token signing.
type SignOptions struct {
	// ExpiresAt overrides the default expiration time.
	ExpiresAt *time.Time

	// Extra contains additional claims to include in the token.
	Extra map[string]interface{}

	// Audience overrides the default audience.
	Audience []string

	// TokenID sets a custom token ID.
	TokenID string
}

// WithExpiresAt sets custom expiration time.
func WithExpiresAt(t time.Time) SignOption {
	return func(o *SignOptions) {
		o.ExpiresAt = &t
	}
}

// WithExtra sets additional claims.
func WithExtra(extra map[string]interface{}) SignOption {
	return func(o *SignOptions) {
		o.Extra = extra
	}
}

// WithAudience sets the token audience.
func WithAudience(aud ...string) SignOption {
	return func(o *SignOptions) {
		o.Audience = aud
	}
}

// WithTokenID sets a custom token ID.
func WithTokenID(id string) SignOption {
	return func(o *SignOptions) {
		o.TokenID = id
	}
}

// BaseToken is a basic Token implementation.
type BaseToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// GetAccessToken returns the access token.
func (t *BaseToken) GetAccessToken() string {
	return t.AccessToken
}

// GetTokenType returns the token type.
func (t *BaseToken) GetTokenType() string {
	return t.TokenType
}

// GetExpiresAt returns the expiration timestamp.
func (t *BaseToken) GetExpiresAt() int64 {
	return t.ExpiresAt
}

// GetExpiresIn returns the duration until expiration.
func (t *BaseToken) GetExpiresIn() int64 {
	return t.ExpiresIn
}

// GetRefreshToken returns the refresh token.
func (t *BaseToken) GetRefreshToken() string {
	return t.RefreshToken
}

// JSON returns the token as JSON.
func (t *BaseToken) JSON() ([]byte, error) {
	return json.Marshal(t)
}

// Map returns the token as a map.
func (t *BaseToken) Map() map[string]interface{} {
	m := map[string]interface{}{
		"access_token": t.AccessToken,
		"token_type":   t.TokenType,
		"expires_at":   t.ExpiresAt,
		"expires_in":   t.ExpiresIn,
	}
	if t.RefreshToken != "" {
		m["refresh_token"] = t.RefreshToken
	}
	return m
}
