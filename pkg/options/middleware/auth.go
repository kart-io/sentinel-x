package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareAuth, func() MiddlewareConfig {
		return NewAuthOptions()
	})
	Register(MiddlewareAuthz, func() MiddlewareConfig {
		return NewAuthzOptions()
	})
}

// 确保 AuthOptions 和 AuthzOptions 实现 MiddlewareConfig 接口。
var (
	_ MiddlewareConfig = (*AuthOptions)(nil)
	_ MiddlewareConfig = (*AuthzOptions)(nil)
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

// NewAuthOptions creates default authentication options.
func NewAuthOptions() *AuthOptions {
	return &AuthOptions{
		TokenLookup:      "header:Authorization",
		AuthScheme:       "Bearer",
		SkipPaths:        []string{},
		SkipPathPrefixes: []string{},
	}
}

// AddFlags adds flags for auth options to the specified FlagSet.
func (o *AuthOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.TokenLookup, options.Join(prefixes...)+"middleware.auth.token-lookup", o.TokenLookup, "Token lookup location (header:Authorization)")
	fs.StringVar(&o.AuthScheme, options.Join(prefixes...)+"middleware.auth.auth-scheme", o.AuthScheme, "Authorization scheme (Bearer)")
	fs.StringSliceVar(&o.SkipPaths, options.Join(prefixes...)+"middleware.auth.skip-paths", o.SkipPaths, "Paths to skip authentication")
	fs.StringSliceVar(&o.SkipPathPrefixes, options.Join(prefixes...)+"middleware.auth.skip-path-prefixes", o.SkipPathPrefixes, "Path prefixes to skip authentication")
}

// Validate validates the auth options.
func (o *AuthOptions) Validate() []error {
	if o == nil {
		return nil
	}
	return nil
}

// Complete completes the auth options with defaults.
func (o *AuthOptions) Complete() error {
	return nil
}

// WithAuthenticator configures and enables auth middleware with an authenticator.
func WithAuthenticator(authenticator auth.Authenticator, skipPaths ...string) Option {
	return func(o *Options) {
		if o.Auth == nil {
			o.Auth = NewAuthOptions()
		}
		o.Auth.Authenticator = authenticator
		if len(skipPaths) > 0 {
			o.Auth.SkipPaths = skipPaths
		}
	}
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

// NewAuthzOptions creates default authorization options.
func NewAuthzOptions() *AuthzOptions {
	return &AuthzOptions{
		SkipPaths:        []string{},
		SkipPathPrefixes: []string{},
	}
}

// AddFlags adds flags for authz options to the specified FlagSet.
func (o *AuthzOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringSliceVar(&o.SkipPaths, options.Join(prefixes...)+"middleware.authz.skip-paths", o.SkipPaths, "Paths to skip authorization")
	fs.StringSliceVar(&o.SkipPathPrefixes, options.Join(prefixes...)+"middleware.authz.skip-path-prefixes", o.SkipPathPrefixes, "Path prefixes to skip authorization")
}

// Validate validates the authz options.
func (o *AuthzOptions) Validate() []error {
	if o == nil {
		return nil
	}
	return nil
}

// Complete completes the authz options with defaults.
func (o *AuthzOptions) Complete() error {
	return nil
}

// WithAuthorizer configures and enables authz middleware with custom skip paths.
func WithAuthorizer(authorizer authz.Authorizer, skipPaths ...string) Option {
	return func(o *Options) {
		if o.Authz == nil {
			o.Authz = NewAuthzOptions()
		}
		o.Authz.Authorizer = authorizer
		if len(skipPaths) > 0 {
			o.Authz.SkipPaths = skipPaths
		}
	}
}
