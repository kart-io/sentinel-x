package security

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// Security header constants.
const (
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderReferrerPolicy          = "Referrer-Policy"
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
)

// HeadersConfig defines the configuration for security headers middleware.
type HeadersConfig struct {
	// EnableHSTS enables Strict-Transport-Security header.
	EnableHSTS bool
	// HSTSMaxAge is the HSTS max-age in seconds.
	HSTSMaxAge int
	// HSTSIncludeSubdomains includes subdomains in HSTS.
	HSTSIncludeSubdomains bool
	// HSTSPreload enables HSTS preload.
	HSTSPreload bool

	// EnableFrameOptions enables X-Frame-Options header.
	EnableFrameOptions bool
	// FrameOptionsValue is the value for X-Frame-Options (DENY, SAMEORIGIN).
	FrameOptionsValue string

	// EnableContentTypeOptions enables X-Content-Type-Options header.
	EnableContentTypeOptions bool

	// EnableXSSProtection enables X-XSS-Protection header.
	EnableXSSProtection bool
	// XSSProtectionValue is the value for X-XSS-Protection.
	XSSProtectionValue string

	// ContentSecurityPolicy is the value for Content-Security-Policy header.
	ContentSecurityPolicy string
	// ReferrerPolicy is the value for Referrer-Policy header.
	ReferrerPolicy string
}

// DefaultHeadersConfig returns the default configuration.
func DefaultHeadersConfig() HeadersConfig {
	return HeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false,

		EnableFrameOptions: true,
		FrameOptionsValue:  "DENY",

		EnableContentTypeOptions: true,

		EnableXSSProtection: true,
		XSSProtectionValue:  "1; mode=block",

		ContentSecurityPolicy: "", // Default to empty (user should configure)
		ReferrerPolicy:        "no-referrer",
	}
}

// Headers returns a middleware that adds security headers with default config.
func Headers() transport.MiddlewareFunc {
	return HeadersWithConfig(DefaultHeadersConfig())
}

// HeadersWithConfig returns a middleware that adds security headers with custom config.
func HeadersWithConfig(config HeadersConfig) transport.MiddlewareFunc {
	if reflect.DeepEqual(config, HeadersConfig{}) {
		config = DefaultHeadersConfig()
	}
	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			// Add HSTS header
			if config.EnableHSTS && isHTTPSConnection(c) {
				hsts := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
				if config.HSTSIncludeSubdomains {
					hsts += "; includeSubDomains"
				}
				if config.HSTSPreload {
					hsts += "; preload"
				}
				c.SetHeader("Strict-Transport-Security", hsts)
			}

			// Add X-Frame-Options header
			if config.EnableFrameOptions {
				c.SetHeader("X-Frame-Options", config.FrameOptionsValue)
			}

			// Add X-Content-Type-Options header
			if config.EnableContentTypeOptions {
				c.SetHeader("X-Content-Type-Options", "nosniff")
			}

			// Add X-XSS-Protection header
			if config.EnableXSSProtection {
				c.SetHeader("X-XSS-Protection", config.XSSProtectionValue)
			}

			// Add Content-Security-Policy header
			if config.ContentSecurityPolicy != "" {
				c.SetHeader("Content-Security-Policy", config.ContentSecurityPolicy)
			}

			// Add Referrer-Policy header
			if config.ReferrerPolicy != "" {
				c.SetHeader("Referrer-Policy", config.ReferrerPolicy)
			}

			next(c)
		}
	}
}

// isHTTPSConnection checks if the current connection is using HTTPS protocol.
// This function examines multiple indicators to determine HTTPS status:
//   - Direct TLS connection (req.TLS != nil)
//   - X-Forwarded-Proto header (for reverse proxy scenarios)
//
// Returns true if the connection is HTTPS, false otherwise.
func isHTTPSConnection(c transport.Context) bool {
	req := c.HTTPRequest()

	// Check if TLS is enabled
	if req.TLS != nil {
		return true
	}

	// Check X-Forwarded-Proto header (for reverse proxy scenarios)
	proto := req.Header.Get("X-Forwarded-Proto")
	return strings.ToLower(proto) == "https"
}
