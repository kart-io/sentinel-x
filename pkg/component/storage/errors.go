package storage

import (
	"errors"
	"fmt"
)

// Common storage error types.
// These errors can be used directly or wrapped with additional context
// using the WithMessage or WithCause methods.
var (
	// ErrNotConnected indicates that the storage client is not connected
	// to the backend. This typically occurs when:
	// - The client was never initialized
	// - The connection was explicitly closed
	// - The connection was lost and not re-established
	ErrNotConnected = &Error{
		Code:    "NOT_CONNECTED",
		Message: "storage client is not connected",
	}

	// ErrConnectionFailed indicates that an attempt to connect to the
	// storage backend failed. This can occur due to:
	// - Network issues (host unreachable, connection timeout)
	// - Authentication failures (invalid credentials)
	// - Configuration errors (wrong port, invalid TLS settings)
	// - Backend unavailability (service down, maintenance mode)
	ErrConnectionFailed = &Error{
		Code:    "CONNECTION_FAILED",
		Message: "failed to connect to storage backend",
	}

	// ErrTimeout indicates that a storage operation exceeded its deadline.
	// This can happen when:
	// - Network latency is too high
	// - Backend is overloaded and slow to respond
	// - Large data sets cause long processing times
	// - Context deadline was too aggressive
	ErrTimeout = &Error{
		Code:    "TIMEOUT",
		Message: "storage operation timed out",
	}

	// ErrInvalidConfig indicates that the storage configuration is invalid.
	// This is typically detected during validation before connection attempts.
	// Common causes include:
	// - Missing required fields (e.g., address, credentials)
	// - Invalid field values (e.g., negative timeout, invalid port)
	// - Incompatible field combinations
	ErrInvalidConfig = &Error{
		Code:    "INVALID_CONFIG",
		Message: "invalid storage configuration",
	}

	// ErrClientNotFound indicates that a requested client was not found
	// in the storage manager. This occurs when:
	// - Attempting to get a client that was never registered
	// - Using an incorrect client name
	// - The client was unregistered
	ErrClientNotFound = &Error{
		Code:    "CLIENT_NOT_FOUND",
		Message: "storage client not found",
	}

	// ErrClientAlreadyExists indicates that a client with the same name
	// is already registered in the storage manager.
	ErrClientAlreadyExists = &Error{
		Code:    "CLIENT_ALREADY_EXISTS",
		Message: "storage client already exists",
	}

	// ErrOperationFailed indicates that a storage operation failed.
	// This is a generic error that should be wrapped with specific details.
	ErrOperationFailed = &Error{
		Code:    "OPERATION_FAILED",
		Message: "storage operation failed",
	}
)

// Error represents a storage-related error with a code and message.
// It implements the error interface and provides methods for error wrapping and context enrichment.
type Error struct {
	// Code is a machine-readable error code (e.g., "NOT_CONNECTED")
	Code string

	// Message is a human-readable error message
	Message string

	// Cause is the underlying error that caused this error, if any
	Cause error

	// Context contains additional contextual information about the error
	Context map[string]interface{}
}

// Error implements the error interface.
// It returns a formatted error message that includes the code, message,
// and cause (if present).
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error.
// This allows the error to work with errors.Is() and errors.As().
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is checks if this error matches the target error.
// This enables the use of errors.Is() for error comparison.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithMessage creates a new StorageError with an updated message.
// The original error is preserved as the base, and the new message
// provides additional context.
//
// Example usage:
//
//	err := storage.ErrConnectionFailed.WithMessage("failed to connect to Redis at localhost:6379")
func (e *Error) WithMessage(msg string) *Error {
	return &Error{
		Code:    e.Code,
		Message: msg,
		Cause:   e.Cause,
		Context: e.Context,
	}
}

// WithCause creates a new StorageError with an underlying cause.
// This is useful for wrapping lower-level errors with storage-specific
// error types.
//
// Example usage:
//
//	err := storage.ErrConnectionFailed.WithCause(netErr)
func (e *Error) WithCause(cause error) *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Cause:   cause,
		Context: e.Context,
	}
}

// WithContext creates a new StorageError with additional context information.
// The context map can contain any relevant data for debugging or logging.
//
// Example usage:
//
//	err := storage.ErrTimeout.WithContext(map[string]interface{}{
//	    "operation": "GET",
//	    "key": "user:123",
//	    "timeout": "5s",
//	})
func (e *Error) WithContext(ctx map[string]interface{}) *Error {
	newContext := make(map[string]interface{}, len(e.Context)+len(ctx))
	for k, v := range e.Context {
		newContext[k] = v
	}
	for k, v := range ctx {
		newContext[k] = v
	}

	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Cause:   e.Cause,
		Context: newContext,
	}
}

// GetContext retrieves a context value by key.
// Returns the value and true if found, nil and false otherwise.
func (e *Error) GetContext(key string) (interface{}, bool) {
	if e.Context == nil {
		return nil, false
	}
	val, ok := e.Context[key]
	return val, ok
}

// IsError checks if an error is a StorageError.
// This is a convenience function for type assertions.
func IsError(err error) bool {
	var storageErr *Error
	return errors.As(err, &storageErr)
}

// GetError extracts a StorageError from an error chain.
// Returns the StorageError and true if found, nil and false otherwise.
func GetError(err error) (*Error, bool) {
	var storageErr *Error
	if errors.As(err, &storageErr) {
		return storageErr, true
	}
	return nil, false
}
