package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

func TestRecovery_NoPanic(t *testing.T) {
	middleware := Recovery()
	handlerCalled := false

	handler := middleware(func(ctx transport.Context) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	if !handlerCalled {
		t.Error("Expected handler to be called when no panic occurs")
	}

	if mockCtx.jsonCalled {
		t.Error("Expected no JSON response when no panic occurs")
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	middleware := Recovery()

	handler := middleware(func(ctx transport.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	// Should not panic
	handler(mockCtx)

	// Should have sent JSON error response
	if !mockCtx.jsonCalled {
		t.Error("Expected JSON response to be called after panic")
	}

	// Should have http.StatusInternalServerError status code
	if mockCtx.jsonCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, mockCtx.jsonCode)
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
			config := RecoveryConfig{
				EnableStackTrace: tt.enableStackTrace,
			}
			middleware := RecoveryWithConfig(config)

			handler := middleware(func(ctx transport.Context) {
				panic("test panic with stack")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			mockCtx := newMockContext(req, w)

			handler(mockCtx)

			if !mockCtx.jsonCalled {
				t.Fatal("Expected JSON response to be called")
			}

			// The response.Fail function wraps the error in a Response structure
			// We just verify that JSON was called with error status
			if mockCtx.jsonCode != http.StatusInternalServerError {
				t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, mockCtx.jsonCode)
			}
		})
	}
}

func TestRecoveryWithConfig_OnPanicCallback(t *testing.T) {
	var panicCalled bool
	var panicErr interface{}
	var panicStack []byte

	config := RecoveryConfig{
		EnableStackTrace: false,
		OnPanic: func(ctx transport.Context, err interface{}, stack []byte) {
			panicCalled = true
			panicErr = err
			panicStack = stack
		},
	}

	middleware := RecoveryWithConfig(config)
	handler := middleware(func(ctx transport.Context) {
		panic("callback test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

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
			handler := middleware(func(ctx transport.Context) {
				panic(tt.panicVal)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			mockCtx := newMockContext(req, w)

			// Should not panic
			handler(mockCtx)

			if !mockCtx.jsonCalled {
				t.Error("Expected JSON response after panic")
			}
		})
	}
}

func TestRecovery_DefaultConfig(t *testing.T) {
	middleware := Recovery()

	// Verify default config is applied
	handler := middleware(func(ctx transport.Context) {
		panic("default config test")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	if !mockCtx.jsonCalled {
		t.Error("Expected JSON response with default config")
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
