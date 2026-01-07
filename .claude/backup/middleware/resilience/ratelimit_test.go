package resilience

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// ============================================================================
// Mock Context Implementation
// ============================================================================

type mockRateLimitContext struct {
	*mockContext
}

func newMockRateLimitContext(path string) *mockRateLimitContext {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	return &mockRateLimitContext{
		mockContext: newMockContext(req, rec),
	}
}

func (m *mockRateLimitContext) SetTestHeader(key, value string) {
	m.req.Header.Set(key, value)
}

func (m *mockRateLimitContext) SetRemoteAddr(addr string) {
	m.req.RemoteAddr = addr
}

func (m *mockRateLimitContext) GetStatusCode() int {
	return m.jsonCode
}

// ============================================================================
// Mock Rate Limiter Implementation
// ============================================================================

type mockRateLimiter struct {
	allowFunc func(ctx context.Context, key string) (bool, error)
	resetFunc func(ctx context.Context, key string) error
	mu        sync.Mutex
	calls     map[string]int
}

func newMockRateLimiter() *mockRateLimiter {
	return &mockRateLimiter{
		calls: make(map[string]int),
	}
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	m.mu.Lock()
	m.calls[key]++
	m.mu.Unlock()

	if m.allowFunc != nil {
		return m.allowFunc(ctx, key)
	}
	return true, nil
}

func (m *mockRateLimiter) Reset(ctx context.Context, key string) error {
	m.mu.Lock()
	delete(m.calls, key)
	m.mu.Unlock()

	if m.resetFunc != nil {
		return m.resetFunc(ctx, key)
	}
	return nil
}

func (m *mockRateLimiter) GetCallCount(key string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[key]
}

// ============================================================================
// Key Extraction Tests
// ============================================================================

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func() *mockRateLimitContext
		opts       mwopts.RateLimitOptions
		expectedIP string
	}{
		{
			name: "extracts from RemoteAddr when proxy headers not trusted",
			setupCtx: func() *mockRateLimitContext {
				ctx := newMockRateLimitContext("/test")
				ctx.SetTestHeader("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
				ctx.SetTestHeader("X-Real-IP", "203.0.113.2")
				ctx.SetRemoteAddr("192.168.1.1:12345")
				return ctx
			},
			opts: mwopts.RateLimitOptions{
				TrustProxyHeaders: false,
				TrustedProxies:    []string{},
			},
			expectedIP: "192.168.1.1",
		},
		{
			name: "extracts from X-Forwarded-For when request from trusted proxy",
			setupCtx: func() *mockRateLimitContext {
				ctx := newMockRateLimitContext("/test")
				ctx.SetTestHeader("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
				ctx.SetRemoteAddr("127.0.0.1:12345")
				return ctx
			},
			opts: mwopts.RateLimitOptions{
				TrustProxyHeaders: true,
				TrustedProxies:    []string{"127.0.0.1"},
			},
			expectedIP: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			ip := extractClientIP(ctx, tt.opts)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestGetRemoteIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "standard IP:port format",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[2001:db8::1]:8080",
			expectedIP: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			ip := getRemoteIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

// ============================================================================
// Middleware Integration Tests
// ============================================================================

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("request within limit", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(_ context.Context, _ string) (bool, error) {
			return true, nil
		}

		opts := mwopts.RateLimitOptions{
			Limit:  10,
			Window: 60,
		}

		middleware := RateLimitWithOptions(opts, limiter)
		handler := middleware(func(c transport.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		ctx := newMockRateLimitContext("/api/test")
		handler(ctx)

		if ctx.GetStatusCode() != http.StatusOK {
			t.Errorf("Expected status 200, got %d", ctx.GetStatusCode())
		}
	})

	t.Run("request exceeds limit", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(_ context.Context, _ string) (bool, error) {
			return false, nil
		}

		opts := mwopts.RateLimitOptions{
			Limit:  10,
			Window: 60,
		}

		middleware := RateLimitWithOptions(opts, limiter)
		handler := middleware(func(c transport.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		ctx := newMockRateLimitContext("/api/test")
		handler(ctx)

		if ctx.GetStatusCode() != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", ctx.GetStatusCode())
		}
	})
}

func TestRateLimit(t *testing.T) {
	middleware := RateLimit()

	handler := middleware(func(c transport.Context) {
		c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	ctx := newMockRateLimitContext("/api/test")
	handler(ctx)

	if ctx.GetStatusCode() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.GetStatusCode())
	}
}

// ============================================================================
// Memory Rate Limiter Tests
// ============================================================================

func TestMemoryRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := NewMemoryRateLimiter(5, 1*time.Second)
		defer limiter.Stop()
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(ctx, "test-key")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}
	})

	t.Run("denies requests exceeding limit", func(t *testing.T) {
		limiter := NewMemoryRateLimiter(3, 1*time.Second)
		defer limiter.Stop()
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			allowed, err := limiter.Allow(ctx, "test-key")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}

		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if allowed {
			t.Error("Request should be denied when limit exceeded")
		}
	})
}
