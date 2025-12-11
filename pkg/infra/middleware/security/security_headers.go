package security

import (
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

// SecurityHeadersConfig defines the configuration for security headers middleware.
type SecurityHeadersConfig struct {
	// XFrameOptions protects against clickjacking attacks.
	// Default: "DENY"
	// Valid values: "DENY", "SAMEORIGIN", "ALLOW-FROM uri"
	XFrameOptions string

	// XContentTypeOptions prevents MIME type sniffing.
	// Default: "nosniff"
	// Valid value: "nosniff"
	XContentTypeOptions string

	// XXSSProtection enables browser XSS protection.
	// Default: "1; mode=block"
	// Valid values: "0", "1", "1; mode=block"
	XXSSProtection string

	// ContentSecurityPolicy defines CSP rules.
	// Default: "default-src 'self'"
	// Example: "default-src 'self'; script-src 'self' 'unsafe-inline'"
	ContentSecurityPolicy string

	// ReferrerPolicy controls referrer information.
	// Default: "strict-origin-when-cross-origin"
	// Valid values: "no-referrer", "no-referrer-when-downgrade", "origin",
	// "origin-when-cross-origin", "same-origin", "strict-origin",
	// "strict-origin-when-cross-origin", "unsafe-url"
	ReferrerPolicy string

	// StrictTransportSecurity configures HSTS.
	// Default: "max-age=31536000; includeSubDomains"
	// Example: "max-age=31536000; includeSubDomains; preload"
	StrictTransportSecurity string

	// EnableHSTS determines whether to enable HSTS header.
	// Default: false (disabled by default for safety)
	// Note: HSTS should only be enabled over HTTPS connections
	EnableHSTS bool
}

// DefaultSecurityHeadersConfig returns the default security headers configuration.
var DefaultSecurityHeadersConfig = SecurityHeadersConfig{
	XFrameOptions:           "DENY",
	XContentTypeOptions:     "nosniff",
	XXSSProtection:          "1; mode=block",
	ContentSecurityPolicy:   "default-src 'self'",
	ReferrerPolicy:          "strict-origin-when-cross-origin",
	StrictTransportSecurity: "max-age=31536000; includeSubDomains",
	EnableHSTS:              false,
}

// SecurityHeaders returns a middleware that adds security headers with default configuration.
// This middleware helps protect against common web vulnerabilities including:
//   - Clickjacking (X-Frame-Options)
//   - MIME type sniffing (X-Content-Type-Options)
//   - XSS attacks (X-XSS-Protection)
//   - Various injection attacks (Content-Security-Policy)
//   - Referrer leakage (Referrer-Policy)
//   - Man-in-the-middle attacks (Strict-Transport-Security, when enabled)
func SecurityHeaders() transport.MiddlewareFunc {
	return SecurityHeadersWithConfig(DefaultSecurityHeadersConfig)
}

// SecurityHeadersWithConfig returns a security headers middleware with custom configuration.
// Each header can be customized individually through the config parameter.
func SecurityHeadersWithConfig(config SecurityHeadersConfig) transport.MiddlewareFunc {
	// Set defaults for empty values
	if config.XFrameOptions == "" {
		config.XFrameOptions = DefaultSecurityHeadersConfig.XFrameOptions
	}
	if config.XContentTypeOptions == "" {
		config.XContentTypeOptions = DefaultSecurityHeadersConfig.XContentTypeOptions
	}
	if config.XXSSProtection == "" {
		config.XXSSProtection = DefaultSecurityHeadersConfig.XXSSProtection
	}
	if config.ContentSecurityPolicy == "" {
		config.ContentSecurityPolicy = DefaultSecurityHeadersConfig.ContentSecurityPolicy
	}
	if config.ReferrerPolicy == "" {
		config.ReferrerPolicy = DefaultSecurityHeadersConfig.ReferrerPolicy
	}
	if config.StrictTransportSecurity == "" {
		config.StrictTransportSecurity = DefaultSecurityHeadersConfig.StrictTransportSecurity
	}

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			// Set X-Frame-Options header
			setXFrameOptions(c, config.XFrameOptions)

			// Set X-Content-Type-Options header
			setXContentTypeOptions(c, config.XContentTypeOptions)

			// Set X-XSS-Protection header
			setXXSSProtection(c, config.XXSSProtection)

			// Set Content-Security-Policy header
			setContentSecurityPolicy(c, config.ContentSecurityPolicy)

			// Set Referrer-Policy header
			setReferrerPolicy(c, config.ReferrerPolicy)

			// Set HSTS header if enabled and connection is HTTPS
			if config.EnableHSTS {
				isHTTPS := isHTTPSConnection(c)
				setHSTS(c, config.StrictTransportSecurity, isHTTPS)
			}

			next(c)
		}
	}
}

// setXFrameOptions sets the X-Frame-Options header to protect against clickjacking attacks.
// This header indicates whether the browser should allow the page to be displayed in a frame.
//
// Parameters:
//   - c: The transport context
//   - value: The X-Frame-Options value (e.g., "DENY", "SAMEORIGIN", "ALLOW-FROM uri")
func setXFrameOptions(c transport.Context, value string) {
	if value != "" {
		c.SetHeader(HeaderXFrameOptions, value)
	}
}

// setXContentTypeOptions sets the X-Content-Type-Options header to prevent MIME type sniffing.
// This header prevents browsers from MIME-sniffing a response away from the declared content-type.
//
// Parameters:
//   - c: The transport context
//   - value: The X-Content-Type-Options value (should be "nosniff")
func setXContentTypeOptions(c transport.Context, value string) {
	if value != "" {
		c.SetHeader(HeaderXContentTypeOptions, value)
	}
}

// setXXSSProtection sets the X-XSS-Protection header to enable browser XSS filtering.
// This header enables the cross-site scripting (XSS) filter built into most browsers.
//
// Parameters:
//   - c: The transport context
//   - value: The X-XSS-Protection value (e.g., "0", "1", "1; mode=block")
func setXXSSProtection(c transport.Context, value string) {
	if value != "" {
		c.SetHeader(HeaderXXSSProtection, value)
	}
}

// setContentSecurityPolicy sets the Content-Security-Policy header to prevent various injection attacks.
// CSP helps detect and mitigate certain types of attacks including XSS and data injection attacks.
//
// Parameters:
//   - c: The transport context
//   - value: The CSP policy string (e.g., "default-src 'self'; script-src 'self' 'unsafe-inline'")
func setContentSecurityPolicy(c transport.Context, value string) {
	if value != "" {
		c.SetHeader(HeaderContentSecurityPolicy, value)
	}
}

// setReferrerPolicy sets the Referrer-Policy header to control referrer information.
// This header controls how much referrer information should be included with requests.
//
// Parameters:
//   - c: The transport context
//   - value: The Referrer-Policy value (e.g., "no-referrer", "strict-origin-when-cross-origin")
func setReferrerPolicy(c transport.Context, value string) {
	if value != "" {
		c.SetHeader(HeaderReferrerPolicy, value)
	}
}

// setHSTS sets the Strict-Transport-Security header to enforce HTTPS connections.
// HSTS tells browsers to only connect to the site using HTTPS, even if the user types http://.
//
// Parameters:
//   - c: The transport context
//   - value: The HSTS value (e.g., "max-age=31536000; includeSubDomains; preload")
//   - isHTTPS: Whether the current connection is HTTPS
//
// Note: HSTS header is only set when isHTTPS is true to prevent security warnings.
func setHSTS(c transport.Context, value string, isHTTPS bool) {
	// Only set HSTS header over HTTPS connections
	// Setting HSTS over HTTP would be ignored by browsers and could cause confusion
	if value != "" && isHTTPS {
		c.SetHeader(HeaderStrictTransportSecurity, value)
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
	if strings.ToLower(proto) == "https" {
		return true
	}

	return false
}
