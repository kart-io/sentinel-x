package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/requestutil"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/security"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

const (
	testPath  = "/test"
	statusMsg = "ok"
	testAddr  = "127.0.0.1:12345"
)

// ============================================================================
// Benchmark Test: Logger Middleware
// ============================================================================

// BenchmarkLoggerMiddleware measures the performance of the Logger middleware.
// Tests structured logging with request ID and latency tracking.
func BenchmarkLoggerMiddleware(b *testing.B) {
	opts := mwopts.LoggerOptions{
		SkipPaths:           []string{},
		UseStructuredLogger: true,
	}
	middleware := observability.LoggerWithOptions(opts, nil)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		// Simulate minimal handler work
		c.JSON(200, map[string]string{"status": statusMsg})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkLoggerMiddlewareWithSkip measures performance when path is skipped.
// Tests the skip path optimization logic.
func BenchmarkLoggerMiddlewareWithSkip(b *testing.B) {
	opts := mwopts.LoggerOptions{
		SkipPaths:           []string{"/health"},
		UseStructuredLogger: true,
	}
	middleware := observability.LoggerWithOptions(opts, nil)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/health", handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// ============================================================================
// Benchmark Test: Recovery Middleware
// ============================================================================

// BenchmarkRecoveryMiddleware measures the performance of Recovery middleware
// in normal operation (no panic).
func BenchmarkRecoveryMiddleware(b *testing.B) {
	opts := mwopts.NewRecoveryOptions()
	middleware := resilience.RecoveryWithOptions(*opts, nil)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkRecoveryMiddlewareWithPanic measures the performance when panic occurs.
// Tests the panic recovery and error response generation.
func BenchmarkRecoveryMiddlewareWithPanic(b *testing.B) {
	opts := mwopts.RecoveryOptions{
		EnableStackTrace: false,
	}
	middleware := resilience.RecoveryWithOptions(opts, nil)

	handler := gin.HandlerFunc(func(_ *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// ============================================================================
// Benchmark Test: RequestID Middleware
// ============================================================================

// BenchmarkRequestIDMiddleware measures the performance of RequestID middleware.
// Tests random ID generation and context storage.
func BenchmarkRequestIDMiddleware(b *testing.B) {
	opts := mwopts.NewRequestIDOptions()
	middleware := RequestIDWithOptions(*opts, nil)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkRequestIDMiddlewareWithExisting measures performance when request ID
// already exists in header.
func BenchmarkRequestIDMiddlewareWithExisting(b *testing.B) {
	opts := mwopts.NewRequestIDOptions()
	middleware := RequestIDWithOptions(*opts, nil)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.Header.Set("X-Request-ID", "existing-id-12345678")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkGenerateRequestID measures the performance of request ID generation.
func BenchmarkGenerateRequestID(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = requestutil.GenerateRequestID()
	}
}

// ============================================================================
// Benchmark Test: RateLimit Middleware
// ============================================================================

// BenchmarkRateLimitMiddleware measures the performance of RateLimit middleware
// with memory-based limiter.
func BenchmarkRateLimitMiddleware(b *testing.B) {
	limiter := resilience.NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()

	opts := mwopts.RateLimitOptions{
		Limit:  1000,
		Window: 60, // seconds
	}
	middleware := resilience.RateLimitWithOptions(opts, limiter)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.RemoteAddr = testAddr

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkRateLimitMiddlewareWithSkip measures performance when path is skipped.
func BenchmarkRateLimitMiddlewareWithSkip(b *testing.B) {
	limiter := resilience.NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()

	opts := mwopts.RateLimitOptions{
		Limit:     1000,
		Window:    60,
		SkipPaths: []string{"/health"},
	}
	middleware := resilience.RateLimitWithOptions(opts, limiter)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/health", handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkMemoryRateLimiterAllow measures the performance of rate limiter
// Allow operation.
func BenchmarkMemoryRateLimiterAllow(b *testing.B) {
	limiter := resilience.NewMemoryRateLimiter(1000, 1*time.Minute)
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
func BenchmarkSecurityHeaders(b *testing.B) {
	// Setup
	opts := mwopts.NewSecurityHeadersOptions()
	middleware := security.SecurityHeadersWithOptions(*opts)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkSecurityHeadersMiddlewareWithHSTS measures performance with HSTS
// enabled.
func BenchmarkSecurityHeadersMiddlewareWithHSTS(b *testing.B) {
	opts := mwopts.SecurityHeadersOptions{
		FrameOptionsValue:     "DENY",
		XSSProtectionValue:    "1; mode=block",
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		EnableHSTS:            true,
	}
	middleware := security.SecurityHeadersWithOptions(opts)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// ============================================================================
// Benchmark Test: Timeout Middleware
// ============================================================================

// BenchmarkTimeoutMiddleware measures the performance of Timeout middleware
// in normal operation.
func BenchmarkTimeoutMiddleware(b *testing.B) {
	opts := mwopts.TimeoutOptions{
		Timeout: 30 * time.Second,
	}
	middleware := TimeoutWithOptions(opts)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkTimeoutMiddlewareWithSkip measures performance when path is skipped.
func BenchmarkTimeoutMiddlewareWithSkip(b *testing.B) {
	opts := mwopts.TimeoutOptions{
		Timeout:   30 * time.Second,
		SkipPaths: []string{"/health"},
	}
	middleware := TimeoutWithOptions(opts)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/health", handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkTimeoutMiddlewareWithDelay measures performance with simulated
// handler delay.
func BenchmarkTimeoutMiddlewareWithDelay(b *testing.B) {
	opts := mwopts.TimeoutOptions{
		Timeout: 30 * time.Second,
	}
	middleware := TimeoutWithOptions(opts)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		// Simulate 1ms processing time
		time.Sleep(1 * time.Millisecond)
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
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
	securityMiddleware := security.SecurityHeaders()
	timeoutOpts := mwopts.TimeoutOptions{Timeout: 30 * time.Second}
	timeoutMiddleware := TimeoutWithOptions(timeoutOpts)

	limiter := resilience.NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()
	rateLimitOpts := mwopts.RateLimitOptions{
		Limit:  1000,
		Window: 60,
	}
	rateLimitMiddleware := resilience.RateLimitWithOptions(rateLimitOpts, limiter)

	// Build handler with complete middleware chain
	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(requestIDMiddleware, loggerMiddleware, recoveryMiddleware, securityMiddleware, timeoutMiddleware, rateLimitMiddleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.RemoteAddr = testAddr

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkMiddlewareChainMinimal measures the performance of a minimal
// middleware chain (RequestID + Logger + Recovery only).
func BenchmarkMiddlewareChainMinimal(b *testing.B) {
	requestIDMiddleware := RequestID()
	loggerMiddleware := Logger()
	recoveryMiddleware := Recovery()

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(requestIDMiddleware, loggerMiddleware, recoveryMiddleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkMiddlewareChainProduction measures the performance of a production
// middleware chain with optimized settings.
func BenchmarkMiddlewareChainProduction(b *testing.B) {
	// Production-optimized configuration
	requestIDMiddleware := RequestID()
	loggerOpts := mwopts.LoggerOptions{
		SkipPaths:           []string{"/health", "/metrics"},
		UseStructuredLogger: true,
	}
	loggerMiddleware := observability.LoggerWithOptions(loggerOpts, nil)
	recoveryOpts := mwopts.RecoveryOptions{
		EnableStackTrace: false, // Disabled in production
	}
	recoveryMiddleware := resilience.RecoveryWithOptions(recoveryOpts, nil)
	securityOpts := mwopts.SecurityHeadersOptions{
		FrameOptionsValue:     "DENY",
		XSSProtectionValue:    "1; mode=block",
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		EnableHSTS:            true,
	}
	securityMiddleware := security.SecurityHeadersWithOptions(securityOpts)

	limiter := resilience.NewMemoryRateLimiter(100, 1*time.Minute)
	defer limiter.Stop()
	rateLimitOpts := mwopts.RateLimitOptions{
		Limit:     100,
		Window:    60,
		SkipPaths: []string{"/health"},
	}
	rateLimitMiddleware := resilience.RateLimitWithOptions(rateLimitOpts, limiter)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(requestIDMiddleware, loggerMiddleware, recoveryMiddleware, securityMiddleware, rateLimitMiddleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.RemoteAddr = "127.0.0.1:12345"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
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

	limiter := resilience.NewMemoryRateLimiter(1000, 1*time.Minute)
	defer limiter.Stop()
	rateLimitOpts := mwopts.RateLimitOptions{
		Limit:  1000,
		Window: 60,
	}
	rateLimitMiddleware := resilience.RateLimitWithOptions(rateLimitOpts, limiter)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(requestIDMiddleware, loggerMiddleware, recoveryMiddleware, rateLimitMiddleware)
	r.GET(testPath, handler)

	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.RemoteAddr = "127.0.0.1:12345"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
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
		middleware gin.HandlerFunc
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
			middleware: security.SecurityHeaders(),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			handler := gin.HandlerFunc(func(c *gin.Context) {
				c.JSON(200, map[string]string{"status": "ok"})
			})

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(tt.middleware)
			r.GET(testPath, handler)

			req := httptest.NewRequest(http.MethodGet, testPath, nil)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				w = httptest.NewRecorder()
				r.ServeHTTP(w, req)
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

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(requestIDMiddleware, loggerMiddleware, recoveryMiddleware)
	r.POST("/login", handler)

	// Prepare request body
	body := []byte(`{"username":"test","password":"secret"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
