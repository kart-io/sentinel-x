package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

func TestSecurityHeaders(t *testing.T) {
	middleware := SecurityHeaders()
	handler := middleware(func(c transport.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := newMockContext(req, rec)

	handler(ctx)

	// Verify default security headers
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "X-Frame-Options",
			header:   HeaderXFrameOptions,
			expected: "DENY",
		},
		{
			name:     "X-Content-Type-Options",
			header:   HeaderXContentTypeOptions,
			expected: "nosniff",
		},
		{
			name:     "X-XSS-Protection",
			header:   HeaderXXSSProtection,
			expected: "1; mode=block",
		},
		{
			name:     "Content-Security-Policy",
			header:   HeaderContentSecurityPolicy,
			expected: "default-src 'self'",
		},
		{
			name:     "Referrer-Policy",
			header:   HeaderReferrerPolicy,
			expected: "strict-origin-when-cross-origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rec.Header().Get(tt.header)
			if got != tt.expected {
				t.Errorf("header %s = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}

	// Verify HSTS is not set by default
	if got := rec.Header().Get(HeaderStrictTransportSecurity); got != "" {
		t.Errorf("HSTS header should not be set by default, got %q", got)
	}
}

func TestSecurityHeadersWithConfig(t *testing.T) {
	config := SecurityHeadersConfig{
		XFrameOptions:           "SAMEORIGIN",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1",
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self' 'unsafe-inline'",
		ReferrerPolicy:          "no-referrer",
		StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
		EnableHSTS:              true,
	}

	middleware := SecurityHeadersWithConfig(config)
	handler := middleware(func(c transport.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Test with HTTPS request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{} // Simulate HTTPS
	rec := httptest.NewRecorder()
	ctx := newMockContext(req, rec)

	handler(ctx)

	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "X-Frame-Options",
			header:   HeaderXFrameOptions,
			expected: "SAMEORIGIN",
		},
		{
			name:     "X-Content-Type-Options",
			header:   HeaderXContentTypeOptions,
			expected: "nosniff",
		},
		{
			name:     "X-XSS-Protection",
			header:   HeaderXXSSProtection,
			expected: "1",
		},
		{
			name:     "Content-Security-Policy",
			header:   HeaderContentSecurityPolicy,
			expected: "default-src 'self'; script-src 'self' 'unsafe-inline'",
		},
		{
			name:     "Referrer-Policy",
			header:   HeaderReferrerPolicy,
			expected: "no-referrer",
		},
		{
			name:     "Strict-Transport-Security",
			header:   HeaderStrictTransportSecurity,
			expected: "max-age=63072000; includeSubDomains; preload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rec.Header().Get(tt.header)
			if got != tt.expected {
				t.Errorf("header %s = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}
}

func TestSecurityHeadersHSTSOverHTTP(t *testing.T) {
	config := SecurityHeadersConfig{
		EnableHSTS:              true,
		StrictTransportSecurity: "max-age=31536000",
	}

	middleware := SecurityHeadersWithConfig(config)
	handler := middleware(func(c transport.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Test with HTTP request (no TLS)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := newMockContext(req, rec)

	handler(ctx)

	// HSTS should not be set over HTTP
	if got := rec.Header().Get(HeaderStrictTransportSecurity); got != "" {
		t.Errorf("HSTS should not be set over HTTP, got %q", got)
	}
}

func TestSecurityHeadersHSTSWithProxy(t *testing.T) {
	config := SecurityHeadersConfig{
		EnableHSTS:              true,
		StrictTransportSecurity: "max-age=31536000",
	}

	middleware := SecurityHeadersWithConfig(config)
	handler := middleware(func(c transport.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Test with X-Forwarded-Proto header (reverse proxy scenario)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	ctx := newMockContext(req, rec)

	handler(ctx)

	// HSTS should be set when X-Forwarded-Proto is https
	got := rec.Header().Get(HeaderStrictTransportSecurity)
	expected := "max-age=31536000"
	if got != expected {
		t.Errorf("HSTS = %q, want %q", got, expected)
	}
}

func TestSetXFrameOptions(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "DENY",
			value:    "DENY",
			expected: "DENY",
		},
		{
			name:     "SAMEORIGIN",
			value:    "SAMEORIGIN",
			expected: "SAMEORIGIN",
		},
		{
			name:     "Empty value",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			setXFrameOptions(ctx, tt.value)

			got := rec.Header().Get(HeaderXFrameOptions)
			if got != tt.expected {
				t.Errorf("X-Frame-Options = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSetXContentTypeOptions(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "nosniff",
			value:    "nosniff",
			expected: "nosniff",
		},
		{
			name:     "Empty value",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			setXContentTypeOptions(ctx, tt.value)

			got := rec.Header().Get(HeaderXContentTypeOptions)
			if got != tt.expected {
				t.Errorf("X-Content-Type-Options = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSetXXSSProtection(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "1; mode=block",
			value:    "1; mode=block",
			expected: "1; mode=block",
		},
		{
			name:     "0",
			value:    "0",
			expected: "0",
		},
		{
			name:     "Empty value",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			setXXSSProtection(ctx, tt.value)

			got := rec.Header().Get(HeaderXXSSProtection)
			if got != tt.expected {
				t.Errorf("X-XSS-Protection = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSetContentSecurityPolicy(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "default-src 'self'",
			value:    "default-src 'self'",
			expected: "default-src 'self'",
		},
		{
			name:     "Complex CSP",
			value:    "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
			expected: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		},
		{
			name:     "Empty value",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			setContentSecurityPolicy(ctx, tt.value)

			got := rec.Header().Get(HeaderContentSecurityPolicy)
			if got != tt.expected {
				t.Errorf("Content-Security-Policy = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSetReferrerPolicy(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "strict-origin-when-cross-origin",
			value:    "strict-origin-when-cross-origin",
			expected: "strict-origin-when-cross-origin",
		},
		{
			name:     "no-referrer",
			value:    "no-referrer",
			expected: "no-referrer",
		},
		{
			name:     "Empty value",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			setReferrerPolicy(ctx, tt.value)

			got := rec.Header().Get(HeaderReferrerPolicy)
			if got != tt.expected {
				t.Errorf("Referrer-Policy = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSetHSTS(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		isHTTPS  bool
		expected string
	}{
		{
			name:     "HTTPS with HSTS",
			value:    "max-age=31536000",
			isHTTPS:  true,
			expected: "max-age=31536000",
		},
		{
			name:     "HTTP with HSTS (should not set)",
			value:    "max-age=31536000",
			isHTTPS:  false,
			expected: "",
		},
		{
			name:     "HTTPS with complex HSTS",
			value:    "max-age=63072000; includeSubDomains; preload",
			isHTTPS:  true,
			expected: "max-age=63072000; includeSubDomains; preload",
		},
		{
			name:     "Empty value",
			value:    "",
			isHTTPS:  true,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			setHSTS(ctx, tt.value, tt.isHTTPS)

			got := rec.Header().Get(HeaderStrictTransportSecurity)
			if got != tt.expected {
				t.Errorf("Strict-Transport-Security = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsHTTPSConnection(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func(*http.Request)
		expected bool
	}{
		{
			name: "HTTPS via TLS",
			setupReq: func(req *http.Request) {
				req.TLS = &tls.ConnectionState{}
			},
			expected: true,
		},
		{
			name: "HTTPS via X-Forwarded-Proto",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
			},
			expected: true,
		},
		{
			name: "HTTPS via X-Forwarded-Proto (uppercase)",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "HTTPS")
			},
			expected: true,
		},
		{
			name: "HTTP",
			setupReq: func(req *http.Request) {
				// No TLS, no X-Forwarded-Proto
			},
			expected: false,
		},
		{
			name: "HTTP via X-Forwarded-Proto",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "http")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupReq(req)
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			got := isHTTPSConnection(ctx)
			if got != tt.expected {
				t.Errorf("isHTTPSConnection() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSecurityHeadersDefaultConfig(t *testing.T) {
	// Test that default config values are correct
	if DefaultSecurityHeadersConfig.XFrameOptions != "DENY" {
		t.Errorf("DefaultSecurityHeadersConfig.XFrameOptions = %q, want %q",
			DefaultSecurityHeadersConfig.XFrameOptions, "DENY")
	}
	if DefaultSecurityHeadersConfig.XContentTypeOptions != "nosniff" {
		t.Errorf("DefaultSecurityHeadersConfig.XContentTypeOptions = %q, want %q",
			DefaultSecurityHeadersConfig.XContentTypeOptions, "nosniff")
	}
	if DefaultSecurityHeadersConfig.XXSSProtection != "1; mode=block" {
		t.Errorf("DefaultSecurityHeadersConfig.XXSSProtection = %q, want %q",
			DefaultSecurityHeadersConfig.XXSSProtection, "1; mode=block")
	}
	if DefaultSecurityHeadersConfig.ContentSecurityPolicy != "default-src 'self'" {
		t.Errorf("DefaultSecurityHeadersConfig.ContentSecurityPolicy = %q, want %q",
			DefaultSecurityHeadersConfig.ContentSecurityPolicy, "default-src 'self'")
	}
	if DefaultSecurityHeadersConfig.ReferrerPolicy != "strict-origin-when-cross-origin" {
		t.Errorf("DefaultSecurityHeadersConfig.ReferrerPolicy = %q, want %q",
			DefaultSecurityHeadersConfig.ReferrerPolicy, "strict-origin-when-cross-origin")
	}
	if DefaultSecurityHeadersConfig.StrictTransportSecurity != "max-age=31536000; includeSubDomains" {
		t.Errorf("DefaultSecurityHeadersConfig.StrictTransportSecurity = %q, want %q",
			DefaultSecurityHeadersConfig.StrictTransportSecurity, "max-age=31536000; includeSubDomains")
	}
	if DefaultSecurityHeadersConfig.EnableHSTS != false {
		t.Errorf("DefaultSecurityHeadersConfig.EnableHSTS = %v, want %v",
			DefaultSecurityHeadersConfig.EnableHSTS, false)
	}
}

func TestSecurityHeadersEmptyConfig(t *testing.T) {
	// Test that empty config uses defaults
	config := SecurityHeadersConfig{}
	middleware := SecurityHeadersWithConfig(config)
	handler := middleware(func(c transport.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := newMockContext(req, rec)

	handler(ctx)

	// Should use default values
	if got := rec.Header().Get(HeaderXFrameOptions); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "DENY")
	}
	if got := rec.Header().Get(HeaderXContentTypeOptions); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}
