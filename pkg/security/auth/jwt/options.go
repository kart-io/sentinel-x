// Package jwt provides JWT configuration options for Sentinel-X.
//
// This package implements JWT options following the onex project design pattern.
// It supports configuration via config files, environment variables, and command-line flags.
//
// Configuration Example (YAML):
//
//	jwt:
//	  key: "your-secret-key-must-be-at-least-64-chars-long-for-security-purposes!!"
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
	"os"
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
	// Increased to 64 characters (512 bits) to provide adequate security
	// against brute force attacks for HMAC-based JWT signing.
	MinKeyLength = 64

	// RecommendedKeyLength is the recommended key length for enhanced security.
	// 128 characters (1024 bits) provides additional security margin.
	RecommendedKeyLength = 128

	// MaxKeyLength is the maximum allowed key length.
	// Prevents excessively large keys that could impact performance.
	MaxKeyLength = 512
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
	// Minimum length: 64 characters for HMAC algorithms (512 bits).
	// Recommended length: 128 characters (1024 bits) for enhanced security.
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

	// For HMAC algorithms, validate key length and warn if weak
	if o.isHMAC() {
		if err := validateKeyLength(o.Key); err != nil {
			return err
		}
		warnWeakKeyStrength(o.Key)
	}

	return nil
}

// validateKeyLength validates that the key meets minimum length requirements.
func validateKeyLength(key string) error {
	if len(key) < MinKeyLength {
		return fmt.Errorf("jwt key must be at least %d characters for HMAC algorithms, got: %d",
			MinKeyLength, len(key))
	}
	if len(key) > MaxKeyLength {
		return fmt.Errorf("jwt key must be at most %d characters, got: %d",
			MaxKeyLength, len(key))
	}
	return nil
}

// warnWeakKeyStrength warns if the key length is below recommended strength.
func warnWeakKeyStrength(key string) {
	if len(key) >= MinKeyLength && len(key) < RecommendedKeyLength {
		fmt.Fprintf(os.Stderr, "WARNING: JWT key length (%d) is below recommended length (%d). "+
			"Consider using a stronger key for enhanced security.\n",
			len(key), RecommendedKeyLength)
	}
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
		"JWT signing key (min 64 chars for HMAC algorithms, 128 chars recommended)")
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
