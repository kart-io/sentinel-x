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
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// JWT implements auth.Authenticator using JSON Web Tokens.
type JWT struct {
	opts   *Options
	store  Store
	method jwt.SigningMethod
}

// Option is a functional option for JWT authenticator.
type Option func(*JWT)

// New creates a new JWT authenticator.
func New(opts ...Option) (*JWT, error) {
	j := &JWT{
		opts: NewOptions(),
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
func WithOptions(opts *Options) Option {
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

// WithMaxRefresh sets the maximum refresh duration.
func WithMaxRefresh(d time.Duration) Option {
	return func(j *JWT) {
		j.opts.MaxRefresh = d
	}
}

// WithAudience sets the token audience.
func WithAudience(audience ...string) Option {
	return func(j *JWT) {
		j.opts.Audience = audience
	}
}

// WithPublicKey sets the public key for RSA/ECDSA algorithms.
func WithPublicKey(key string) Option {
	return func(j *JWT) {
		j.opts.PublicKey = key
	}
}

// WithKeyID sets the key identifier.
func WithKeyID(kid string) Option {
	return func(j *JWT) {
		j.opts.KeyID = kid
	}
}

// Type returns the authenticator type.
func (j *JWT) Type() string {
	return "jwt"
}

// IsDisabled returns true if authentication is disabled.
func (j *JWT) IsDisabled() bool {
	return j.opts.DisableAuth
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
		var err error
		tokenID, err = generateTokenID()
		if err != nil {
			return nil, err
		}
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
	signingKey, err := j.getSigningKey()
	if err != nil {
		return nil, err
	}
	tokenString, err := token.SignedString(signingKey)
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
		return j.getVerifyingKey()
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

	// Determine subject based on IdentityKey
	subject := claims.Subject
	if j.opts.IdentityKey != "" && j.opts.IdentityKey != "sub" {
		if val, ok := claims.Extra[j.opts.IdentityKey]; ok {
			if s, ok := val.(string); ok && s != "" {
				subject = s
			} else {
				// Handle non-string claims (e.g. numeric ID)
				subject = fmt.Sprintf("%v", val)
			}
		}
	}

	return &auth.Claims{
		Subject:   subject,
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
// Security: Revokes old token and generates new token ID to prevent session fixation attacks.
func (j *JWT) Refresh(ctx context.Context, tokenString string) (auth.Token, error) {
	// 1. Verify the token is valid for refresh
	claims, err := j.verifyForRefresh(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// 2. Check if refresh is allowed (within max refresh window)
	if err := j.checkRefreshWindow(claims); err != nil {
		return nil, err
	}

	// 3. Revoke old token (non-blocking - log warnings on failure)
	j.revokeOldToken(ctx, tokenString)

	// 4. Generate new token with new ID (do not reuse old TokenID)
	return j.generateRefreshToken(ctx, claims)
}

// checkRefreshWindow validates if the token is within the allowed refresh window.
func (j *JWT) checkRefreshWindow(claims *auth.Claims) error {
	issuedAt := time.Unix(claims.IssuedAt, 0)
	maxRefreshTime := issuedAt.Add(j.opts.MaxRefresh)
	if time.Now().After(maxRefreshTime) {
		return errors.ErrSessionExpired.WithMessage("token refresh window expired")
	}
	return nil
}

// revokeOldToken attempts to revoke the old token.
// Failures are logged but do not block the refresh process.
func (j *JWT) revokeOldToken(ctx context.Context, tokenString string) {
	if j.store == nil {
		return
	}

	if err := j.Revoke(ctx, tokenString); err != nil {
		// Log warning but don't block refresh
		// Using structured logging for better observability
		logger.Warnw("failed to revoke old token during refresh",
			"error", err,
			"tokenPrefix", tokenPrefix(tokenString))
	}
}

// generateRefreshToken creates a new token with same claims but new ID.
func (j *JWT) generateRefreshToken(ctx context.Context, claims *auth.Claims) (auth.Token, error) {
	// Build sign options with existing claims (but NOT the old TokenID)
	signOpts := []auth.SignOption{}
	if len(claims.Extra) > 0 {
		signOpts = append(signOpts, auth.WithExtra(claims.Extra))
	}
	if len(claims.Audience) > 0 {
		signOpts = append(signOpts, auth.WithAudience(claims.Audience...))
	}
	// Note: We deliberately do NOT pass auth.WithTokenID() here
	// This ensures a new token ID is generated, preventing session fixation

	return j.Sign(ctx, claims.Subject, signOpts...)
}

// tokenPrefix returns the first 16 characters of a token for logging purposes.
func tokenPrefix(tokenString string) string {
	if len(tokenString) > 16 {
		return tokenString[:16] + "..."
	}
	return tokenString
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
		return j.getVerifyingKey()
	})
	if err != nil {
		return nil, j.mapParseError(err)
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok {
		return nil, errors.ErrInvalidToken.WithMessage("invalid claims type")
	}

	// SECURITY FIX: Enforce MaxRefresh time check
	// Even though we allow expired tokens for refresh, we must enforce the MaxRefresh window
	if claims.IssuedAt == nil {
		return nil, errors.ErrInvalidToken.WithMessage("missing issued at claim")
	}
	issuedAt := claims.IssuedAt.Time
	maxRefreshTime := issuedAt.Add(j.opts.MaxRefresh)
	if time.Now().After(maxRefreshTime) {
		return nil, errors.ErrSessionExpired.WithMessage("token refresh period exceeded")
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

	// Determine subject based on IdentityKey
	subject := claims.Subject
	if j.opts.IdentityKey != "" && j.opts.IdentityKey != "sub" {
		if val, ok := claims.Extra[j.opts.IdentityKey]; ok {
			if s, ok := val.(string); ok && s != "" {
				subject = s
			} else {
				subject = fmt.Sprintf("%v", val)
			}
		}
	}

	return &auth.Claims{
		Subject:   subject,
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
// It stores the token in the revocation list until its MaxRefresh time expires.
// This prevents race conditions where a token expires between verification and revocation.
func (j *JWT) Revoke(ctx context.Context, tokenString string) error {
	if j.store == nil {
		return errors.ErrNotImplemented.WithMessage("token revocation requires a store")
	}

	if tokenString == "" {
		return errors.ErrInvalidToken.WithMessage("token is empty")
	}

	// Parse token without validation to extract claims
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if token.Method.Alg() != j.method.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Validate signature
		return j.getVerifyingKey()
	})
	if err != nil {
		return j.mapParseError(err)
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok {
		return errors.ErrInvalidToken.WithMessage("invalid claims type")
	}

	// Verify IssuedAt exists
	if claims.IssuedAt == nil {
		return errors.ErrInvalidToken.WithMessage("missing issued at claim")
	}

	// Calculate TTL until MaxRefresh time (not ExpiresAt)
	issuedAt := claims.IssuedAt.Time
	maxRefreshTime := issuedAt.Add(j.opts.MaxRefresh)
	ttl := time.Until(maxRefreshTime)

	// If beyond MaxRefresh window, token is already invalid
	if ttl <= 0 {
		return nil
	}

	// Store token in revocation list with TTL until MaxRefresh
	return j.store.Revoke(ctx, tokenString, ttl)
}

// getSigningKey returns the key used for signing.
func (j *JWT) getSigningKey() (interface{}, error) {
	// For HMAC algorithms, return the key as bytes
	if strings.HasPrefix(j.opts.SigningMethod, "HS") {
		return []byte(j.opts.Key), nil
	}

	// For RSA/ECDSA algorithms, parse PEM format private key
	block, _ := pem.Decode([]byte(j.opts.Key))
	if block == nil {
		return nil, errors.ErrInvalidParam.WithMessage("invalid private key PEM format")
	}

	// Parse RSA private key
	if strings.HasPrefix(j.opts.SigningMethod, "RS") {
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			// Try PKCS8 format as fallback
			pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err2 != nil {
				return nil, errors.ErrInvalidParam.WithCause(err).WithMessage("failed to parse RSA private key")
			}
			return pkcs8Key, nil
		}
		return key, nil
	}

	// Parse ECDSA private key
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format as fallback
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, errors.ErrInvalidParam.WithCause(err).WithMessage("failed to parse ECDSA private key")
		}
		return pkcs8Key, nil
	}
	return key, nil
}

// getVerifyingKey returns the key used for verification.
func (j *JWT) getVerifyingKey() (interface{}, error) {
	// For HMAC, use the same key for verification
	if strings.HasPrefix(j.opts.SigningMethod, "HS") {
		return []byte(j.opts.Key), nil
	}

	// For RSA/ECDSA, require public key
	if j.opts.PublicKey == "" {
		return nil, errors.ErrInvalidParam.WithMessage("public key required for RSA/ECDSA verification")
	}

	// Parse PEM format public key
	block, _ := pem.Decode([]byte(j.opts.PublicKey))
	if block == nil {
		return nil, errors.ErrInvalidParam.WithMessage("invalid public key PEM format")
	}

	// Parse public key
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errors.ErrInvalidParam.WithCause(err).WithMessage("failed to parse public key")
	}

	return key, nil
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
func generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", errors.ErrInternal.WithCause(err).WithMessage("failed to generate token ID")
	}
	return hex.EncodeToString(b), nil
}
