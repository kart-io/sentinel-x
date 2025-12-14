// Package auth provides authentication middleware.
package auth

import (
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// Options defines authentication middleware options.
type Options struct {
	// Authenticator is the authenticator to use.
	Authenticator auth.Authenticator

	// TokenLookup defines how to extract the token.
	// Format: "header:<name>" or "query:<name>" or "cookie:<name>"
	// Default: "header:Authorization"
	TokenLookup string

	// AuthScheme is the authorization scheme (e.g., "Bearer").
	// Default: "Bearer"
	AuthScheme string

	// SkipPaths is a list of paths to skip authentication.
	SkipPaths []string

	// SkipPathPrefixes is a list of path prefixes to skip authentication.
	SkipPathPrefixes []string

	// ErrorHandler is called when authentication fails.
	// If nil, default error response is returned.
	ErrorHandler func(ctx transport.Context, err error)

	// SuccessHandler is called after successful authentication.
	// Can be used for custom context injection.
	SuccessHandler func(ctx transport.Context, claims *auth.Claims)
}

// Option is a functional option for auth middleware.
type Option func(*Options)

// NewOptions creates default auth options.
func NewOptions() *Options {
	return &Options{
		TokenLookup:      "header:Authorization",
		AuthScheme:       "Bearer",
		SkipPaths:        []string{},
		SkipPathPrefixes: []string{},
	}
}

// WithAuthenticator sets the authenticator.
func WithAuthenticator(a auth.Authenticator) Option {
	return func(o *Options) {
		o.Authenticator = a
	}
}

// WithTokenLookup sets how to extract the token.
func WithTokenLookup(lookup string) Option {
	return func(o *Options) {
		o.TokenLookup = lookup
	}
}

// WithAuthScheme sets the authorization scheme.
func WithAuthScheme(scheme string) Option {
	return func(o *Options) {
		o.AuthScheme = scheme
	}
}

// WithSkipPaths sets paths to skip authentication.
func WithSkipPaths(paths ...string) Option {
	return func(o *Options) {
		o.SkipPaths = paths
	}
}

// WithSkipPathPrefixes sets path prefixes to skip authentication.
func WithSkipPathPrefixes(prefixes ...string) Option {
	return func(o *Options) {
		o.SkipPathPrefixes = prefixes
	}
}

// WithErrorHandler sets the error handler.
func WithErrorHandler(handler func(ctx transport.Context, err error)) Option {
	return func(o *Options) {
		o.ErrorHandler = handler
	}
}

// WithSuccessHandler sets the success handler.
func WithSuccessHandler(handler func(ctx transport.Context, claims *auth.Claims)) Option {
	return func(o *Options) {
		o.SuccessHandler = handler
	}
}

// Auth creates an authentication middleware.
func Auth(opts ...Option) transport.MiddlewareFunc {
	options := NewOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Parse token lookup
	lookup := parseTokenLookup(options.TokenLookup)

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(ctx transport.Context) {
			// Check if path should be skipped
			path := ctx.HTTPRequest().URL.Path
			if shouldSkipAuth(path, options.SkipPaths, options.SkipPathPrefixes) {
				next(ctx)
				return
			}

			// Check if authenticator is configured
			if options.Authenticator == nil {
				handleAuthError(ctx, options, errors.ErrInternal.WithMessage("authenticator not configured"))
				return
			}

			// Extract token
			tokenString := extractToken(ctx, lookup, options.AuthScheme)
			if tokenString == "" {
				handleAuthError(ctx, options, errors.ErrUnauthorized.WithMessage("missing authentication token"))
				return
			}

			// Verify token
			claims, err := options.Authenticator.Verify(ctx.Request(), tokenString)
			if err != nil {
				// Log authentication failure for security audit
				logAuthFailure(ctx, tokenString, err)
				handleAuthError(ctx, options, err)
				return
			}

			// Debug: log successful auth
			logger.Infow("authentication successful",
				"subject", claims.Subject,
				"path", path,
			)

			// Inject claims into context
			newCtx := auth.InjectAuth(ctx.Request(), claims, tokenString)
			ctx.SetRequest(newCtx)

			// Call success handler if set
			if options.SuccessHandler != nil {
				options.SuccessHandler(ctx, claims)
			}

			next(ctx)
		}
	}
}

// tokenLookup represents a token extraction method.
type tokenLookup struct {
	source string // "header", "query", "cookie"
	name   string // name of the header/query/cookie
}

// parseTokenLookup parses the token lookup string.
func parseTokenLookup(lookup string) tokenLookup {
	parts := strings.SplitN(lookup, ":", 2)
	if len(parts) != 2 {
		return tokenLookup{source: "header", name: "Authorization"}
	}
	return tokenLookup{source: parts[0], name: parts[1]}
}

// extractToken extracts the token from the request.
func extractToken(ctx transport.Context, lookup tokenLookup, scheme string) string {
	var token string

	switch lookup.source {
	case "header":
		token = ctx.Header(lookup.name)
		if scheme != "" && strings.HasPrefix(token, scheme+" ") {
			token = strings.TrimPrefix(token, scheme+" ")
		}
	case "query":
		token = ctx.Query(lookup.name)
	case "cookie":
		if cookie, err := ctx.HTTPRequest().Cookie(lookup.name); err == nil {
			token = cookie.Value
		}
	}

	// Sanitize token
	token = strings.ReplaceAll(token, " ", "")
	token = strings.ReplaceAll(token, "+", "-")
	token = strings.ReplaceAll(token, "/", "_")
	token = strings.TrimRight(token, "=")
	return token
}

// shouldSkipAuth checks if the path should skip authentication.
func shouldSkipAuth(path string, skipPaths, skipPrefixes []string) bool {
	// Check exact match
	for _, p := range skipPaths {
		if path == p {
			return true
		}
	}

	// Check prefix match
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// handleAuthError handles authentication errors.
func handleAuthError(ctx transport.Context, options *Options, err error) {
	if options.ErrorHandler != nil {
		options.ErrorHandler(ctx, err)
		return
	}

	// Default error handling
	errno := errors.FromError(err)
	ctx.JSON(errno.HTTPStatus(), response.Err(errno))
}

// logAuthFailure logs authentication failures for security audit.
// This helps detect brute force attacks, token forgery attempts, and other security issues.
func logAuthFailure(ctx transport.Context, token string, err error) {
	// Get request information
	req := ctx.HTTPRequest()
	if req == nil {
		return
	}

	// Only record token prefix to avoid leaking complete token in logs
	tokenPrefix := ""
	if len(token) > 20 {
		tokenPrefix = token[:20] + "..."
	} else if len(token) > 0 {
		tokenPrefix = token[:len(token)/2] + "..."
	}

	// Log using structured logger with security-relevant fields
	logger.Warnw("authentication failed",
		"error", err.Error(),
		"remote_addr", req.RemoteAddr,
		"token_prefix", tokenPrefix,
		"path", req.URL.Path,
		"method", req.Method,
		"user_agent", req.UserAgent(),
	)
}
