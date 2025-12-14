package security_test

import (
	"fmt"
	"net/http"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/security"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// ExampleSecurityHeaders demonstrates the basic usage of security headers middleware.
func ExampleSecurityHeaders() {
	// Create default security headers middleware
	mw := security.Headers()

	// Apply middleware to a handler
	handler := mw(func(c transport.Context) {
		c.String(http.StatusOK, "Secure response")
	})

	// Use the handler (pseudo-code)
	_ = handler
	fmt.Println("Security headers middleware applied")
	// Output: Security headers middleware applied
}

// ExampleSecurityHeadersWithConfig demonstrates custom configuration for security headers.
func ExampleSecurityHeadersWithConfig() {
	// Create custom configuration
	config := middleware.SecurityHeadersConfig{
		FrameOptionsValue:        "SAMEORIGIN",
		XSSProtectionValue:       "0",
		ContentSecurityPolicy:    "default-src 'self'",
		ReferrerPolicy:           "no-referrer",
		HSTSMaxAge:               31536000,
		HSTSIncludeSubdomains:    true,
		HSTSPreload:              true,
		EnableHSTS:               true,
		EnableFrameOptions:       true,
		EnableContentTypeOptions: true,
		EnableXSSProtection:      true,
	}

	// Create middleware with custom configuration
	// Note: middleware.SecurityHeadersWithConfig is an alias to security.HeadersWithConfig
	securityMiddleware := middleware.SecurityHeadersWithConfig(config)

	// Apply middleware to a handler
	handler := securityMiddleware(func(c transport.Context) {
		c.String(http.StatusOK, "Secure response with custom headers")
	})

	// Use the handler
	_ = handler
	fmt.Println("Security headers middleware applied with custom configuration")
	// Output: Security headers middleware applied with custom configuration
}

// ExampleSecurityHeadersWithConfig_development demonstrates a configuration for development environment.
func ExampleSecurityHeadersWithConfig_development() {
	// Development configuration (more relaxed)
	config := middleware.SecurityHeadersConfig{
		FrameOptionsValue:        "SAMEORIGIN",
		XSSProtectionValue:       "1; mode=block",
		ContentSecurityPolicy:    "default-src 'self' 'unsafe-inline' 'unsafe-eval'", // Allow inline scripts/styles for development
		ReferrerPolicy:           "strict-origin-when-cross-origin",
		EnableHSTS:               false, // Disable HSTS in development
		EnableFrameOptions:       true,
		EnableContentTypeOptions: true,
		EnableXSSProtection:      true,
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
		FrameOptionsValue:        "DENY",
		XSSProtectionValue:       "1; mode=block",
		ContentSecurityPolicy:    "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'",
		ReferrerPolicy:           "no-referrer",
		HSTSMaxAge:               63072000,
		HSTSIncludeSubdomains:    true,
		HSTSPreload:              true,
		EnableHSTS:               true, // Enable HSTS with preload for maximum security
		EnableFrameOptions:       true,
		EnableContentTypeOptions: true,
		EnableXSSProtection:      true,
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
		FrameOptionsValue:        "DENY",
		XSSProtectionValue:       "1; mode=block",
		ContentSecurityPolicy:    "default-src 'none'; frame-ancestors 'none'", // Minimal CSP for APIs
		ReferrerPolicy:           "no-referrer",
		HSTSMaxAge:               31536000,
		HSTSIncludeSubdomains:    true,
		EnableHSTS:               true,
		EnableFrameOptions:       true,
		EnableContentTypeOptions: true,
		EnableXSSProtection:      true,
	}

	securityMiddleware := middleware.SecurityHeadersWithConfig(config)
	_ = securityMiddleware
	fmt.Println("API server security headers configured")
	// Output: API server security headers configured
}
