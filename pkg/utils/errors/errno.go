// Package errors provides a unified error handling system for Sentinel-X.
//
// This package implements a structured error code system following the onex project
// error code design specifications. It provides:
//
//   - Globally unique error codes
//   - Module-based error categorization
//   - Clear business semantics
//   - Multi-language support (EN/ZH)
//   - HTTP and gRPC status code mapping
//
// Error Code Format: AABBCCC (7 digits)
//
//	AA  (00-99): Service/Module code - identifies the source service
//	BB  (00-99): Category code - identifies the error category
//	CCC (000-999): Sequence number - specific error within the category
//
// Service Codes (AA):
//
//	00: Common/Base errors (shared by all services)
//	01: Gateway service
//	02: User service
//	03: Scheduler service
//	04: API service
//	05-09: Reserved for core services
//	10-19: Infrastructure errors (DB, Cache, MQ)
//	20-79: Business service errors
//	80-89: Internal service errors
//	90-99: Third-party service errors
//
// Category Codes (BB):
//
//	00: Success
//	01: Request/Validation errors (400)
//	02: Authentication errors (401)
//	03: Authorization errors (403)
//	04: Resource not found errors (404)
//	05: Conflict errors (409)
//	06: Rate limiting errors (429)
//	07: Internal errors (500)
//	08: Database errors (500)
//	09: Cache errors (500)
//	10: Network errors (502/503)
//	11: Timeout errors (504)
//	12: Configuration errors (500)
//
// Usage:
//
//	// Using predefined errors
//	return errors.ErrInvalidParam.WithMessage("username is required")
//
//	// Wrapping underlying errors
//	return errors.ErrDatabase.WithCause(err)
//
//	// Creating custom errors
//	var ErrCustom = errors.Register(&errors.Errno{
//	    Code:      errors.MakeCode(20, 1, 1), // Business service, request error
//	    HTTP:      http.StatusBadRequest,
//	    GRPCCode:  codes.InvalidArgument,
//	    MessageEN: "Custom error",
//	    MessageZH: "自定义错误",
//	})
package errors

import (
	"fmt"
	"net/http"
	"sync"

	"google.golang.org/grpc/codes"
)

// Errno represents a structured error with code and messages.
type Errno struct {
	// Code is the unique error code
	Code int `json:"code"`

	// HTTP is the HTTP status code to return
	HTTP int `json:"-"`

	// GRPCCode is the gRPC status code
	GRPCCode codes.Code `json:"-"`

	// MessageEN is the English error message
	MessageEN string `json:"message"`

	// MessageZH is the Chinese error message
	MessageZH string `json:"message_zh,omitempty"`

	// cause is the underlying error
	cause error
}

// Error implements the error interface.
func (e *Errno) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("errno %d: %s: %v", e.Code, e.MessageEN, e.cause)
	}
	return fmt.Sprintf("errno %d: %s", e.Code, e.MessageEN)
}

// Unwrap returns the underlying cause.
func (e *Errno) Unwrap() error {
	return e.cause
}

// WithCause creates a new Errno with the given cause.
func (e *Errno) WithCause(cause error) *Errno {
	return &Errno{
		Code:      e.Code,
		HTTP:      e.HTTP,
		GRPCCode:  e.GRPCCode,
		MessageEN: e.MessageEN,
		MessageZH: e.MessageZH,
		cause:     cause,
	}
}

// WithMessage creates a new Errno with custom English message.
func (e *Errno) WithMessage(msg string) *Errno {
	return &Errno{
		Code:      e.Code,
		HTTP:      e.HTTP,
		GRPCCode:  e.GRPCCode,
		MessageEN: msg,
		MessageZH: e.MessageZH,
		cause:     e.cause,
	}
}

// WithMessagef creates a new Errno with formatted English message.
func (e *Errno) WithMessagef(format string, args ...interface{}) *Errno {
	return &Errno{
		Code:      e.Code,
		HTTP:      e.HTTP,
		GRPCCode:  e.GRPCCode,
		MessageEN: fmt.Sprintf(format, args...),
		MessageZH: e.MessageZH,
		cause:     e.cause,
	}
}

// Message returns the message based on language.
func (e *Errno) Message(lang string) string {
	if lang == "zh" || lang == "zh-CN" || lang == "zh_CN" {
		if e.MessageZH != "" {
			return e.MessageZH
		}
	}
	return e.MessageEN
}

// HTTPStatus returns the HTTP status code.
func (e *Errno) HTTPStatus() int {
	if e.HTTP != 0 {
		return e.HTTP
	}
	return http.StatusInternalServerError
}

// GRPCStatus returns the gRPC status code.
func (e *Errno) GRPCStatus() codes.Code {
	if e.GRPCCode != codes.OK {
		return e.GRPCCode
	}
	return codes.Internal
}

// Is checks if this error matches the target error code.
func (e *Errno) Is(target error) bool {
	if t, ok := target.(*Errno); ok {
		return e.Code == t.Code
	}
	return false
}

// errnoRegistry stores all registered error codes for uniqueness validation.
var (
	errnoRegistry = make(map[int]*Errno)
	registryMu    sync.RWMutex
)

// Register registers an Errno and validates uniqueness.
// Panics if the code is already registered.
func Register(e *Errno) *Errno {
	registryMu.Lock()
	defer registryMu.Unlock()

	if existing, ok := errnoRegistry[e.Code]; ok {
		panic(fmt.Sprintf("errno code %d already registered: %s", e.Code, existing.MessageEN))
	}
	errnoRegistry[e.Code] = e
	return e
}

// MustRegister is an alias for Register for consistency.
func MustRegister(e *Errno) *Errno {
	return Register(e)
}

// Lookup returns the registered Errno for the given code.
func Lookup(code int) (*Errno, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	e, ok := errnoRegistry[code]
	return e, ok
}

// New creates a new Errno with the given parameters.
func New(code int, httpStatus int, grpcCode codes.Code, messageEN, messageZH string) *Errno {
	return &Errno{
		Code:      code,
		HTTP:      httpStatus,
		GRPCCode:  grpcCode,
		MessageEN: messageEN,
		MessageZH: messageZH,
	}
}

// FromError converts any error to Errno.
// If err is already an Errno, returns it directly.
// Otherwise, wraps it as ErrInternal.
func FromError(err error) *Errno {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Errno); ok {
		return e
	}
	return ErrInternal.WithCause(err)
}

// IsCode checks if the error has the given error code.
func IsCode(err error, code int) bool {
	if e, ok := err.(*Errno); ok {
		return e.Code == code
	}
	return false
}

// GetCode returns the error code from an error.
// Returns -1 if the error is not an Errno.
func GetCode(err error) int {
	if e, ok := err.(*Errno); ok {
		return e.Code
	}
	return -1
}

// GetAllRegistered returns all registered error codes.
// This is useful for documentation and debugging.
func GetAllRegistered() map[int]*Errno {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make(map[int]*Errno, len(errnoRegistry))
	for k, v := range errnoRegistry {
		result[k] = v
	}
	return result
}

// RegistrySize returns the number of registered error codes.
func RegistrySize() int {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return len(errnoRegistry)
}

// WithMessageZH creates a new Errno with custom Chinese message.
func (e *Errno) WithMessageZH(msg string) *Errno {
	return &Errno{
		Code:      e.Code,
		HTTP:      e.HTTP,
		GRPCCode:  e.GRPCCode,
		MessageEN: e.MessageEN,
		MessageZH: msg,
		cause:     e.cause,
	}
}

// WithMessages creates a new Errno with custom English and Chinese messages.
func (e *Errno) WithMessages(en, zh string) *Errno {
	return &Errno{
		Code:      e.Code,
		HTTP:      e.HTTP,
		GRPCCode:  e.GRPCCode,
		MessageEN: en,
		MessageZH: zh,
		cause:     e.cause,
	}
}

// Format implements fmt.Formatter for better error formatting.
func (e *Errno) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "errno %d [HTTP %d, gRPC %s]: %s", e.Code, e.HTTP, e.GRPCCode.String(), e.MessageEN)
			if e.MessageZH != "" {
				_, _ = fmt.Fprintf(s, " (%s)", e.MessageZH)
			}
			if e.cause != nil {
				_, _ = fmt.Fprintf(s, "\ncaused by: %+v", e.cause)
			}
			return
		}
		fallthrough
	case 's':
		_, _ = fmt.Fprint(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}
