package resilience

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// TestCircuitBreakerWithOptions_Basic 测试熔断器基本功能。
func TestCircuitBreakerWithOptions_Basic(t *testing.T) {
	opts := mwopts.CircuitBreakerOptions{
		MaxFailures:      3,
		Timeout:          2, // 2 秒
		HalfOpenMaxCalls: 1,
		ErrorThreshold:   500,
		Enabled:          true,
	}

	middleware := CircuitBreakerWithOptions(opts)

	// 创建测试处理器
	successHandler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	errorHandler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	})

	// 测试正常请求
	t.Run("normal requests succeed", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware)
		r.GET("/test", successHandler)

		for i := 0; i < 5; i++ {
			w = httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
		}
	})

	// 测试熔断触发
	t.Run("circuit opens after failures", func(t *testing.T) {
		// 为这个测试创建独立的中间件实例
		middleware2 := CircuitBreakerWithOptions(opts)

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware2)
		r.GET("/test", errorHandler)

		// 发送失败请求直到熔断器打开
		for i := 0; i < opts.MaxFailures; i++ {
			w = httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)
		}

		// 下一个请求应该被熔断器拒绝
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected circuit breaker to reject request with 503, got %d", w.Code)
		}
	})
}

// TestCircuitBreakerWithOptions_SkipPaths 测试跳过路径功能。
func TestCircuitBreakerWithOptions_SkipPaths(t *testing.T) {
	opts := mwopts.CircuitBreakerOptions{
		MaxFailures:      2,
		Timeout:          10,
		HalfOpenMaxCalls: 1,
		SkipPaths:        []string{"/health", "/metrics"},
		ErrorThreshold:   500,
		Enabled:          true,
	}

	middleware := CircuitBreakerWithOptions(opts)

	errorHandler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "error"})
	})

	// 跳过的路径不应触发熔断器
	skipPaths := []string{"/health", "/metrics"}
	for _, path := range skipPaths {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware)
		r.GET(path, errorHandler)

		for i := 0; i < 5; i++ {
			w = httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			r.ServeHTTP(w, req)

			// 即使返回 500，跳过的路径也不应触发熔断器
			if w.Code != http.StatusInternalServerError {
				t.Errorf("path %s should skip circuit breaker, got status %d", path, w.Code)
			}
		}
	}

	// 验证其他路径仍然正常工作
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/api/test", errorHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	r.ServeHTTP(w, req)

	// 第一次请求应该正常处理（即使失败）
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestCircuitBreakerWithOptions_SkipPathPrefixes 测试跳过路径前缀功能。
func TestCircuitBreakerWithOptions_SkipPathPrefixes(t *testing.T) {
	opts := mwopts.CircuitBreakerOptions{
		MaxFailures:      2,
		Timeout:          10,
		HalfOpenMaxCalls: 1,
		SkipPathPrefixes: []string{"/static/", "/public/"},
		ErrorThreshold:   500,
		Enabled:          true,
	}

	middleware := CircuitBreakerWithOptions(opts)

	errorHandler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "error"})
	})

	// 匹配前缀的路径不应触发熔断器
	skipPaths := []string{
		"/static/css/main.css",
		"/static/js/app.js",
		"/public/images/logo.png",
	}

	for _, path := range skipPaths {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware)
		r.GET(path, errorHandler)

		for i := 0; i < 5; i++ {
			w = httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusInternalServerError {
				t.Errorf("path %s should skip circuit breaker, got status %d", path, w.Code)
			}
		}
	}
}

// TestCircuitBreakerWithOptions_ErrorThreshold 测试错误阈值配置。
func TestCircuitBreakerWithOptions_ErrorThreshold(t *testing.T) {
	tests := []struct {
		name           string
		errorThreshold int
		statusCode     int
		shouldTrigger  bool
	}{
		{
			name:           "500 triggers with threshold 500",
			errorThreshold: 500,
			statusCode:     http.StatusInternalServerError,
			shouldTrigger:  true,
		},
		{
			name:           "404 does not trigger with threshold 500",
			errorThreshold: 500,
			statusCode:     http.StatusNotFound,
			shouldTrigger:  false,
		},
		{
			name:           "404 triggers with threshold 400",
			errorThreshold: 400,
			statusCode:     http.StatusNotFound,
			shouldTrigger:  true,
		},
		{
			name:           "200 does not trigger",
			errorThreshold: 500,
			statusCode:     http.StatusOK,
			shouldTrigger:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := mwopts.CircuitBreakerOptions{
				MaxFailures:      2,
				Timeout:          10,
				HalfOpenMaxCalls: 1,
				ErrorThreshold:   tt.errorThreshold,
				Enabled:          true,
			}

			middleware := CircuitBreakerWithOptions(opts)

			handler := gin.HandlerFunc(func(c *gin.Context) {
				c.JSON(tt.statusCode, map[string]string{"status": "test"})
			})

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.GET("/test", handler)

			// 发送多次请求
			for i := 0; i < opts.MaxFailures+1; i++ {
				w = httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				r.ServeHTTP(w, req)
			}

			// 再发送一次，检查是否被熔断
			w = httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			if tt.shouldTrigger {
				if w.Code != http.StatusServiceUnavailable {
					t.Errorf("expected circuit breaker to trigger (503), got %d", w.Code)
				}
			} else {
				if w.Code == http.StatusServiceUnavailable {
					t.Errorf("circuit breaker should not trigger, got 503")
				}
			}
		})
	}
}

// TestCircuitBreakerWithOptions_HalfOpen 测试半开状态。
func TestCircuitBreakerWithOptions_HalfOpen(t *testing.T) {
	opts := mwopts.CircuitBreakerOptions{
		MaxFailures:      2,
		Timeout:          1, // 1 秒超时
		HalfOpenMaxCalls: 1,
		ErrorThreshold:   500,
		Enabled:          true,
	}

	middleware := CircuitBreakerWithOptions(opts)

	successHandler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	errorHandler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "error"})
	})

	// 1. 触发熔断器打开
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", errorHandler)

	for i := 0; i < opts.MaxFailures; i++ {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)
	}

	// 2. 验证熔断器已打开
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected circuit breaker to be open (503), got %d", w.Code)
	}

	// 3. 等待超时，进入半开状态
	time.Sleep(time.Duration(opts.Timeout+1) * time.Second)

	// 4. 切换到成功处理器，验证半开状态允许请求
	w = httptest.NewRecorder()
	_, r = gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", successHandler)

	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected half-open state to allow request (200), got %d", w.Code)
	}

	// 5. 成功后应该恢复到关闭状态
	// 再发送一次请求验证
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected circuit breaker to be closed (200), got %d", w.Code)
	}
}

// BenchmarkCircuitBreaker_Success 测试熔断器成功请求的性能。
func BenchmarkCircuitBreaker_Success(b *testing.B) {
	opts := mwopts.CircuitBreakerOptions{
		MaxFailures:      100,
		Timeout:          60,
		HalfOpenMaxCalls: 1,
		ErrorThreshold:   500,
		Enabled:          true,
	}

	middleware := CircuitBreakerWithOptions(opts)

	handler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(context.Background())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkCircuitBreaker_Open 测试熔断器打开状态的性能。
func BenchmarkCircuitBreaker_Open(b *testing.B) {
	opts := mwopts.CircuitBreakerOptions{
		MaxFailures:      2,
		Timeout:          60,
		HalfOpenMaxCalls: 1,
		ErrorThreshold:   500,
		Enabled:          true,
	}

	middleware := CircuitBreakerWithOptions(opts)

	errorHandler := gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "error"})
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", errorHandler)

	// 触发熔断器打开
	for i := 0; i < opts.MaxFailures; i++ {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(context.Background())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
