package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/options"
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
// 纯配置选项，不包含运行时依赖（Authenticator、ErrorHandler、SuccessHandler）。
type AuthOptions struct {
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

// AuthzOptions defines authorization middleware options.
// 纯配置选项，不包含运行时依赖（Authorizer、提取器、ErrorHandler）。
type AuthzOptions struct {
	// SkipPaths is a list of paths to skip authorization.
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`

	// SkipPathPrefixes is a list of path prefixes to skip authorization.
	SkipPathPrefixes []string `json:"skip-path-prefixes" mapstructure:"skip-path-prefixes"`
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
