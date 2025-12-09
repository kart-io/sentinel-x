package auth

import (
	"context"
)

// contextKey is the type for context keys in this package.
type contextKey string

const (
	// claimsKey is the context key for storing Claims.
	claimsKey contextKey = "auth:claims"

	// subjectKey is the context key for storing subject (user ID).
	subjectKey contextKey = "auth:subject"

	// tokenKey is the context key for storing the raw token string.
	tokenKey contextKey = "auth:token"
)

// ContextWithClaims returns a new context with the given claims.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext returns the claims from the context.
// Returns nil if no claims are found.
func ClaimsFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(claimsKey).(*Claims); ok {
		return claims
	}
	return nil
}

// ContextWithSubject returns a new context with the given subject (user ID).
func ContextWithSubject(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, subjectKey, subject)
}

// SubjectFromContext returns the subject (user ID) from the context.
// Returns empty string if no subject is found.
func SubjectFromContext(ctx context.Context) string {
	if subject, ok := ctx.Value(subjectKey).(string); ok {
		return subject
	}
	return ""
}

// ContextWithToken returns a new context with the given token string.
func ContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// TokenFromContext returns the token string from the context.
// Returns empty string if no token is found.
func TokenFromContext(ctx context.Context) string {
	if token, ok := ctx.Value(tokenKey).(string); ok {
		return token
	}
	return ""
}

// InjectAuth injects all authentication information into the context.
func InjectAuth(ctx context.Context, claims *Claims, token string) context.Context {
	ctx = ContextWithClaims(ctx, claims)
	ctx = ContextWithToken(ctx, token)
	if claims != nil {
		ctx = ContextWithSubject(ctx, claims.Subject)
	}
	return ctx
}

// MustSubjectFromContext returns the subject from context or panics.
func MustSubjectFromContext(ctx context.Context) string {
	subject := SubjectFromContext(ctx)
	if subject == "" {
		panic("auth: subject not found in context")
	}
	return subject
}

// MustClaimsFromContext returns the claims from context or panics.
func MustClaimsFromContext(ctx context.Context) *Claims {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		panic("auth: claims not found in context")
	}
	return claims
}
