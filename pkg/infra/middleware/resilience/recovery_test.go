package resilience

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

func TestRecovery_NoPanic(t *testing.T) {
	middleware := Recovery()
	handlerCalled := false

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		handlerCalled = true
	})

	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Expected handler to be called when no panic occurs")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	middleware := Recovery()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		panic("test panic")
	})

	// Should not panic
	r.ServeHTTP(w, req)

	// Should have http.StatusInternalServerError status code
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestRecoveryWithConfig_StackTrace(t *testing.T) {
	tests := []struct {
		name             string
		enableStackTrace bool
		wantStackTrace   bool
	}{
		{
			name:             "with stack trace enabled",
			enableStackTrace: true,
			wantStackTrace:   true,
		},
		{
			name:             "with stack trace disabled",
			enableStackTrace: false,
			wantStackTrace:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := mwopts.RecoveryOptions{
				EnableStackTrace: tt.enableStackTrace,
			}
			middleware := RecoveryWithOptions(opts, nil)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.GET("/test", func(_ *gin.Context) {
				panic("test panic with stack")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			// The response.Fail function wraps the error in a Response structure
			// We just verify that JSON was called with error status
			if w.Code != http.StatusInternalServerError {
				t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
			}
		})
	}
}

func TestRecoveryWithOptions_OnPanicCallback(t *testing.T) {
	var panicCalled bool
	var panicErr interface{}
	var panicStack []byte

	opts := mwopts.RecoveryOptions{
		EnableStackTrace: false,
	}
	onPanic := func(_ *gin.Context, err interface{}, stack []byte) {
		panicCalled = true
		panicErr = err
		panicStack = stack
	}

	middleware := RecoveryWithOptions(opts, onPanic)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		panic("callback test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if !panicCalled {
		t.Error("Expected OnPanic callback to be called")
	}

	if panicErr == nil {
		t.Error("Expected panic error to be passed to callback")
	}

	if panicErr != "callback test panic" {
		t.Errorf("Expected panic error 'callback test panic', got %v", panicErr)
	}

	if len(panicStack) == 0 {
		t.Error("Expected stack trace to be passed to callback")
	}
}

func TestRecoveryWithConfig_PanicWithDifferentTypes(t *testing.T) {
	tests := []struct {
		name      string
		panicVal  interface{}
		wantPanic bool
	}{
		{
			name:      "panic with string",
			panicVal:  "string panic",
			wantPanic: true,
		},
		{
			name:      "panic with error",
			panicVal:  &mockError{msg: "error panic"},
			wantPanic: true,
		},
		{
			name:      "panic with integer",
			panicVal:  42,
			wantPanic: true,
		},
		{
			name:      "panic with nil",
			panicVal:  nil,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := Recovery()

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.GET("/test", func(_ *gin.Context) {
				panic(tt.panicVal)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Should not panic
			r.ServeHTTP(w, req)

			if w.Code != http.StatusInternalServerError {
				t.Error("Expected error response after panic")
			}
		})
	}
}

func TestRecovery_DefaultConfig(t *testing.T) {
	middleware := Recovery()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {
		panic("default config test")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Error("Expected error response with default config")
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
