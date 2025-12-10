package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
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
	m.mockContext.req.Header.Set(key, value)
}

func (m *mockRateLimitContext) SetRemoteAddr(addr string) {
	m.mockContext.req.RemoteAddr = addr
}

func (m *mockRateLimitContext) GetStatusCode() int {
	return m.mockContext.jsonCode
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
// Configuration Tests
// ============================================================================

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   RateLimitConfig
		validate func(t *testing.T, result RateLimitConfig)
	}{
		{
			name:   "empty config gets defaults",
			config: RateLimitConfig{},
			validate: func(t *testing.T, result RateLimitConfig) {
				if result.Limit != DefaultRateLimitConfig.Limit {
					t.Errorf("Expected limit %d, got %d", DefaultRateLimitConfig.Limit, result.Limit)
				}
				if result.Window != DefaultRateLimitConfig.Window {
					t.Errorf("Expected window %v, got %v", DefaultRateLimitConfig.Window, result.Window)
				}
				if result.KeyFunc == nil {
					t.Error("Expected default KeyFunc to be set")
				}
				if result.Limiter == nil {
					t.Error("Expected default Limiter to be set")
				}
			},
		},
		{
			name: "negative limit gets default",
			config: RateLimitConfig{
				Limit: -10,
			},
			validate: func(t *testing.T, result RateLimitConfig) {
				if result.Limit != DefaultRateLimitConfig.Limit {
					t.Errorf("Expected default limit, got %d", result.Limit)
				}
			},
		},
		{
			name: "custom values are preserved",
			config: RateLimitConfig{
				Limit:  50,
				Window: 2 * time.Minute,
			},
			validate: func(t *testing.T, result RateLimitConfig) {
				if result.Limit != 50 {
					t.Errorf("Expected limit 50, got %d", result.Limit)
				}
				if result.Window != 2*time.Minute {
					t.Errorf("Expected window 2m, got %v", result.Window)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateConfig(tt.config)
			tt.validate(t, result)
		})
	}
}

// ============================================================================
// Key Extraction Tests
// ============================================================================

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func() *mockRateLimitContext
		expectedIP string
	}{
		{
			name: "extract from X-Forwarded-For",
			setupCtx: func() *mockRateLimitContext {
				ctx := newMockRateLimitContext("/test")
				ctx.SetTestHeader("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
				ctx.SetRemoteAddr("192.168.1.1:12345")
				return ctx
			},
			expectedIP: "203.0.113.1",
		},
		{
			name: "extract from X-Real-IP",
			setupCtx: func() *mockRateLimitContext {
				ctx := newMockRateLimitContext("/test")
				ctx.SetTestHeader("X-Real-IP", "203.0.113.2")
				ctx.SetRemoteAddr("192.168.1.1:12345")
				return ctx
			},
			expectedIP: "203.0.113.2",
		},
		{
			name: "extract from RemoteAddr",
			setupCtx: func() *mockRateLimitContext {
				ctx := newMockRateLimitContext("/test")
				ctx.SetRemoteAddr("203.0.113.3:54321")
				return ctx
			},
			expectedIP: "203.0.113.3",
		},
		{
			name: "X-Forwarded-For takes precedence",
			setupCtx: func() *mockRateLimitContext {
				ctx := newMockRateLimitContext("/test")
				ctx.SetTestHeader("X-Forwarded-For", "203.0.113.10")
				ctx.SetTestHeader("X-Real-IP", "203.0.113.20")
				ctx.SetRemoteAddr("203.0.113.30:12345")
				return ctx
			},
			expectedIP: "203.0.113.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			ip := extractClientIP(ctx)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestDefaultKeyFunc(t *testing.T) {
	ctx := newMockRateLimitContext("/test")
	ctx.SetRemoteAddr("192.168.1.100:12345")

	key := defaultKeyFunc(ctx)
	if key != "192.168.1.100" {
		t.Errorf("Expected key '192.168.1.100', got '%s'", key)
	}
}

func TestExtractKey(t *testing.T) {
	tests := []struct {
		name        string
		keyFunc     func(c transport.Context) string
		expectedKey string
	}{
		{
			name: "custom key function",
			keyFunc: func(c transport.Context) string {
				return "custom-key"
			},
			expectedKey: "custom-key",
		},
		{
			name: "empty key falls back to IP",
			keyFunc: func(c transport.Context) string {
				return ""
			},
			expectedKey: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newMockRateLimitContext("/test")
			ctx.SetRemoteAddr("192.168.1.1:12345")

			key := extractKey(ctx, tt.keyFunc)
			if key != tt.expectedKey {
				t.Errorf("Expected key '%s', got '%s'", tt.expectedKey, key)
			}
		})
	}
}

// ============================================================================
// Path Skipping Tests
// ============================================================================

func TestBuildSkipPathsMap(t *testing.T) {
	paths := []string{"/health", "/metrics", "/ready"}
	skipMap := buildSkipPathsMap(paths)

	if len(skipMap) != 3 {
		t.Errorf("Expected map size 3, got %d", len(skipMap))
	}

	for _, path := range paths {
		if !skipMap[path] {
			t.Errorf("Path %s not in skip map", path)
		}
	}
}

func TestShouldSkipPath(t *testing.T) {
	skipMap := map[string]bool{
		"/health":  true,
		"/metrics": true,
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/health", true},
		{"/metrics", true},
		{"/api/users", false},
		{"/health/check", false}, // Exact match only
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldSkipPath(tt.path, skipMap)
			if result != tt.expected {
				t.Errorf("Expected %v for path %s, got %v", tt.expected, tt.path, result)
			}
		})
	}
}

// ============================================================================
// Rate Limit Check Tests
// ============================================================================

func TestCheckRateLimit(t *testing.T) {
	ctx := context.Background()

	t.Run("allowed request", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(ctx context.Context, key string) (bool, error) {
			return true, nil
		}

		allowed, err := checkRateLimit(ctx, limiter, "test-key")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Error("Expected request to be allowed")
		}
	})

	t.Run("denied request", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(ctx context.Context, key string) (bool, error) {
			return false, nil
		}

		allowed, err := checkRateLimit(ctx, limiter, "test-key")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if allowed {
			t.Error("Expected request to be denied")
		}
	})

	t.Run("limiter error", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(ctx context.Context, key string) (bool, error) {
			return false, fmt.Errorf("limiter error")
		}

		_, err := checkRateLimit(ctx, limiter, "test-key")
		if err == nil {
			t.Error("Expected error from limiter")
		}
	})
}

// ============================================================================
// Middleware Integration Tests
// ============================================================================

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("request within limit", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(ctx context.Context, key string) (bool, error) {
			return true, nil
		}

		config := RateLimitConfig{
			Limit:   10,
			Window:  1 * time.Minute,
			Limiter: limiter,
		}

		middleware := RateLimitWithConfig(config)
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
		limiter.allowFunc = func(ctx context.Context, key string) (bool, error) {
			return false, nil
		}

		config := RateLimitConfig{
			Limit:   10,
			Window:  1 * time.Minute,
			Limiter: limiter,
		}

		middleware := RateLimitWithConfig(config)
		handler := middleware(func(c transport.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		ctx := newMockRateLimitContext("/api/test")
		handler(ctx)

		if ctx.GetStatusCode() != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", ctx.GetStatusCode())
		}
	})

	t.Run("skip configured paths", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(ctx context.Context, key string) (bool, error) {
			return false, nil // Always deny
		}

		config := RateLimitConfig{
			Limit:     10,
			Window:    1 * time.Minute,
			SkipPaths: []string{"/health", "/metrics"},
			Limiter:   limiter,
		}

		middleware := RateLimitWithConfig(config)
		handler := middleware(func(c transport.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		ctx := newMockRateLimitContext("/health")
		handler(ctx)

		// Should succeed even though limiter denies
		if ctx.GetStatusCode() != http.StatusOK {
			t.Errorf("Expected status 200 for skipped path, got %d", ctx.GetStatusCode())
		}

		// Limiter should not be called for skipped paths
		if limiter.GetCallCount("192.168.1.1") > 0 {
			t.Error("Limiter should not be called for skipped paths")
		}
	})

	t.Run("custom key function", func(t *testing.T) {
		limiter := newMockRateLimiter()

		config := RateLimitConfig{
			Limit:  10,
			Window: 1 * time.Minute,
			KeyFunc: func(c transport.Context) string {
				return "user-123"
			},
			Limiter: limiter,
		}

		middleware := RateLimitWithConfig(config)
		handler := middleware(func(c transport.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		ctx := newMockRateLimitContext("/api/test")
		handler(ctx)

		if limiter.GetCallCount("user-123") != 1 {
			t.Errorf("Expected 1 call with key 'user-123', got %d", limiter.GetCallCount("user-123"))
		}
	})

	t.Run("OnLimitReached callback", func(t *testing.T) {
		callbackCalled := false
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(ctx context.Context, key string) (bool, error) {
			return false, nil
		}

		config := RateLimitConfig{
			Limit:   10,
			Window:  1 * time.Minute,
			Limiter: limiter,
			OnLimitReached: func(c transport.Context) {
				callbackCalled = true
			},
		}

		middleware := RateLimitWithConfig(config)
		handler := middleware(func(c transport.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		ctx := newMockRateLimitContext("/api/test")
		handler(ctx)

		if !callbackCalled {
			t.Error("Expected OnLimitReached callback to be called")
		}
	})

	t.Run("limiter error allows request", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.allowFunc = func(ctx context.Context, key string) (bool, error) {
			return false, fmt.Errorf("redis connection error")
		}

		config := RateLimitConfig{
			Limit:   10,
			Window:  1 * time.Minute,
			Limiter: limiter,
		}

		middleware := RateLimitWithConfig(config)
		handler := middleware(func(c transport.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		ctx := newMockRateLimitContext("/api/test")
		handler(ctx)

		// Should allow request on error
		if ctx.GetStatusCode() != http.StatusOK {
			t.Errorf("Expected status 200 on limiter error, got %d", ctx.GetStatusCode())
		}
	})
}

func TestRateLimit(t *testing.T) {
	// Test default configuration
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

		// First 3 requests should succeed
		for i := 0; i < 3; i++ {
			allowed, err := limiter.Allow(ctx, "test-key")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}

		// 4th request should be denied
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if allowed {
			t.Error("Request should be denied when limit exceeded")
		}
	})

	t.Run("resets after time window", func(t *testing.T) {
		limiter := NewMemoryRateLimiter(2, 100*time.Millisecond)
		defer limiter.Stop()
		ctx := context.Background()

		// Use up the limit
		limiter.Allow(ctx, "test-key")
		limiter.Allow(ctx, "test-key")

		// Should be denied immediately
		allowed, _ := limiter.Allow(ctx, "test-key")
		if allowed {
			t.Error("Request should be denied")
		}

		// Wait for window to expire (with margin)
		time.Sleep(200 * time.Millisecond)

		// Should be allowed again
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Error("Request should be allowed after window reset")
		}
	})

	t.Run("different keys are independent", func(t *testing.T) {
		limiter := NewMemoryRateLimiter(1, 1*time.Second)
		defer limiter.Stop()
		ctx := context.Background()

		// Use up limit for key1
		limiter.Allow(ctx, "key1")
		allowed, _ := limiter.Allow(ctx, "key1")
		if allowed {
			t.Error("key1 should be rate limited")
		}

		// key2 should still be allowed
		allowed, err := limiter.Allow(ctx, "key2")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Error("key2 should be allowed")
		}
	})

	t.Run("reset clears counter", func(t *testing.T) {
		limiter := NewMemoryRateLimiter(2, 1*time.Second)
		defer limiter.Stop()
		ctx := context.Background()

		// Use up the limit
		limiter.Allow(ctx, "test-key")
		limiter.Allow(ctx, "test-key")

		// Should be denied
		allowed, _ := limiter.Allow(ctx, "test-key")
		if allowed {
			t.Error("Request should be denied")
		}

		// Reset the key
		err := limiter.Reset(ctx, "test-key")
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}

		// Should be allowed again
		allowed, err = limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Error("Request should be allowed after reset")
		}
	})
}

func TestMemoryRateLimiterConcurrency(t *testing.T) {
	limiter := NewMemoryRateLimiter(100, 1*time.Second)
	defer limiter.Stop()
	ctx := context.Background()

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// Launch 200 concurrent requests
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := limiter.Allow(ctx, "test-key")
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow exactly 100 requests
	if allowedCount != 100 {
		t.Errorf("Expected 100 allowed requests, got %d", allowedCount)
	}
}

func TestFilterExpiredRequests(t *testing.T) {
	now := time.Now()
	requests := []time.Time{
		now.Add(-5 * time.Second),
		now.Add(-3 * time.Second),
		now.Add(-1 * time.Second),
		now,
	}

	cutoff := now.Add(-2 * time.Second)
	filtered := filterExpiredRequests(requests, cutoff)

	// Should keep last 2 requests
	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered requests, got %d", len(filtered))
	}

	// All filtered requests should be after cutoff
	for _, req := range filtered {
		if req.Before(cutoff) {
			t.Error("Filtered request is before cutoff")
		}
	}
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestHandleRateLimitExceeded(t *testing.T) {
	t.Run("returns rate limit error", func(t *testing.T) {
		ctx := newMockRateLimitContext("/api/test")
		handleRateLimitExceeded(ctx, nil)

		// Check that the error response was set
		if ctx.GetStatusCode() != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", ctx.GetStatusCode())
		}
	})

	t.Run("calls callback", func(t *testing.T) {
		callbackCalled := false
		callback := func(c transport.Context) {
			callbackCalled = true
		}

		ctx := newMockRateLimitContext("/api/test")
		handleRateLimitExceeded(ctx, callback)

		if !callbackCalled {
			t.Error("Expected callback to be called")
		}
	})
}
