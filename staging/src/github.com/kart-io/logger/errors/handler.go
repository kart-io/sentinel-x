package errors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/logger/core"
)

// ErrorType represents different types of errors that can occur
type ErrorType int

const (
	ConfigError ErrorType = iota
	EngineError
	OutputError
	OTLPError
	SystemError
)

func (e ErrorType) String() string {
	switch e {
	case ConfigError:
		return "config_error"
	case EngineError:
		return "engine_error"
	case OutputError:
		return "output_error"
	case OTLPError:
		return "otlp_error"
	case SystemError:
		return "system_error"
	default:
		return "unknown_error"
	}
}

// LoggerError represents an error that occurred in the logger system
type LoggerError struct {
	Type      ErrorType
	Message   string
	Cause     error
	Component string
	Timestamp time.Time
}

func (e *LoggerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s [%s]: %s (caused by: %v)", e.Type, e.Component, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s [%s]: %s", e.Type, e.Component, e.Message)
}

func (e *LoggerError) Unwrap() error {
	return e.Cause
}

// NewError creates a new LoggerError
func NewError(errType ErrorType, component, message string, cause error) *LoggerError {
	return &LoggerError{
		Type:      errType,
		Message:   message,
		Cause:     cause,
		Component: component,
		Timestamp: time.Now(),
	}
}

// RetryPolicy defines retry behavior for recoverable errors
type RetryPolicy struct {
	MaxRetries      int
	RetryDelay      time.Duration
	BackoffFactor   float64
	MaxRetryDelay   time.Duration
	RetryableErrors []ErrorType
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		RetryDelay:    100 * time.Millisecond,
		BackoffFactor: 2.0,
		MaxRetryDelay: 5 * time.Second,
		RetryableErrors: []ErrorType{
			OutputError,
			OTLPError,
			SystemError,
		},
	}
}

// IsRetryable checks if an error type is retryable according to the policy
func (p *RetryPolicy) IsRetryable(errType ErrorType) bool {
	for _, retryableType := range p.RetryableErrors {
		if errType == retryableType {
			return true
		}
	}
	return false
}

// ErrorHandler manages error handling and degradation strategies
type ErrorHandler struct {
	retryPolicy    *RetryPolicy
	fallbackLogger core.Logger
	errorCallback  func(*LoggerError)
	mu             sync.RWMutex
	errorCounts    map[string]int
	lastErrors     map[string]*LoggerError
}

// NewErrorHandler creates a new error handler with the given retry policy
func NewErrorHandler(retryPolicy *RetryPolicy) *ErrorHandler {
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}

	return &ErrorHandler{
		retryPolicy:    retryPolicy,
		fallbackLogger: NewNoOpLogger(),
		errorCounts:    make(map[string]int),
		lastErrors:     make(map[string]*LoggerError),
	}
}

// SetFallbackLogger sets the fallback logger to use when all else fails
func (h *ErrorHandler) SetFallbackLogger(logger core.Logger) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fallbackLogger = logger
}

// SetErrorCallback sets a callback function to be called when errors occur
func (h *ErrorHandler) SetErrorCallback(callback func(*LoggerError)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errorCallback = callback
}

// HandleError processes an error and returns whether the operation should be retried
func (h *ErrorHandler) HandleError(err *LoggerError) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Record error statistics
	key := fmt.Sprintf("%s:%s", err.Type, err.Component)
	h.errorCounts[key]++
	h.lastErrors[key] = err

	// Call error callback if set
	if h.errorCallback != nil {
		go h.errorCallback(err)
	}

	// Check if this error type is retryable
	if !h.retryPolicy.IsRetryable(err.Type) {
		return false
	}

	// Check if we've exceeded max retries for this component
	if h.errorCounts[key] > h.retryPolicy.MaxRetries {
		return false
	}

	return true
}

// ExecuteWithRetry executes a function with retry logic
func (h *ErrorHandler) ExecuteWithRetry(ctx context.Context, component string, operation func() error) error {
	var lastErr error
	delay := h.retryPolicy.RetryDelay

	// Reset error count for this component before starting
	h.mu.Lock()
	componentKey := fmt.Sprintf("%s:%s", SystemError, component)
	delete(h.errorCounts, componentKey)
	h.mu.Unlock()

	for attempt := 0; attempt <= h.retryPolicy.MaxRetries; attempt++ {
		// Check context before each attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Apply backoff
			delay = time.Duration(float64(delay) * h.retryPolicy.BackoffFactor)
			if delay > h.retryPolicy.MaxRetryDelay {
				delay = h.retryPolicy.MaxRetryDelay
			}
		}

		err := operation()
		if err == nil {
			// Operation succeeded, reset error count
			h.mu.Lock()
			delete(h.errorCounts, componentKey)
			h.mu.Unlock()
			return nil
		}

		lastErr = err

		// Determine error type
		var loggerErr *LoggerError
		errType := SystemError

		if le, ok := err.(*LoggerError); ok {
			loggerErr = le
			errType = le.Type
		} else {
			loggerErr = NewError(errType, component, "operation failed", err)
		}

		// Check if we should retry
		if !h.HandleError(loggerErr) {
			break
		}

		// Check if we've hit the retry limit
		h.mu.RLock()
		currentCount := h.errorCounts[fmt.Sprintf("%s:%s", errType, component)]
		h.mu.RUnlock()

		if currentCount > h.retryPolicy.MaxRetries {
			break
		}
	}

	return lastErr
}

// GetErrorStats returns error statistics
func (h *ErrorHandler) GetErrorStats() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := make(map[string]int)
	for k, v := range h.errorCounts {
		stats[k] = v
	}
	return stats
}

// GetLastErrors returns the most recent error for each component
func (h *ErrorHandler) GetLastErrors() map[string]*LoggerError {
	h.mu.RLock()
	defer h.mu.RUnlock()

	errors := make(map[string]*LoggerError)
	for k, v := range h.lastErrors {
		errors[k] = v
	}
	return errors
}

// Reset clears all error statistics and cached errors
func (h *ErrorHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.errorCounts = make(map[string]int)
	h.lastErrors = make(map[string]*LoggerError)
}

// GetFallbackLogger returns the fallback logger for emergency use
func (h *ErrorHandler) GetFallbackLogger() core.Logger {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.fallbackLogger
}
