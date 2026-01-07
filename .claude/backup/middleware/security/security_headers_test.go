package security

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

func TestSecurityHeaders_DefaultConfig(t *testing.T) {
	// Setup
	opts := mwopts.NewSecurityHeadersOptions()
	handler := SecurityHeadersWithOptions(*opts)(
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

func TestSecurityHeaders_CustomConfig(t *testing.T) {
	opts := mwopts.SecurityHeadersOptions{
		EnableFrameOptions:       true,
		FrameOptionsValue:        "SAMEORIGIN",
		EnableContentTypeOptions: true,
		EnableXSSProtection:      true,
		XSSProtectionValue:       "0",
		ContentSecurityPolicy:    "default-src 'self'; script-src 'self' 'unsafe-inline'",
		ReferrerPolicy:           "no-referrer",
		EnableHSTS:               true,
		HSTSMaxAge:               63072000,
		HSTSIncludeSubdomains:    true,
		HSTSPreload:              true,
	}

	middleware := SecurityHeadersWithOptions(opts)
	handler := middleware(func(c transport.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{} // Simulate HTTPS
	rec := httptest.NewRecorder()
	ctx := newMockContext(req, rec)

	handler(ctx)

	// Check headers from mockContext.headers (SetHeader writes to both places)
	if got := ctx.headers[HeaderXFrameOptions]; got != "SAMEORIGIN" {
		t.Errorf("header %s = %q, want %q", HeaderXFrameOptions, got, "SAMEORIGIN")
	}
	if got := ctx.headers[HeaderXContentTypeOptions]; got != "nosniff" {
		t.Errorf("header %s = %q, want %q", HeaderXContentTypeOptions, got, "nosniff")
	}
	if got := ctx.headers[HeaderXXSSProtection]; got != "0" {
		t.Errorf("header %s = %q, want %q", HeaderXXSSProtection, got, "0")
	}
	if got := ctx.headers[HeaderContentSecurityPolicy]; got != "default-src 'self'; script-src 'self' 'unsafe-inline'" {
		t.Errorf("header %s = %q, want %q", HeaderContentSecurityPolicy, got, "default-src 'self'; script-src 'self' 'unsafe-inline'")
	}
	if got := ctx.headers[HeaderReferrerPolicy]; got != "no-referrer" {
		t.Errorf("header %s = %q, want %q", HeaderReferrerPolicy, got, "no-referrer")
	}
	if got := ctx.headers[HeaderStrictTransportSecurity]; got != "max-age=63072000; includeSubDomains; preload" {
		t.Errorf("header %s = %q, want %q", HeaderStrictTransportSecurity, got, "max-age=63072000; includeSubDomains; preload")
	}
}

func TestSecurityHeaders_HSTS(t *testing.T) {
	tests := []struct {
		name     string
		opts     mwopts.SecurityHeadersOptions
		setupReq func(*http.Request)
		wantHSTS string
	}{
		{
			name: "HSTS over HTTPS",
			opts: mwopts.SecurityHeadersOptions{
				EnableHSTS:            true,
				HSTSMaxAge:            31536000,
				HSTSIncludeSubdomains: true,
				HSTSPreload:           true,
			},
			setupReq: func(req *http.Request) {
				req.TLS = &tls.ConnectionState{}
			},
			wantHSTS: "max-age=31536000; includeSubDomains; preload",
		},
		{
			name: "HSTS over HTTP (should not set)",
			opts: mwopts.SecurityHeadersOptions{
				EnableHSTS: true,
				HSTSMaxAge: 31536000,
			},
			setupReq: func(_ *http.Request) {
				// No TLS, no X-Forwarded-Proto
			},
			wantHSTS: "",
		},
		{
			name: "HSTS with X-Forwarded-Proto",
			opts: mwopts.SecurityHeadersOptions{
				EnableHSTS: true,
				HSTSMaxAge: 31536000,
			},
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
			},
			wantHSTS: "max-age=31536000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := SecurityHeadersWithOptions(tt.opts)
			handler := middleware(func(c transport.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setupReq != nil {
				tt.setupReq(req)
			}
			rec := httptest.NewRecorder()
			ctx := newMockContext(req, rec)

			handler(ctx)

			got := rec.Header().Get(HeaderStrictTransportSecurity)
			if got != tt.wantHSTS {
				t.Errorf("HSTS = %q, want %q", got, tt.wantHSTS)
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
			name: "HTTP",
			setupReq: func(_ *http.Request) {
				// No TLS, no X-Forwarded-Proto
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
