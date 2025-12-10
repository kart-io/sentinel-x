package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// ============================================================================
// Benchmark Test: Logger Middleware
// ============================================================================

// BenchmarkLoggerMiddleware measures the performance of the Logger middleware.
// Tests structured logging with request ID and latency tracking.
func BenchmarkLoggerMiddleware(b *testing.B) {
	middleware := LoggerWithConfig(LoggerConfig{
		SkipPaths:           []string{},
		UseStructuredLogger: true,
	})

	handler := middleware(func(c transport.Context) {
		// Simulate minimal handler work
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkLoggerMiddlewareWithSkip measures performance when path is skipped.
// Tests the skip path optimization logic.
func BenchmarkLoggerMiddlewareWithSkip(b *testing.B) {
	middleware := LoggerWithConfig(LoggerConfig{
		SkipPaths:           []string{"/health"},
		UseStructuredLogger: true,
	})

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// ============================================================================
// Benchmark Test: Recovery Middleware
// ============================================================================

// BenchmarkRecoveryMiddleware measures the performance of Recovery middleware
// in normal operation (no panic).
func BenchmarkRecoveryMiddleware(b *testing.B) {
	middleware := RecoveryWithConfig(DefaultRecoveryConfig)

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkRecoveryMiddlewareWithPanic measures the performance when panic occurs.
// Tests the panic recovery and error response generation.
func BenchmarkRecoveryMiddlewareWithPanic(b *testing.B) {
	middleware := RecoveryWithConfig(RecoveryConfig{
		EnableStackTrace: false,
		OnPanic:          nil,
	})

	handler := middleware(func(c transport.Context) {
		panic("test panic")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// ============================================================================
// Benchmark Test: RequestID Middleware
// ============================================================================

// BenchmarkRequestIDMiddleware measures the performance of RequestID middleware.
// Tests random ID generation and context storage.
func BenchmarkRequestIDMiddleware(b *testing.B) {
	middleware := RequestIDWithConfig(DefaultRequestIDConfig)

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkRequestIDMiddlewareWithExisting measures performance when request ID
// already exists in header.
func BenchmarkRequestIDMiddlewareWithExisting(b *testing.B) {
	middleware := RequestIDWithConfig(DefaultRequestIDConfig)

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "existing-id-12345678")
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkGenerateRequestID measures the performance of request ID generation.
func BenchmarkGenerateRequestID(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = generateRequestID()
	}
}

// ============================================================================
// Benchmark Test: RateLimit Middleware
// ============================================================================

// BenchmarkRateLimitMiddleware measures the performance of RateLimit middleware
// with memory-based limiter.
func BenchmarkRateLimitMiddleware(b *testing.B) {
	limiter := NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()

	middleware := RateLimitWithConfig(RateLimitConfig{
		Limit:   1000,
		Window:  1 * time.Minute,
		Limiter: limiter,
	})

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkRateLimitMiddlewareWithSkip measures performance when path is skipped.
func BenchmarkRateLimitMiddlewareWithSkip(b *testing.B) {
	limiter := NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()

	middleware := RateLimitWithConfig(RateLimitConfig{
		Limit:     1000,
		Window:    1 * time.Minute,
		SkipPaths: []string{"/health"},
		Limiter:   limiter,
	})

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkMemoryRateLimiterAllow measures the performance of rate limiter
// Allow operation.
func BenchmarkMemoryRateLimiterAllow(b *testing.B) {
	limiter := NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(ctx, "test-key")
	}
}

// ============================================================================
// Benchmark Test: SecurityHeaders Middleware
// ============================================================================

// BenchmarkSecurityHeadersMiddleware measures the performance of SecurityHeaders
// middleware.
func BenchmarkSecurityHeadersMiddleware(b *testing.B) {
	middleware := SecurityHeadersWithConfig(DefaultSecurityHeadersConfig)

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkSecurityHeadersMiddlewareWithHSTS measures performance with HSTS
// enabled.
func BenchmarkSecurityHeadersMiddlewareWithHSTS(b *testing.B) {
	middleware := SecurityHeadersWithConfig(SecurityHeadersConfig{
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ContentSecurityPolicy:   "default-src 'self'",
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		EnableHSTS:              true,
	})

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// ============================================================================
// Benchmark Test: Timeout Middleware
// ============================================================================

// BenchmarkTimeoutMiddleware measures the performance of Timeout middleware
// in normal operation.
func BenchmarkTimeoutMiddleware(b *testing.B) {
	middleware := TimeoutWithConfig(TimeoutConfig{
		Timeout: 30 * time.Second,
	})

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkTimeoutMiddlewareWithSkip measures performance when path is skipped.
func BenchmarkTimeoutMiddlewareWithSkip(b *testing.B) {
	middleware := TimeoutWithConfig(TimeoutConfig{
		Timeout:   30 * time.Second,
		SkipPaths: []string{"/health"},
	})

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkTimeoutMiddlewareWithDelay measures performance with simulated
// handler delay.
func BenchmarkTimeoutMiddlewareWithDelay(b *testing.B) {
	middleware := TimeoutWithConfig(TimeoutConfig{
		Timeout: 30 * time.Second,
	})

	handler := middleware(func(c transport.Context) {
		// Simulate 1ms processing time
		time.Sleep(1 * time.Millisecond)
		c.JSON(200, map[string]string{"status": "ok"})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// ============================================================================
// Benchmark Test: Middleware Chain
// ============================================================================

// BenchmarkMiddlewareChain measures the performance of a complete middleware
// chain with all middlewares combined.
func BenchmarkMiddlewareChain(b *testing.B) {
	// Create all middlewares
	requestIDMiddleware := RequestID()
	loggerMiddleware := Logger()
	recoveryMiddleware := Recovery()
	securityMiddleware := SecurityHeaders()
	timeoutMiddleware := Timeout(30 * time.Second)

	limiter := NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()
	rateLimitMiddleware := RateLimitWithConfig(RateLimitConfig{
		Limit:   1000,
		Window:  1 * time.Minute,
		Limiter: limiter,
	})

	// Build handler with complete middleware chain
	handler := func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	}

	// Chain middlewares (in reverse order)
	handler = timeoutMiddleware(handler)
	handler = rateLimitMiddleware(handler)
	handler = securityMiddleware(handler)
	handler = recoveryMiddleware(handler)
	handler = loggerMiddleware(handler)
	handler = requestIDMiddleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkMiddlewareChainMinimal measures the performance of a minimal
// middleware chain (RequestID + Logger + Recovery only).
func BenchmarkMiddlewareChainMinimal(b *testing.B) {
	requestIDMiddleware := RequestID()
	loggerMiddleware := Logger()
	recoveryMiddleware := Recovery()

	handler := func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	}

	handler = recoveryMiddleware(handler)
	handler = loggerMiddleware(handler)
	handler = requestIDMiddleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// BenchmarkMiddlewareChainProduction measures the performance of a production
// middleware chain with optimized settings.
func BenchmarkMiddlewareChainProduction(b *testing.B) {
	// Production-optimized configuration
	requestIDMiddleware := RequestID()
	loggerMiddleware := LoggerWithConfig(LoggerConfig{
		SkipPaths:           []string{"/health", "/metrics"},
		UseStructuredLogger: true,
	})
	recoveryMiddleware := RecoveryWithConfig(RecoveryConfig{
		EnableStackTrace: false, // Disabled in production
	})
	securityMiddleware := SecurityHeadersWithConfig(SecurityHeadersConfig{
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ContentSecurityPolicy:   "default-src 'self'",
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		EnableHSTS:              true,
	})

	limiter := NewMemoryRateLimiter(100, 1*time.Minute)
	defer limiter.Stop()
	rateLimitMiddleware := RateLimitWithConfig(RateLimitConfig{
		Limit:     100,
		Window:    1 * time.Minute,
		SkipPaths: []string{"/health"},
		Limiter:   limiter,
	})

	handler := func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	}

	handler = rateLimitMiddleware(handler)
	handler = securityMiddleware(handler)
	handler = recoveryMiddleware(handler)
	handler = loggerMiddleware(handler)
	handler = requestIDMiddleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}

// ============================================================================
// Benchmark Test: Concurrent Access
// ============================================================================

// BenchmarkMiddlewareChainConcurrent measures the performance of middleware
// chain under concurrent load.
func BenchmarkMiddlewareChainConcurrent(b *testing.B) {
	requestIDMiddleware := RequestID()
	loggerMiddleware := Logger()
	recoveryMiddleware := Recovery()

	limiter := NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()
	rateLimitMiddleware := RateLimitWithConfig(RateLimitConfig{
		Limit:   1000,
		Window:  1 * time.Minute,
		Limiter: limiter,
	})

	handler := func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	}

	handler = rateLimitMiddleware(handler)
	handler = recoveryMiddleware(handler)
	handler = loggerMiddleware(handler)
	handler = requestIDMiddleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()
			ctx := newMockContext(req, w)
			handler(ctx)
		}
	})
}

// ============================================================================
// Benchmark Test: Memory Allocation
// ============================================================================

// BenchmarkMiddlewareMemoryAllocation measures memory allocation patterns
// for different middleware configurations.
func BenchmarkMiddlewareMemoryAllocation(b *testing.B) {
	tests := []struct {
		name       string
		middleware transport.MiddlewareFunc
	}{
		{
			name:       "RequestID",
			middleware: RequestID(),
		},
		{
			name:       "Logger",
			middleware: Logger(),
		},
		{
			name:       "Recovery",
			middleware: Recovery(),
		},
		{
			name:       "SecurityHeaders",
			middleware: SecurityHeaders(),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			handler := tt.middleware(func(c transport.Context) {
				c.JSON(200, map[string]string{"status": "ok"})
			})

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				ctx := newMockContext(req, w)
				handler(ctx)
			}
		})
	}
}

// ============================================================================
// Benchmark Test: Request Processing with Body
// ============================================================================

// BenchmarkMiddlewareChainWithBody measures the performance when handling
// requests with JSON body.
func BenchmarkMiddlewareChainWithBody(b *testing.B) {
	requestIDMiddleware := RequestID()
	loggerMiddleware := Logger()
	recoveryMiddleware := Recovery()

	handler := func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	}

	handler = recoveryMiddleware(handler)
	handler = loggerMiddleware(handler)
	handler = requestIDMiddleware(handler)

	// Prepare request body
	body := []byte(`{"username":"test","password":"secret"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)
		handler(ctx)
	}
}
