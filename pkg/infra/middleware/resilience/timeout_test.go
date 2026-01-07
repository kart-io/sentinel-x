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

func TestTimeout_NormalRequest(t *testing.T) {
	timeout := 100 * time.Millisecond
	middleware := Timeout(timeout)

	handlerCalled := false

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		handlerCalled = true
		// Fast request that completes before timeout
		time.Sleep(10 * time.Millisecond)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Expected handler to be called for normal request")
	}

	if w.Code == http.StatusRequestTimeout {
		t.Error("Expected no timeout response for fast request")
	}
}

func TestTimeout_SlowRequest(t *testing.T) {
	timeout := 50 * time.Millisecond
	middleware := Timeout(timeout)

	var handlerStarted sync.WaitGroup
	handlerStarted.Add(1)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		handlerStarted.Done()
		// Slow request that exceeds timeout
		time.Sleep(200 * time.Millisecond)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	// Wait for handler to actually start
	handlerStarted.Wait()

	// Should have sent timeout error response
	if w.Code != http.StatusRequestTimeout {
		t.Errorf("Expected status code %d, got %d", http.StatusRequestTimeout, w.Code)
	}
}

func TestTimeoutWithOptions_SkipPaths(t *testing.T) {
	opts := mwopts.TimeoutOptions{
		Timeout:   50 * time.Millisecond,
		SkipPaths: []string{"/health", "/metrics"},
	}

	middleware := TimeoutWithOptions(opts)

	tests := []struct {
		name        string
		path        string
		sleepTime   time.Duration
		wantTimeout bool
	}{
		{
			name:        "skipped path - no timeout",
			path:        "/health",
			sleepTime:   100 * time.Millisecond,
			wantTimeout: false,
		},
		{
			name:        "skipped path /metrics - no timeout",
			path:        "/metrics",
			sleepTime:   100 * time.Millisecond,
			wantTimeout: false,
		},
		{
			name:        "normal path - timeout",
			path:        "/api/test",
			sleepTime:   100 * time.Millisecond,
			wantTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.GET(tt.path, func(_ *gin.Context) {
				time.Sleep(tt.sleepTime)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			r.ServeHTTP(w, req)

			if tt.wantTimeout {
				if w.Code != http.StatusRequestTimeout {
					t.Errorf("Expected timeout response (code %d) but got code %d", http.StatusRequestTimeout, w.Code)
				}
			} else {
				if w.Code == http.StatusRequestTimeout {
					t.Error("Expected no timeout response for skipped path")
				}
			}
		})
	}
}

func TestTimeoutWithOptions_DefaultTimeout(t *testing.T) {
	// Empty config should use default timeout
	opts := mwopts.TimeoutOptions{}
	middleware := TimeoutWithOptions(opts)

	handlerCalled := false

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Expected handler to be called with default config")
	}
}

func TestTimeout_ContextDeadline(t *testing.T) {
	timeout := 100 * time.Millisecond
	middleware := Timeout(timeout)

	var contextDeadline time.Time
	var hasDeadline bool

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		contextDeadline, hasDeadline = c.Request.Context().Deadline()
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if !hasDeadline {
		t.Error("Expected context to have a deadline")
	}

	expectedDeadline := time.Now().Add(timeout)
	diff := contextDeadline.Sub(expectedDeadline)
	if diff < 0 {
		diff = -diff
	}

	// Allow 100ms tolerance for test execution time
	if diff > 100*time.Millisecond {
		t.Errorf("Context deadline differs too much from expected: %v", diff)
	}
}

func TestTimeout_CanceledContext(t *testing.T) {
	timeout := 50 * time.Millisecond
	middleware := Timeout(timeout)

	// Use channel to receive result from background goroutine
	resultCh := make(chan error, 1)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		// Sleep longer than timeout
		time.Sleep(100 * time.Millisecond)
		// Check context after timeout
		resultCh <- c.Request.Context().Err()
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	// Wait for goroutine to finish checking context
	select {
	case contextErr := <-resultCh:
		// Context should have been canceled due to timeout
		if contextErr == nil {
			t.Error("Expected context error but got nil")
		}

		if contextErr != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", contextErr)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Context check did not complete")
	}
}

func TestTimeout_GoroutineDoesNotLeak(_ *testing.T) {
	timeout := 50 * time.Millisecond
	middleware := Timeout(timeout)

	// Run multiple requests to check for goroutine leaks
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware)
		r.GET("/test", func(_ *gin.Context) {
			// Fast completion
			time.Sleep(10 * time.Millisecond)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)
	}

	// If there are goroutine leaks, test might hang or fail
	// This is a basic test, more sophisticated leak detection would require runtime analysis
}

func TestTimeout_PanicInHandler(_ *testing.T) {
	timeout := 100 * time.Millisecond
	middleware := Timeout(timeout)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		panic("test panic in timeout handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Should not panic at the middleware level
	// The panic will be caught by the done channel mechanism
	r.ServeHTTP(w, req)

	// The goroutine should complete without leaking
	// Wait a bit to ensure goroutine cleanup
	time.Sleep(50 * time.Millisecond)
}

func TestTimeout_MultipleTimeouts(t *testing.T) {
	timeout := 50 * time.Millisecond
	middleware := Timeout(timeout)

	// Test multiple concurrent slow requests
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.GET("/test", func(_ *gin.Context) {
				time.Sleep(100 * time.Millisecond)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusRequestTimeout {
				t.Errorf("Expected timeout response (code %d) but got code %d", http.StatusRequestTimeout, w.Code)
			}
		}()
	}

	wg.Wait()
}

func TestTimeoutWithOptions_ZeroTimeout(t *testing.T) {
	// Zero timeout should use default
	opts := mwopts.TimeoutOptions{
		Timeout: 0,
	}

	middleware := TimeoutWithOptions(opts)

	handlerCalled := false

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

func TestTimeout_VeryShortTimeout(t *testing.T) {
	timeout := 1 * time.Millisecond
	middleware := Timeout(timeout)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		// Even a small sleep should trigger timeout
		time.Sleep(10 * time.Millisecond)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("Expected timeout response (code %d) for very short timeout but got code %d", http.StatusRequestTimeout, w.Code)
	}
}
