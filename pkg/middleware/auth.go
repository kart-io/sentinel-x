package middleware

import (
	"strings"

	"github.com/kart-io/sentinel-x/pkg/auth"
	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/response"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// AuthOptions defines authentication middleware options.
type AuthOptions struct {
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

// AuthOption is a functional option for auth middleware.
type AuthOption func(*AuthOptions)

// NewAuthOptions creates default auth options.
func NewAuthOptions() *AuthOptions {
	return &AuthOptions{
		TokenLookup:      "header:Authorization",
		AuthScheme:       "Bearer",
		SkipPaths:        []string{},
		SkipPathPrefixes: []string{},
	}
}

// AuthWithAuthenticator sets the authenticator.
func AuthWithAuthenticator(a auth.Authenticator) AuthOption {
	return func(o *AuthOptions) {
		o.Authenticator = a
	}
}

// AuthWithTokenLookup sets how to extract the token.
func AuthWithTokenLookup(lookup string) AuthOption {
	return func(o *AuthOptions) {
		o.TokenLookup = lookup
	}
}

// AuthWithAuthScheme sets the authorization scheme.
func AuthWithAuthScheme(scheme string) AuthOption {
	return func(o *AuthOptions) {
		o.AuthScheme = scheme
	}
}

// AuthWithSkipPaths sets paths to skip authentication.
func AuthWithSkipPaths(paths ...string) AuthOption {
	return func(o *AuthOptions) {
		o.SkipPaths = paths
	}
}

// AuthWithSkipPathPrefixes sets path prefixes to skip authentication.
func AuthWithSkipPathPrefixes(prefixes ...string) AuthOption {
	return func(o *AuthOptions) {
		o.SkipPathPrefixes = prefixes
	}
}

// AuthWithErrorHandler sets the error handler.
func AuthWithErrorHandler(handler func(ctx transport.Context, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.ErrorHandler = handler
	}
}

// AuthWithSuccessHandler sets the success handler.
func AuthWithSuccessHandler(handler func(ctx transport.Context, claims *auth.Claims)) AuthOption {
	return func(o *AuthOptions) {
		o.SuccessHandler = handler
	}
}

// Auth creates an authentication middleware.
func Auth(opts ...AuthOption) transport.MiddlewareFunc {
	options := NewAuthOptions()
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
				handleAuthError(ctx, options, err)
				return
			}

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

	return strings.TrimSpace(token)
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
func handleAuthError(ctx transport.Context, options *AuthOptions, err error) {
	if options.ErrorHandler != nil {
		options.ErrorHandler(ctx, err)
		return
	}

	// Default error handling
	errno := errors.FromError(err)
	ctx.JSON(errno.HTTPStatus(), response.Err(errno))
}

// WithAuth configures and enables auth middleware in Options.
func WithAuth(opts ...AuthOption) Option {
	return func(o *Options) {
		o.DisableAuth = false
		for _, opt := range opts {
			opt(&o.Auth)
		}
	}
}

// WithoutAuth disables auth middleware.
func WithoutAuth() Option {
	return func(o *Options) {
		o.DisableAuth = true
	}
}
