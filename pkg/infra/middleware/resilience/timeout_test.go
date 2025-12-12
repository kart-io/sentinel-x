package resilience

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

func TestTimeout_NormalRequest(t *testing.T) {
	timeout := 100 * time.Millisecond
	middleware := Timeout(timeout)

	handlerCalled := false
	handler := middleware(func(ctx transport.Context) {
		handlerCalled = true
		// Fast request that completes before timeout
		time.Sleep(10 * time.Millisecond)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	if !handlerCalled {
		t.Error("Expected handler to be called for normal request")
	}

	if mockCtx.jsonCalled {
		t.Error("Expected no timeout response for fast request")
	}
}

func TestTimeout_SlowRequest(t *testing.T) {
	timeout := 50 * time.Millisecond
	middleware := Timeout(timeout)

	var handlerStarted sync.WaitGroup
	handlerStarted.Add(1)

	handler := middleware(func(ctx transport.Context) {
		handlerStarted.Done()
		// Slow request that exceeds timeout
		time.Sleep(200 * time.Millisecond)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Wait for handler to actually start
	handlerStarted.Wait()

	// Should have sent timeout error response
	if !mockCtx.jsonCalled {
		t.Error("Expected timeout error response for slow request")
	}

	if mockCtx.jsonCode != http.StatusRequestTimeout {
		t.Errorf("Expected status code %d, got %d", http.StatusRequestTimeout, mockCtx.jsonCode)
	}
}

func TestTimeoutWithConfig_SkipPaths(t *testing.T) {
	config := TimeoutConfig{
		Timeout:   50 * time.Millisecond,
		SkipPaths: []string{"/health", "/metrics"},
	}

	middleware := TimeoutWithConfig(config)

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
			handler := middleware(func(ctx transport.Context) {
				time.Sleep(tt.sleepTime)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			mockCtx := newMockContext(req, w)

			handler(mockCtx)

			if tt.wantTimeout {
				if !mockCtx.jsonCalled {
					t.Error("Expected timeout response but got none")
				}
			} else {
				if mockCtx.jsonCalled {
					t.Error("Expected no timeout response for skipped path")
				}
			}
		})
	}
}

func TestTimeoutWithConfig_DefaultTimeout(t *testing.T) {
	// Empty config should use default timeout
	config := TimeoutConfig{}
	middleware := TimeoutWithConfig(config)

	handlerCalled := false
	handler := middleware(func(ctx transport.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	if !handlerCalled {
		t.Error("Expected handler to be called with default config")
	}
}

func TestTimeout_ContextDeadline(t *testing.T) {
	timeout := 100 * time.Millisecond
	middleware := Timeout(timeout)

	var contextDeadline time.Time
	var hasDeadline bool

	handler := middleware(func(ctx transport.Context) {
		contextDeadline, hasDeadline = ctx.Request().Deadline()
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

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

	var contextErr error
	var ctxChecked bool

	handler := middleware(func(ctx transport.Context) {
		// Sleep longer than timeout
		time.Sleep(100 * time.Millisecond)
		// Check context after timeout
		contextErr = ctx.Request().Err()
		ctxChecked = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Wait for goroutine to finish checking context
	time.Sleep(150 * time.Millisecond)

	if !ctxChecked {
		t.Error("Context check did not complete")
	}

	// Context should have been canceled due to timeout
	if contextErr == nil {
		t.Error("Expected context error but got nil")
	}

	if contextErr != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", contextErr)
	}
}

func TestTimeout_GoroutineDoesNotLeak(t *testing.T) {
	timeout := 50 * time.Millisecond
	middleware := Timeout(timeout)

	// Run multiple requests to check for goroutine leaks
	for i := 0; i < 10; i++ {
		handler := middleware(func(ctx transport.Context) {
			// Fast completion
			time.Sleep(10 * time.Millisecond)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		mockCtx := newMockContext(req, w)

		handler(mockCtx)
	}

	// If there are goroutine leaks, test might hang or fail
	// This is a basic test, more sophisticated leak detection would require runtime analysis
}

func TestTimeout_PanicInHandler(t *testing.T) {
	timeout := 100 * time.Millisecond
	middleware := Timeout(timeout)

	handler := middleware(func(ctx transport.Context) {
		panic("test panic in timeout handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	// Should not panic at the middleware level
	// The panic will be caught by the done channel mechanism
	handler(mockCtx)

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

			handler := middleware(func(ctx transport.Context) {
				time.Sleep(100 * time.Millisecond)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			mockCtx := newMockContext(req, w)

			handler(mockCtx)

			if !mockCtx.jsonCalled {
				t.Error("Expected timeout response")
			}
		}()
	}

	wg.Wait()
}

func TestTimeoutWithConfig_ZeroTimeout(t *testing.T) {
	// Zero timeout should use default
	config := TimeoutConfig{
		Timeout: 0,
	}

	middleware := TimeoutWithConfig(config)

	handlerCalled := false
	handler := middleware(func(ctx transport.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

func TestTimeout_VeryShortTimeout(t *testing.T) {
	timeout := 1 * time.Millisecond
	middleware := Timeout(timeout)

	handler := middleware(func(ctx transport.Context) {
		// Even a small sleep should trigger timeout
		time.Sleep(10 * time.Millisecond)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	if !mockCtx.jsonCalled {
		t.Error("Expected timeout response for very short timeout")
	}
}
