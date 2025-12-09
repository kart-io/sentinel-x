// Package jwt provides JWT-based authentication for Sentinel-X.
//
// This package implements the auth.Authenticator interface using JSON Web Tokens.
// It supports multiple signing algorithms (HMAC, RSA, ECDSA) and provides
// token generation, verification, refresh, and revocation capabilities.
//
// Features:
//   - Multiple signing algorithms: HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512
//   - Token revocation via pluggable store interface
//   - Custom claims support
//   - Configurable expiration and refresh policies
//
// Usage:
//
//	// Create JWT authenticator with options
//	jwtAuth, err := jwt.New(
//	    jwt.WithKey("your-secret-key-min-32-chars-long"),
//	    jwt.WithExpired(2 * time.Hour),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Sign a token
//	token, err := jwtAuth.Sign(ctx, "user-123")
//
//	// Verify a token
//	claims, err := jwtAuth.Verify(ctx, tokenString)
package jwt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/kart-io/sentinel-x/pkg/auth"
	"github.com/kart-io/sentinel-x/pkg/errors"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/jwt"
)

// JWT implements auth.Authenticator using JSON Web Tokens.
type JWT struct {
	opts   *jwtopts.Options
	store  Store
	method jwt.SigningMethod
}

// Option is a functional option for JWT authenticator.
type Option func(*JWT)

// New creates a new JWT authenticator.
func New(opts ...Option) (*JWT, error) {
	j := &JWT{
		opts: jwtopts.NewOptions(),
	}

	for _, opt := range opts {
		opt(j)
	}

	// Complete and validate options
	if err := j.opts.Complete(); err != nil {
		return nil, fmt.Errorf("complete options: %w", err)
	}

	if err := j.opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %w", err)
	}

	// Get signing method
	j.method = jwt.GetSigningMethod(j.opts.SigningMethod)
	if j.method == nil {
		return nil, fmt.Errorf("unsupported signing method: %s", j.opts.SigningMethod)
	}

	return j, nil
}

// WithOptions sets the JWT options.
func WithOptions(opts *jwtopts.Options) Option {
	return func(j *JWT) {
		if opts != nil {
			j.opts = opts
		}
	}
}

// WithKey sets the signing key.
func WithKey(key string) Option {
	return func(j *JWT) {
		j.opts.Key = key
	}
}

// WithSigningMethod sets the signing algorithm.
func WithSigningMethod(method string) Option {
	return func(j *JWT) {
		j.opts.SigningMethod = method
	}
}

// WithExpired sets the token expiration duration.
func WithExpired(d time.Duration) Option {
	return func(j *JWT) {
		j.opts.Expired = d
	}
}

// WithIssuer sets the token issuer.
func WithIssuer(issuer string) Option {
	return func(j *JWT) {
		j.opts.Issuer = issuer
	}
}

// WithStore sets the token store for revocation support.
func WithStore(store Store) Option {
	return func(j *JWT) {
		j.store = store
	}
}

// Type returns the authenticator type.
func (j *JWT) Type() string {
	return "jwt"
}

// Sign creates a new token for the given subject.
func (j *JWT) Sign(ctx context.Context, subject string, opts ...auth.SignOption) (auth.Token, error) {
	signOpts := &auth.SignOptions{}
	for _, opt := range opts {
		opt(signOpts)
	}

	now := time.Now()
	expiresAt := now.Add(j.opts.Expired)
	if signOpts.ExpiresAt != nil {
		expiresAt = *signOpts.ExpiresAt
	}

	// Generate token ID
	tokenID := signOpts.TokenID
	if tokenID == "" {
		tokenID = generateTokenID()
	}

	// Build claims
	audience := j.opts.Audience
	if len(signOpts.Audience) > 0 {
		audience = signOpts.Audience
	}

	claims := &customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    j.opts.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
		Extra: signOpts.Extra,
	}

	if len(audience) > 0 {
		claims.Audience = audience
	}

	// Create token
	token := jwt.NewWithClaims(j.method, claims)

	// Set key ID if configured
	if j.opts.KeyID != "" {
		token.Header["kid"] = j.opts.KeyID
	}

	// Sign token
	tokenString, err := token.SignedString(j.getSigningKey())
	if err != nil {
		return nil, errors.ErrInternal.WithCause(err).WithMessage("failed to sign token")
	}

	return &auth.BaseToken{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt.Unix(),
		ExpiresIn:   int64(expiresAt.Sub(now).Seconds()),
	}, nil
}

// Verify validates the token and returns the claims.
func (j *JWT) Verify(ctx context.Context, tokenString string) (*auth.Claims, error) {
	if tokenString == "" {
		return nil, errors.ErrInvalidToken.WithMessage("token is empty")
	}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if token.Method.Alg() != j.method.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.getVerifyingKey(), nil
	})

	if err != nil {
		return nil, j.mapParseError(err)
	}

	if !token.Valid {
		return nil, errors.ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(*customClaims)
	if !ok {
		return nil, errors.ErrInvalidToken.WithMessage("invalid claims type")
	}

	// Check if token is revoked
	if j.store != nil {
		revoked, err := j.store.IsRevoked(ctx, tokenString)
		if err != nil {
			return nil, errors.ErrInternal.WithCause(err).WithMessage("failed to check token revocation")
		}
		if revoked {
			return nil, errors.ErrTokenRevoked
		}
	}

	return &auth.Claims{
		Subject:   claims.Subject,
		Issuer:    claims.Issuer,
		Audience:  claims.Audience,
		ExpiresAt: claims.ExpiresAt.Unix(),
		IssuedAt:  claims.IssuedAt.Unix(),
		NotBefore: claims.NotBefore.Unix(),
		ID:        claims.ID,
		Extra:     claims.Extra,
	}, nil
}

// Refresh creates a new token using a valid existing token.
func (j *JWT) Refresh(ctx context.Context, tokenString string) (auth.Token, error) {
	// Verify the existing token (allow expired for refresh)
	claims, err := j.verifyForRefresh(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Check if refresh is allowed (within max refresh window)
	issuedAt := time.Unix(claims.IssuedAt, 0)
	maxRefreshTime := issuedAt.Add(j.opts.MaxRefresh)
	if time.Now().After(maxRefreshTime) {
		return nil, errors.ErrSessionExpired.WithMessage("token refresh window expired")
	}

	// Create new token with same subject and extra claims
	signOpts := []auth.SignOption{}
	if len(claims.Extra) > 0 {
		signOpts = append(signOpts, auth.WithExtra(claims.Extra))
	}
	if len(claims.Audience) > 0 {
		signOpts = append(signOpts, auth.WithAudience(claims.Audience...))
	}

	return j.Sign(ctx, claims.Subject, signOpts...)
}

// verifyForRefresh verifies a token for refresh, allowing expired tokens.
func (j *JWT) verifyForRefresh(ctx context.Context, tokenString string) (*auth.Claims, error) {
	if tokenString == "" {
		return nil, errors.ErrInvalidToken.WithMessage("token is empty")
	}

	// Parse token without expiration validation
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != j.method.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.getVerifyingKey(), nil
	})

	if err != nil {
		return nil, j.mapParseError(err)
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok {
		return nil, errors.ErrInvalidToken.WithMessage("invalid claims type")
	}

	// Check if token is revoked
	if j.store != nil {
		revoked, err := j.store.IsRevoked(ctx, tokenString)
		if err != nil {
			return nil, errors.ErrInternal.WithCause(err).WithMessage("failed to check token revocation")
		}
		if revoked {
			return nil, errors.ErrTokenRevoked
		}
	}

	return &auth.Claims{
		Subject:   claims.Subject,
		Issuer:    claims.Issuer,
		Audience:  claims.Audience,
		ExpiresAt: claims.ExpiresAt.Unix(),
		IssuedAt:  claims.IssuedAt.Unix(),
		NotBefore: claims.NotBefore.Unix(),
		ID:        claims.ID,
		Extra:     claims.Extra,
	}, nil
}

// Revoke invalidates the given token.
func (j *JWT) Revoke(ctx context.Context, tokenString string) error {
	if j.store == nil {
		return errors.ErrNotImplemented.WithMessage("token revocation requires a store")
	}

	// Parse token to get expiration
	claims, err := j.verifyForRefresh(ctx, tokenString)
	if err != nil {
		// Allow revoking invalid tokens (they're already unusable)
		if errors.IsCode(err, errors.ErrTokenExpired.Code) {
			return nil
		}
		return err
	}

	// Calculate TTL for store (time until token expires)
	expiration := time.Until(time.Unix(claims.ExpiresAt, 0))
	if expiration < 0 {
		// Token already expired, no need to store
		return nil
	}

	return j.store.Revoke(ctx, tokenString, expiration)
}

// getSigningKey returns the key used for signing.
func (j *JWT) getSigningKey() interface{} {
	// For HMAC algorithms, return the key as bytes
	if strings.HasPrefix(j.opts.SigningMethod, "HS") {
		return []byte(j.opts.Key)
	}
	// For RSA/ECDSA, the key should be parsed (simplified for now)
	return []byte(j.opts.Key)
}

// getVerifyingKey returns the key used for verification.
func (j *JWT) getVerifyingKey() interface{} {
	// For HMAC, use the same key for verification
	if strings.HasPrefix(j.opts.SigningMethod, "HS") {
		return []byte(j.opts.Key)
	}
	// For RSA/ECDSA, use public key if available
	if j.opts.PublicKey != "" {
		return []byte(j.opts.PublicKey)
	}
	return []byte(j.opts.Key)
}

// mapParseError maps jwt parse errors to sentinel-x errors.
func (j *JWT) mapParseError(err error) *errors.Errno {
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(err.Error(), "token is expired"):
		return errors.ErrTokenExpired
	case strings.Contains(err.Error(), "signature is invalid"):
		return errors.ErrInvalidToken.WithMessage("invalid signature")
	case strings.Contains(err.Error(), "token is malformed"):
		return errors.ErrInvalidToken.WithMessage("malformed token")
	case strings.Contains(err.Error(), "token is not valid yet"):
		return errors.ErrInvalidToken.WithMessage("token not valid yet")
	default:
		return errors.ErrInvalidToken.WithCause(err)
	}
}

// customClaims extends jwt.RegisteredClaims with extra fields.
type customClaims struct {
	jwt.RegisteredClaims
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// generateTokenID generates a random token ID.
func generateTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
