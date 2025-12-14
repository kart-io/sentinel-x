package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

func TestCORSConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  CORSConfig
		wantErr bool
	}{
		{
			name: "valid config with specific origins",
			config: CORSConfig{
				AllowOrigins: []string{"https://example.com"},
			},
			wantErr: false,
		},
		{
			name: "valid config with port",
			config: CORSConfig{
				AllowOrigins: []string{"https://example.com:8080"},
			},
			wantErr: false,
		},
		{
			name: "valid config with localhost",
			config: CORSConfig{
				AllowOrigins: []string{"http://localhost:3000"},
			},
			wantErr: false,
		},
		{
			name: "empty origins should fail",
			config: CORSConfig{
				AllowOrigins: []string{},
			},
			wantErr: true,
		},
		{
			name: "wildcard with credentials should fail",
			config: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
			},
			wantErr: true,
		},
		{
			name: "wildcard without credentials is ok",
			config: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: false,
			},
			wantErr: false,
		},
		{
			name: "multiple origins are valid",
			config: CORSConfig{
				AllowOrigins: []string{"https://example.com", "https://api.example.com"},
			},
			wantErr: false,
		},
		{
			name: "origin without scheme should fail",
			config: CORSConfig{
				AllowOrigins: []string{"example.com"},
			},
			wantErr: true,
		},
		{
			name: "origin with path should fail",
			config: CORSConfig{
				AllowOrigins: []string{"https://example.com/api"},
			},
			wantErr: true,
		},
		{
			name: "origin with query should fail",
			config: CORSConfig{
				AllowOrigins: []string{"https://example.com?param=value"},
			},
			wantErr: true,
		},
		{
			name: "empty origin string should fail",
			config: CORSConfig{
				AllowOrigins: []string{""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCORSWithConfig_PreflightRequest(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	middleware := CORSWithConfig(config)
	handlerCalled := false
	handler := middleware(func(_ transport.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	mockCtx := newMockContext(req, w)
	handler(mockCtx)

	// Preflight should not call the next handler
	if handlerCalled {
		t.Error("Expected handler not to be called for preflight request")
	}

	// Check CORS headers
	if got := mockCtx.headers["Access-Control-Allow-Origin"]; got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, "https://example.com")
	}

	if got := mockCtx.headers["Access-Control-Allow-Credentials"]; got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %v, want %v", got, "true")
	}

	if got := mockCtx.headers["Access-Control-Allow-Methods"]; got == "" {
		t.Error("Access-Control-Allow-Methods header not set")
	}

	if got := mockCtx.headers["Access-Control-Allow-Headers"]; got == "" {
		t.Error("Access-Control-Allow-Headers header not set")
	}

	if got := mockCtx.headers["Access-Control-Max-Age"]; got != "3600" {
		t.Errorf("Access-Control-Max-Age = %v, want %v", got, "3600")
	}
}

func TestCORSWithConfig_NormalRequest(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:  []string{"https://example.com"},
		ExposeHeaders: []string{"X-Custom-Header"},
	}

	middleware := CORSWithConfig(config)
	handlerCalled := false
	handler := middleware(func(_ transport.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	mockCtx := newMockContext(req, w)
	handler(mockCtx)

	// Normal request should call the next handler
	if !handlerCalled {
		t.Error("Expected handler to be called for normal request")
	}

	// Check CORS headers
	if got := mockCtx.headers["Access-Control-Allow-Origin"]; got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, "https://example.com")
	}

	if got := mockCtx.headers["Access-Control-Expose-Headers"]; got != "X-Custom-Header" {
		t.Errorf("Access-Control-Expose-Headers = %v, want %v", got, "X-Custom-Header")
	}
}

func TestCORSWithConfig_DisallowedOrigin(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	}

	middleware := CORSWithConfig(config)
	handlerCalled := false
	handler := middleware(func(_ transport.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	mockCtx := newMockContext(req, w)
	handler(mockCtx)

	// Handler should still be called but no CORS headers
	if !handlerCalled {
		t.Error("Expected handler to be called even for disallowed origin")
	}

	// CORS headers should not be set
	if got := mockCtx.headers["Access-Control-Allow-Origin"]; got != "" {
		t.Errorf("Access-Control-Allow-Origin should not be set, got %v", got)
	}
}

func TestCORSWithConfig_WildcardOrigin(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"*"},
	}

	middleware := CORSWithConfig(config)
	handler := middleware(func(_ transport.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://any-domain.com")
	w := httptest.NewRecorder()

	mockCtx := newMockContext(req, w)
	handler(mockCtx)

	// Wildcard should allow any origin
	if got := mockCtx.headers["Access-Control-Allow-Origin"]; got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, "*")
	}
}

func TestCORSWithConfig_NoOriginHeader(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	}

	middleware := CORSWithConfig(config)
	handlerCalled := false
	handler := middleware(func(_ transport.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Origin header set
	w := httptest.NewRecorder()

	mockCtx := newMockContext(req, w)
	handler(mockCtx)

	// Handler should be called
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	// No CORS headers should be set
	if got := mockCtx.headers["Access-Control-Allow-Origin"]; got != "" {
		t.Errorf("Access-Control-Allow-Origin should not be set, got %v", got)
	}
}

func TestCORS_DefaultConfig(t *testing.T) {
	// Default config now has localhost defaults which should not panic
	middleware := CORS()
	if middleware == nil {
		t.Error("Expected CORS() to return a valid middleware")
	}

	// Test that default config works
	handlerCalled := false
	handler := middleware(func(_ transport.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	mockCtx := newMockContext(req, w)
	handler(mockCtx)

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	// Check that CORS headers are set for allowed origin
	if got := mockCtx.headers["Access-Control-Allow-Origin"]; got != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, "http://localhost:3000")
	}
}

func TestCORSWithConfig_Panic(t *testing.T) {
	// Invalid config should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected CORSWithConfig to panic with invalid config")
		}
	}()

	_ = CORSWithConfig(CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})
}
