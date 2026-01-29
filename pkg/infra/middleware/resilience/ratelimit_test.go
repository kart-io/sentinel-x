package resilience

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// 测试常量
const (
	testRemoteAddr = "192.168.1.1:12345"
)

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
		setupReq   func() *http.Request
		opts       mwopts.RateLimitOptions
		expectedIP string
	}{
		{
			name: "extracts from RemoteAddr when proxy headers not trusted",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
				req.Header.Set("X-Real-IP", "203.0.113.2")
				req.RemoteAddr = testRemoteAddr
				return req
			},
			opts: mwopts.RateLimitOptions{
				TrustProxyHeaders: false,
				TrustedProxies:    []string{},
			},
			expectedIP: "192.168.1.1",
		},
		{
			name: "extracts from X-Forwarded-For when request from trusted proxy",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
				req.RemoteAddr = "127.0.0.1:12345"
				return req
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
			req := tt.setupReq()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			ip := extractClientIP(c, tt.opts)
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

// testRateLimitMiddleware 是测试限流中间件的通用辅助函数
func testRateLimitMiddleware(t *testing.T, allowed bool, expectedStatus int) {
	t.Helper()

	limiter := newMockRateLimiter()
	limiter.allowFunc = func(_ context.Context, _ string) (bool, error) {
		return allowed, nil
	}

	opts := mwopts.RateLimitOptions{
		Limit:  10,
		Window: 60,
	}

	middleware := RateLimitWithOptions(opts, limiter)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = testRemoteAddr
	w := httptest.NewRecorder()

	// 使用 Gin 测试上下文
	c, r := gin.CreateTestContext(w)
	c.Request = req
	r.Use(middleware)
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	r.ServeHTTP(w, req)

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("request within limit", func(t *testing.T) {
		testRateLimitMiddleware(t, true, http.StatusOK)
	})

	t.Run("request exceeds limit", func(t *testing.T) {
		testRateLimitMiddleware(t, false, http.StatusTooManyRequests)
	})
}

func TestRateLimit(t *testing.T) {
	middleware := RateLimit()

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	// 使用 Gin 测试上下文
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
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
