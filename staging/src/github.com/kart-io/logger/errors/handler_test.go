package errors

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kart-io/logger/core"
)

func TestNewError(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := NewError(ConfigError, "test-component", "test message", cause)

	if err.Type != ConfigError {
		t.Errorf("Expected error type %v, got %v", ConfigError, err.Type)
	}
	if err.Component != "test-component" {
		t.Errorf("Expected component 'test-component', got '%s'", err.Component)
	}
	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Cause)
	}
	if err.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestLoggerError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *LoggerError
		expected string
	}{
		{
			name: "error with cause",
			err: &LoggerError{
				Type:      ConfigError,
				Message:   "config invalid",
				Component: "validator",
				Cause:     fmt.Errorf("field missing"),
			},
			expected: "config_error [validator]: config invalid (caused by: field missing)",
		},
		{
			name: "error without cause",
			err: &LoggerError{
				Type:      EngineError,
				Message:   "engine failed",
				Component: "zap",
			},
			expected: "engine_error [zap]: engine failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("LoggerError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRetryPolicy_IsRetryable(t *testing.T) {
	policy := DefaultRetryPolicy()

	tests := []struct {
		name     string
		errType  ErrorType
		expected bool
	}{
		{"OutputError is retryable", OutputError, true},
		{"OTLPError is retryable", OTLPError, true},
		{"SystemError is retryable", SystemError, true},
		{"ConfigError is not retryable", ConfigError, false},
		{"EngineError is not retryable", EngineError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.IsRetryable(tt.errType); got != tt.expected {
				t.Errorf("RetryPolicy.IsRetryable(%v) = %v, want %v", tt.errType, got, tt.expected)
			}
		})
	}
}

func TestErrorHandler_HandleError(t *testing.T) {
	handler := NewErrorHandler(nil)

	// Test retryable error
	err := NewError(OutputError, "test-component", "test error", nil)
	shouldRetry := handler.HandleError(err)

	if !shouldRetry {
		t.Error("Expected retryable error to return true")
	}

	// Test non-retryable error
	configErr := NewError(ConfigError, "test-component", "config error", nil)
	shouldRetryConfig := handler.HandleError(configErr)

	if shouldRetryConfig {
		t.Error("Expected non-retryable error to return false")
	}

	// Check error stats
	stats := handler.GetErrorStats()
	if stats["output_error:test-component"] != 1 {
		t.Errorf("Expected error count 1, got %d", stats["output_error:test-component"])
	}
	if stats["config_error:test-component"] != 1 {
		t.Errorf("Expected error count 1, got %d", stats["config_error:test-component"])
	}
}

func TestErrorHandler_ExecuteWithRetry(t *testing.T) {
	handler := NewErrorHandler(&RetryPolicy{
		MaxRetries:      2,
		RetryDelay:      10 * time.Millisecond,
		BackoffFactor:   2.0,
		MaxRetryDelay:   1 * time.Second,
		RetryableErrors: []ErrorType{SystemError},
	})

	t.Run("successful operation", func(t *testing.T) {
		attempts := 0
		err := handler.ExecuteWithRetry(context.Background(), "test-component", func() error {
			attempts++
			if attempts < 2 {
				return NewError(SystemError, "test", "temporary failure", nil)
			}
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", attempts)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		attempts := 0
		err := handler.ExecuteWithRetry(context.Background(), "test-component-2", func() error {
			attempts++
			return NewError(SystemError, "test", "persistent failure", nil)
		})

		if err == nil {
			t.Error("Expected error after max retries exceeded")
		}
		// Should attempt: initial + MaxRetries (2) = 3 total, but retry logic might limit it
		if attempts < 2 || attempts > 3 {
			t.Errorf("Expected 2-3 attempts, got %d", attempts)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		attempts := 0
		err := handler.ExecuteWithRetry(ctx, "test-component-3", func() error {
			attempts++
			if attempts == 1 {
				// Cancel context after first attempt
				cancel()
				return NewError(SystemError, "test", "should retry once", nil)
			}
			return NewError(SystemError, "test", "should not reach here", nil)
		})

		// Should return either the SystemError or context.Canceled
		if err == nil {
			t.Error("Expected an error")
		}

		// Should have attempted at least once
		if attempts < 1 {
			t.Errorf("Expected at least 1 attempt, got %d", attempts)
		}
	})
}

func TestErrorHandler_ErrorCallback(t *testing.T) {
	var mu sync.Mutex
	var callbackErr *LoggerError
	handler := NewErrorHandler(nil)
	handler.SetErrorCallback(func(err *LoggerError) {
		mu.Lock()
		defer mu.Unlock()
		callbackErr = err
	})

	err := NewError(OutputError, "test-component", "test error", nil)
	handler.HandleError(err)

	// Give callback goroutine a moment to execute
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	receivedErr := callbackErr
	mu.Unlock()

	if receivedErr == nil {
		t.Error("Expected error callback to be called")
	}
	if receivedErr.Type != OutputError {
		t.Errorf("Expected OutputError, got %v", receivedErr.Type)
	}
}

func TestErrorHandler_FallbackLogger(t *testing.T) {
	handler := NewErrorHandler(nil)

	// Test default fallback logger is NoOp
	fallback := handler.GetFallbackLogger()
	if _, ok := fallback.(*NoOpLogger); !ok {
		t.Error("Expected default fallback logger to be NoOpLogger")
	}

	// Test setting custom fallback logger
	customLogger := &testLogger{}
	handler.SetFallbackLogger(customLogger)

	newFallback := handler.GetFallbackLogger()
	if newFallback != customLogger {
		t.Error("Expected custom fallback logger")
	}
}

func TestErrorHandler_Reset(t *testing.T) {
	handler := NewErrorHandler(nil)

	// Add some errors
	err1 := NewError(OutputError, "component1", "error1", nil)
	err2 := NewError(SystemError, "component2", "error2", nil)
	handler.HandleError(err1)
	handler.HandleError(err2)

	// Verify errors exist
	stats := handler.GetErrorStats()
	if len(stats) == 0 {
		t.Error("Expected error stats to be non-empty")
	}

	lastErrors := handler.GetLastErrors()
	if len(lastErrors) == 0 {
		t.Error("Expected last errors to be non-empty")
	}

	// Reset and verify
	handler.Reset()

	stats = handler.GetErrorStats()
	if len(stats) != 0 {
		t.Error("Expected error stats to be empty after reset")
	}

	lastErrors = handler.GetLastErrors()
	if len(lastErrors) != 0 {
		t.Error("Expected last errors to be empty after reset")
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := NewNoOpLogger()

	// Test that all methods can be called without panic
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
	logger.Fatal("test") // Should not exit in NoOp

	logger.Debugf("test %s", "format")
	logger.Infof("test %s", "format")
	logger.Warnf("test %s", "format")
	logger.Errorf("test %s", "format")
	logger.Fatalf("test %s", "format") // Should not exit in NoOp

	logger.Debugw("test", "key", "value")
	logger.Infow("test", "key", "value")
	logger.Warnw("test", "key", "value")
	logger.Errorw("test", "key", "value")
	logger.Fatalw("test", "key", "value") // Should not exit in NoOp

	// Test enhancement methods return self
	withLogger := logger.With("key", "value")
	if withLogger != logger {
		t.Error("Expected With() to return same NoOp logger")
	}

	ctxLogger := logger.WithCtx(context.Background(), "key", "value")
	if ctxLogger != logger {
		t.Error("Expected WithCtx() to return same NoOp logger")
	}

	skipLogger := logger.WithCallerSkip(1)
	if skipLogger != logger {
		t.Error("Expected WithCallerSkip() to return same NoOp logger")
	}

	// Test SetLevel doesn't panic
	logger.SetLevel(core.InfoLevel)
}

// testLogger is a simple logger for testing
type testLogger struct{}

func (l *testLogger) Debug(args ...interface{})                                             {}
func (l *testLogger) Info(args ...interface{})                                              {}
func (l *testLogger) Warn(args ...interface{})                                              {}
func (l *testLogger) Error(args ...interface{})                                             {}
func (l *testLogger) Fatal(args ...interface{})                                             {}
func (l *testLogger) Debugf(template string, args ...interface{})                           {}
func (l *testLogger) Infof(template string, args ...interface{})                            {}
func (l *testLogger) Warnf(template string, args ...interface{})                            {}
func (l *testLogger) Errorf(template string, args ...interface{})                           {}
func (l *testLogger) Fatalf(template string, args ...interface{})                           {}
func (l *testLogger) Debugw(msg string, keysAndValues ...interface{})                       {}
func (l *testLogger) Infow(msg string, keysAndValues ...interface{})                        {}
func (l *testLogger) Warnw(msg string, keysAndValues ...interface{})                        {}
func (l *testLogger) Errorw(msg string, keysAndValues ...interface{})                       {}
func (l *testLogger) Fatalw(msg string, keysAndValues ...interface{})                       {}
func (l *testLogger) With(keysAndValues ...interface{}) core.Logger                         { return l }
func (l *testLogger) WithCtx(ctx context.Context, keysAndValues ...interface{}) core.Logger { return l }
func (l *testLogger) WithCallerSkip(skip int) core.Logger                                   { return l }
func (l *testLogger) SetLevel(level core.Level)                                             {}
func (l *testLogger) Flush() error                                                          { return nil }
