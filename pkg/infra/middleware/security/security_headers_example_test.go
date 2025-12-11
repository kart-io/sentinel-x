package security_test

import (
	"fmt"
	"net/http"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// ExampleSecurityHeaders demonstrates the basic usage of security headers middleware.
func ExampleSecurityHeaders() {
	// Create middleware with default security headers
	securityMiddleware := middleware.SecurityHeaders()

	// Apply middleware to a handler
	handler := securityMiddleware(func(c transport.Context) {
		c.String(http.StatusOK, "Secure response")
	})

	// Use the handler in your router
	_ = handler
	fmt.Println("Security headers middleware applied with defaults")
	// Output: Security headers middleware applied with defaults
}

// ExampleSecurityHeadersWithConfig demonstrates custom configuration for security headers.
func ExampleSecurityHeadersWithConfig() {
	// Create custom configuration
	config := middleware.SecurityHeadersConfig{
		XFrameOptions:           "SAMEORIGIN", // Allow same-origin framing
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		ReferrerPolicy:          "no-referrer",
		StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
		EnableHSTS:              true, // Enable HSTS for production HTTPS sites
	}

	// Create middleware with custom configuration
	securityMiddleware := middleware.SecurityHeadersWithConfig(config)

	// Apply middleware to a handler
	handler := securityMiddleware(func(c transport.Context) {
		c.String(http.StatusOK, "Secure response with custom headers")
	})

	// Use the handler in your router
	_ = handler
	fmt.Println("Security headers middleware applied with custom configuration")
	// Output: Security headers middleware applied with custom configuration
}

// ExampleSecurityHeadersWithConfig_development demonstrates a configuration for development environment.
func ExampleSecurityHeadersWithConfig_development() {
	// Development configuration (more relaxed)
	config := middleware.SecurityHeadersConfig{
		XFrameOptions:         "SAMEORIGIN",
		XContentTypeOptions:   "nosniff",
		XXSSProtection:        "1; mode=block",
		ContentSecurityPolicy: "default-src 'self' 'unsafe-inline' 'unsafe-eval'", // Allow inline scripts/styles for development
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		EnableHSTS:            false, // Disable HSTS in development
	}

	securityMiddleware := middleware.SecurityHeadersWithConfig(config)
	_ = securityMiddleware
	fmt.Println("Development environment security headers configured")
	// Output: Development environment security headers configured
}

// ExampleSecurityHeadersWithConfig_production demonstrates a configuration for production environment.
func ExampleSecurityHeadersWithConfig_production() {
	// Production configuration (strict security)
	config := middleware.SecurityHeadersConfig{
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'",
		ReferrerPolicy:          "no-referrer",
		StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
		EnableHSTS:              true, // Enable HSTS with preload for maximum security
	}

	securityMiddleware := middleware.SecurityHeadersWithConfig(config)
	_ = securityMiddleware
	fmt.Println("Production environment security headers configured")
	// Output: Production environment security headers configured
}

// ExampleSecurityHeadersWithConfig_api demonstrates a configuration for API servers.
func ExampleSecurityHeadersWithConfig_api() {
	// API server configuration
	config := middleware.SecurityHeadersConfig{
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ContentSecurityPolicy:   "default-src 'none'; frame-ancestors 'none'", // Minimal CSP for APIs
		ReferrerPolicy:          "no-referrer",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		EnableHSTS:              true,
	}

	securityMiddleware := middleware.SecurityHeadersWithConfig(config)
	_ = securityMiddleware
	fmt.Println("API server security headers configured")
	// Output: API server security headers configured
}
