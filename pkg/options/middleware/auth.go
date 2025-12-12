package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
)

// AuthOptions defines authentication middleware options.
type AuthOptions struct {
	// Authenticator is the authenticator to use.
	Authenticator auth.Authenticator `json:"-" mapstructure:"-"`

	// TokenLookup defines how to extract the token.
	// Format: "header:<name>" or "query:<name>" or "cookie:<name>"
	// Default: "header:Authorization"
	TokenLookup string `json:"token-lookup" mapstructure:"token-lookup"`

	// AuthScheme is the authorization scheme (e.g., "Bearer").
	// Default: "Bearer"
	AuthScheme string `json:"auth-scheme" mapstructure:"auth-scheme"`

	// SkipPaths is a list of paths to skip authentication.
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`

	// SkipPathPrefixes is a list of path prefixes to skip authentication.
	SkipPathPrefixes []string `json:"skip-path-prefixes" mapstructure:"skip-path-prefixes"`

	// ErrorHandler is called when authentication fails.
	ErrorHandler func(ctx transport.Context, err error) `json:"-" mapstructure:"-"`

	// SuccessHandler is called after successful authentication.
	SuccessHandler func(ctx transport.Context, claims *auth.Claims) `json:"-" mapstructure:"-"`
}

// WithAuth configures and enables auth middleware.
func WithAuth(authenticator auth.Authenticator, skipPaths ...string) Option {
	return func(o *Options) {
		o.DisableAuth = false
		o.Auth.Authenticator = authenticator
		if len(skipPaths) > 0 {
			o.Auth.SkipPaths = skipPaths
		}
	}
}

// WithoutAuth disables auth middleware.
func WithoutAuth() Option {
	return func(o *Options) { o.DisableAuth = true }
}

// AuthzOptions defines authorization middleware options.
type AuthzOptions struct {
	// Authorizer is the authorizer to use.
	Authorizer authz.Authorizer `json:"-" mapstructure:"-"`

	// ResourceExtractor extracts the resource from the request.
	// Default: extracts from request path.
	ResourceExtractor func(ctx transport.Context) string `json:"-" mapstructure:"-"`

	// ActionExtractor extracts the action from the request.
	// Default: maps HTTP method to action (GET->read, POST->create, etc.).
	ActionExtractor func(ctx transport.Context) string `json:"-" mapstructure:"-"`

	// SubjectExtractor extracts the subject from the request.
	// Default: extracts from auth claims in context.
	SubjectExtractor func(ctx transport.Context) string `json:"-" mapstructure:"-"`

	// SkipPaths is a list of paths to skip authorization.
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`

	// SkipPathPrefixes is a list of path prefixes to skip authorization.
	SkipPathPrefixes []string `json:"skip-path-prefixes" mapstructure:"skip-path-prefixes"`

	// ErrorHandler is called when authorization fails.
	ErrorHandler func(ctx transport.Context, err error) `json:"-" mapstructure:"-"`
}

// WithAuthz configures and enables authz middleware.
func WithAuthz(skipPaths ...string) Option {
	return func(o *Options) {
		o.DisableAuthz = false
		if len(skipPaths) > 0 {
			o.Authz.SkipPaths = skipPaths
		}
	}
}

// WithoutAuthz disables authz middleware.
func WithoutAuthz() Option {
	return func(o *Options) { o.DisableAuthz = true }
}
