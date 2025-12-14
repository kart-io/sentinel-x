package security

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_DefaultConfig(t *testing.T) {
	// Setup
	handler := Headers()(
		transport.HandlerFunc(func(c transport.Context) {
			c.String(http.StatusOK, "OK")
		}),
	)

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
			expected: "", // Default is empty
		},
		{
			name:     "Referrer-Policy",
			header:   HeaderReferrerPolicy,
			expected: "no-referrer",
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

func TestSecurityHeaders_Config(t *testing.T) {
	tests := []struct {
		name     string
		config   HeadersConfig
		reqSetup func(*http.Request)
		checks   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Custom all headers",
			config: HeadersConfig{
				FrameOptionsValue:        "SAMEORIGIN",
				XSSProtectionValue:       "0",
				ContentSecurityPolicy:    "default-src 'self'; script-src 'self' 'unsafe-inline'",
				ReferrerPolicy:           "no-referrer",
				HSTSMaxAge:               63072000,
				HSTSIncludeSubdomains:    true,
				HSTSPreload:              true,
				EnableHSTS:               true,
				EnableFrameOptions:       true,
				EnableContentTypeOptions: true,
				EnableXSSProtection:      true,
			},
			reqSetup: func(req *http.Request) {
				req.TLS = &tls.ConnectionState{} // Simulate HTTPS
			},
			checks: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "SAMEORIGIN", w.Header().Get(HeaderXFrameOptions))
				assert.Equal(t, "nosniff", w.Header().Get(HeaderXContentTypeOptions)) // Fixed expectation for hardcoded value in middleware
				assert.Equal(t, "0", w.Header().Get(HeaderXXSSProtection))
				assert.Equal(t, "default-src 'self'; script-src 'self' 'unsafe-inline'", w.Header().Get(HeaderContentSecurityPolicy))
				assert.Equal(t, "no-referrer", w.Header().Get(HeaderReferrerPolicy))
				assert.Equal(t, "max-age=63072000; includeSubDomains; preload", w.Header().Get(HeaderStrictTransportSecurity))
			},
		},
		{
			name: "HSTS over HTTP should not be set",
			config: HeadersConfig{
				EnableHSTS: true,
				HSTSMaxAge: 31536000,
			},
			reqSetup: func(_ *http.Request) {
				// No TLS, no X-Forwarded-Proto
			},
			checks: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Empty(t, w.Header().Get(HeaderStrictTransportSecurity))
			},
		},
		{
			name: "HSTS with X-Forwarded-Proto",
			config: HeadersConfig{
				EnableHSTS: true,
				HSTSMaxAge: 31536000,
			},
			reqSetup: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
			},
			checks: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "max-age=31536000", w.Header().Get(HeaderStrictTransportSecurity))
			},
		},
		{
			name:   "Empty config uses defaults",
			config: HeadersConfig{},
			reqSetup: func(_ *http.Request) {
				// No special setup
			},
			checks: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "DENY", w.Header().Get(HeaderXFrameOptions))
				assert.Equal(t, "nosniff", w.Header().Get(HeaderXContentTypeOptions))
				assert.Empty(t, w.Header().Get(HeaderStrictTransportSecurity)) // HSTS disabled by default
			},
		},
		{
			name: "Disable HSTS",
			config: func() HeadersConfig {
				c := DefaultHeadersConfig()
				c.EnableHSTS = false
				return c
			}(),
			reqSetup: func(req *http.Request) {
				req.TLS = &tls.ConnectionState{} // Simulate HTTPS
			},
			checks: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Empty(t, w.Header().Get(HeaderStrictTransportSecurity))
			},
		},
		{
			name: "Custom HSTS",
			config: func() HeadersConfig {
				c := DefaultHeadersConfig()
				c.EnableHSTS = true
				c.HSTSMaxAge = 60
				c.HSTSIncludeSubdomains = false
				c.HSTSPreload = true
				return c
			}(),
			reqSetup: func(req *http.Request) {
				req.TLS = &tls.ConnectionState{} // Simulate HTTPS
			},
			checks: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "max-age=60; preload", w.Header().Get(HeaderStrictTransportSecurity))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := HeadersWithConfig(tt.config)
			handler := middleware(func(c transport.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.reqSetup != nil {
				tt.reqSetup(req)
			}
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			handler(ctx)

			tt.checks(t, rec)
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
			setupReq: func(_ *http.Request) {
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
