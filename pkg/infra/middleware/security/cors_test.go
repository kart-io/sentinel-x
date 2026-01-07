package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// TestCORSConfigValidate 测试已被移除，因为验证逻辑已集成到 WithOptions 函数中

func TestCORSWithOptions_PreflightRequest(t *testing.T) {
	opts := mwopts.CORSOptions{
		AllowOrigins:     []string{"https://example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	middleware := CORSWithOptions(opts)
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

func TestCORSWithOptions_NormalRequest(t *testing.T) {
	opts := mwopts.CORSOptions{
		AllowOrigins:  []string{"https://example.com"},
		ExposeHeaders: []string{"X-Custom-Header"},
	}

	middleware := CORSWithOptions(opts)
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

func TestCORSWithOptions_DisallowedOrigin(t *testing.T) {
	opts := mwopts.CORSOptions{
		AllowOrigins: []string{"https://example.com"},
	}

	middleware := CORSWithOptions(opts)
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

func TestCORSWithOptions_WildcardOrigin(t *testing.T) {
	opts := mwopts.CORSOptions{
		AllowOrigins: []string{"*"},
	}

	middleware := CORSWithOptions(opts)
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

func TestCORSWithOptions_NoOriginHeader(t *testing.T) {
	opts := mwopts.CORSOptions{
		AllowOrigins: []string{"https://example.com"},
	}

	middleware := CORSWithOptions(opts)
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

	// Check that CORS headers are set - default config uses "*"
	if got := mockCtx.headers["Access-Control-Allow-Origin"]; got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, "*")
	}
}

func TestCORSWithOptions_Panic(t *testing.T) {
	// Invalid config should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected CORSWithOptions to panic with invalid config")
		}
	}()

	_ = CORSWithOptions(mwopts.CORSOptions{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})
}
