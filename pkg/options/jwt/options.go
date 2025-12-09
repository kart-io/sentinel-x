// Package jwt provides JWT configuration options for Sentinel-X.
//
// This package implements JWT options following the onex project design pattern.
// It supports configuration via config files, environment variables, and command-line flags.
//
// Configuration Example (YAML):
//
//	jwt:
//	  key: "your-secret-key-min-32-chars-long"
//	  signing-method: "HS256"
//	  expired: "2h"
//	  max-refresh: "24h"
//	  issuer: "sentinel-x"
//	  audience: ["api"]
//
// Environment Variables:
//
//	JWT_KEY           - JWT signing key
//	JWT_SIGNING_METHOD - Signing algorithm
//	JWT_EXPIRED        - Token expiration duration
//	JWT_MAX_REFRESH    - Maximum refresh duration
//	JWT_ISSUER         - Token issuer
package jwt

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

const (
	// DefaultSigningMethod is the default JWT signing algorithm.
	DefaultSigningMethod = "HS256"

	// DefaultExpired is the default token expiration time.
	DefaultExpired = 2 * time.Hour

	// DefaultMaxRefresh is the default maximum refresh duration.
	DefaultMaxRefresh = 24 * time.Hour

	// DefaultIssuer is the default token issuer.
	DefaultIssuer = "sentinel-x"

	// MinKeyLength is the minimum required key length for security.
	MinKeyLength = 32

	// MaxKeyLength is the maximum allowed key length.
	MaxKeyLength = 256
)

// SupportedSigningMethods contains all supported JWT signing algorithms.
var SupportedSigningMethods = map[string]bool{
	"HS256": true,
	"HS384": true,
	"HS512": true,
	"RS256": true,
	"RS384": true,
	"RS512": true,
	"ES256": true,
	"ES384": true,
	"ES512": true,
}

// Options contains JWT configuration.
type Options struct {
	// DisableAuth disables JWT authentication.
	// When true, no JWT token is required for protected endpoints.
	// Default: false (authentication enabled by default for security)
	DisableAuth bool `json:"disable-auth" mapstructure:"disable-auth"`

	// Key is the secret key used to sign tokens.
	// For HMAC algorithms (HS256/HS384/HS512), this should be a secure random string.
	// For RSA/ECDSA algorithms, this should be the private key in PEM format.
	// Minimum length: 32 characters for HMAC algorithms.
	Key string `json:"key" mapstructure:"key"`

	// SigningMethod is the JWT signing algorithm.
	// Supported: HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512.
	// Default: HS256
	SigningMethod string `json:"signing-method" mapstructure:"signing-method"`

	// Expired is the token expiration duration.
	// Default: 2h
	Expired time.Duration `json:"expired" mapstructure:"expired"`

	// MaxRefresh is the maximum duration a token can be refreshed.
	// After this duration from the original issue time, the user must re-authenticate.
	// Default: 24h
	MaxRefresh time.Duration `json:"max-refresh" mapstructure:"max-refresh"`

	// Issuer is the token issuer (iss claim).
	// Default: sentinel-x
	Issuer string `json:"issuer" mapstructure:"issuer"`

	// Audience is the intended audience for the token (aud claim).
	// Can be a single value or multiple values.
	Audience []string `json:"audience" mapstructure:"audience"`

	// PublicKey is the public key for RSA/ECDSA algorithms (optional).
	// If not set, the Key field will be used for symmetric algorithms.
	PublicKey string `json:"public-key" mapstructure:"public-key"`

	// KeyID is an optional key identifier (kid header).
	// Useful for key rotation scenarios.
	KeyID string `json:"key-id" mapstructure:"key-id"`
}

// NewOptions creates a new Options with default values.
func NewOptions() *Options {
	return &Options{
		DisableAuth:   false, // Enabled by default for security
		SigningMethod: DefaultSigningMethod,
		Expired:       DefaultExpired,
		MaxRefresh:    DefaultMaxRefresh,
		Issuer:        DefaultIssuer,
		Audience:      []string{},
	}
}

// Validate validates the JWT options.
func (o *Options) Validate() error {
	// Skip validation if auth is disabled
	if o.DisableAuth {
		return nil
	}

	// Validate signing method
	if !SupportedSigningMethods[o.SigningMethod] {
		return fmt.Errorf("unsupported signing method: %s", o.SigningMethod)
	}

	// Validate key based on signing method
	if err := o.validateKey(); err != nil {
		return err
	}

	// Validate expiration
	if o.Expired <= 0 {
		return fmt.Errorf("expired must be positive, got: %v", o.Expired)
	}

	// Validate max refresh
	if o.MaxRefresh <= 0 {
		return fmt.Errorf("max-refresh must be positive, got: %v", o.MaxRefresh)
	}

	// MaxRefresh should be >= Expired
	if o.MaxRefresh < o.Expired {
		return fmt.Errorf("max-refresh (%v) must be >= expired (%v)", o.MaxRefresh, o.Expired)
	}

	return nil
}

// validateKey validates the signing key based on the algorithm.
func (o *Options) validateKey() error {
	if o.Key == "" {
		return fmt.Errorf("jwt key is required")
	}

	// For HMAC algorithms, validate key length
	if o.isHMAC() {
		if len(o.Key) < MinKeyLength {
			return fmt.Errorf("jwt key must be at least %d characters for HMAC algorithms, got: %d",
				MinKeyLength, len(o.Key))
		}
		if len(o.Key) > MaxKeyLength {
			return fmt.Errorf("jwt key must be at most %d characters, got: %d",
				MaxKeyLength, len(o.Key))
		}
	}

	return nil
}

// isHMAC returns true if the signing method is an HMAC algorithm.
func (o *Options) isHMAC() bool {
	return o.SigningMethod == "HS256" || o.SigningMethod == "HS384" || o.SigningMethod == "HS512"
}

// Complete fills in default values for unset fields.
func (o *Options) Complete() error {
	if o.SigningMethod == "" {
		o.SigningMethod = DefaultSigningMethod
	}
	if o.Expired == 0 {
		o.Expired = DefaultExpired
	}
	if o.MaxRefresh == 0 {
		o.MaxRefresh = DefaultMaxRefresh
	}
	if o.Issuer == "" {
		o.Issuer = DefaultIssuer
	}
	return nil
}

// AddFlags adds flags for JWT options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.DisableAuth, "jwt.disable-auth", o.DisableAuth,
		"Disable JWT authentication")
	fs.StringVar(&o.Key, "jwt.key", o.Key,
		"JWT signing key (min 32 chars for HMAC algorithms)")
	fs.StringVar(&o.SigningMethod, "jwt.signing-method", o.SigningMethod,
		"JWT signing algorithm (HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512)")
	fs.DurationVar(&o.Expired, "jwt.expired", o.Expired,
		"JWT token expiration duration")
	fs.DurationVar(&o.MaxRefresh, "jwt.max-refresh", o.MaxRefresh,
		"Maximum duration a token can be refreshed")
	fs.StringVar(&o.Issuer, "jwt.issuer", o.Issuer,
		"JWT token issuer (iss claim)")
	fs.StringSliceVar(&o.Audience, "jwt.audience", o.Audience,
		"JWT token audience (aud claim)")
	fs.StringVar(&o.PublicKey, "jwt.public-key", o.PublicKey,
		"JWT public key for RSA/ECDSA algorithms")
	fs.StringVar(&o.KeyID, "jwt.key-id", o.KeyID,
		"JWT key identifier (kid header)")
}

// Option is a function that configures Options.
type Option func(*Options)

// WithKey sets the signing key.
func WithKey(key string) Option {
	return func(o *Options) {
		o.Key = key
	}
}

// WithSigningMethod sets the signing algorithm.
func WithSigningMethod(method string) Option {
	return func(o *Options) {
		o.SigningMethod = method
	}
}

// WithExpired sets the token expiration duration.
func WithExpired(d time.Duration) Option {
	return func(o *Options) {
		o.Expired = d
	}
}

// WithMaxRefresh sets the maximum refresh duration.
func WithMaxRefresh(d time.Duration) Option {
	return func(o *Options) {
		o.MaxRefresh = d
	}
}

// WithIssuer sets the token issuer.
func WithIssuer(issuer string) Option {
	return func(o *Options) {
		o.Issuer = issuer
	}
}

// WithAudience sets the token audience.
func WithAudience(audience ...string) Option {
	return func(o *Options) {
		o.Audience = audience
	}
}

// WithPublicKey sets the public key for RSA/ECDSA algorithms.
func WithPublicKey(key string) Option {
	return func(o *Options) {
		o.PublicKey = key
	}
}

// WithKeyID sets the key identifier.
func WithKeyID(kid string) Option {
	return func(o *Options) {
		o.KeyID = kid
	}
}

// ApplyOptions applies functional options to the Options.
func (o *Options) ApplyOptions(opts ...Option) *Options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}
