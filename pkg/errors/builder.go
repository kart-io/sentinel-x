package errors

import (
	"fmt"
	"net/http"
	"sync"

	"google.golang.org/grpc/codes"
)

// ============================================================================
// Service Registration for External Modules
// ============================================================================

// serviceRegistry tracks registered service codes to prevent conflicts.
var (
	serviceRegistry = make(map[int]string) // service code -> service name
	serviceMu       sync.RWMutex
)

// RegisterService registers a service code with a name.
// This should be called once during service initialization.
// Panics if the service code is already registered by another service.
//
// Example:
//
//	func init() {
//	    errors.RegisterService(25, "order-service")
//	}
func RegisterService(code int, name string) {
	serviceMu.Lock()
	defer serviceMu.Unlock()

	if existing, ok := serviceRegistry[code]; ok {
		if existing != name {
			panic(fmt.Sprintf("service code %d already registered by '%s', cannot register for '%s'", code, existing, name))
		}
		return // Already registered with same name, ignore
	}
	serviceRegistry[code] = name
}

// GetServiceName returns the registered name for a service code.
func GetServiceName(code int) (string, bool) {
	serviceMu.RLock()
	defer serviceMu.RUnlock()
	name, ok := serviceRegistry[code]
	return name, ok
}

// GetAllServices returns all registered services.
func GetAllServices() map[int]string {
	serviceMu.RLock()
	defer serviceMu.RUnlock()

	result := make(map[int]string, len(serviceRegistry))
	for k, v := range serviceRegistry {
		result[k] = v
	}
	return result
}

// ============================================================================
// Error Builder for External Modules
// ============================================================================

// ErrnoBuilder provides a fluent API for building error codes.
// This is the recommended way for external modules to define errors.
//
// Example:
//
//	var ErrOrderNotFound = errors.NewBuilder(ServiceOrder, errors.CategoryResource, 1).
//	    HTTP(http.StatusNotFound).
//	    GRPC(codes.NotFound).
//	    Message("Order not found", "订单不存在").
//	    MustBuild()
type ErrnoBuilder struct {
	service   int
	category  int
	sequence  int
	http      int
	grpc      codes.Code
	messageEN string
	messageZH string
}

// NewBuilder creates a new ErrnoBuilder with the given service, category, and sequence.
//
// Parameters:
//   - service: Service/module code (use constants like ServiceUser, or your own 20-79)
//   - category: Error category (use constants like CategoryRequest, CategoryResource, etc.)
//   - sequence: Unique sequence number within this service+category (0-999)
//
// Example:
//
//	// Define service code for your module
//	const ServiceOrder = 25
//
//	// Create errors using builder
//	var ErrOrderNotFound = errors.NewBuilder(ServiceOrder, errors.CategoryResource, 1).
//	    HTTP(http.StatusNotFound).
//	    GRPC(codes.NotFound).
//	    Message("Order not found", "订单不存在").
//	    MustBuild()
func NewBuilder(service, category, sequence int) *ErrnoBuilder {
	return &ErrnoBuilder{
		service:  service,
		category: category,
		sequence: sequence,
		http:     http.StatusInternalServerError, // default
		grpc:     codes.Internal,                 // default
	}
}

// HTTP sets the HTTP status code.
func (b *ErrnoBuilder) HTTP(status int) *ErrnoBuilder {
	b.http = status
	return b
}

// GRPC sets the gRPC status code.
func (b *ErrnoBuilder) GRPC(code codes.Code) *ErrnoBuilder {
	b.grpc = code
	return b
}

// Message sets both English and Chinese messages.
func (b *ErrnoBuilder) Message(en, zh string) *ErrnoBuilder {
	b.messageEN = en
	b.messageZH = zh
	return b
}

// MessageEN sets only the English message.
func (b *ErrnoBuilder) MessageEN(en string) *ErrnoBuilder {
	b.messageEN = en
	return b
}

// MessageZH sets only the Chinese message.
func (b *ErrnoBuilder) MessageZH(zh string) *ErrnoBuilder {
	b.messageZH = zh
	return b
}

// Build creates and registers the Errno.
// Returns an error if registration fails (e.g., duplicate code).
func (b *ErrnoBuilder) Build() (*Errno, error) {
	if b.messageEN == "" {
		return nil, fmt.Errorf("English message is required")
	}

	e := &Errno{
		Code:      MakeCode(b.service, b.category, b.sequence),
		HTTP:      b.http,
		GRPCCode:  b.grpc,
		MessageEN: b.messageEN,
		MessageZH: b.messageZH,
	}

	// Try to register
	registryMu.Lock()
	defer registryMu.Unlock()

	if existing, ok := errnoRegistry[e.Code]; ok {
		return nil, fmt.Errorf("errno code %d already registered: %s", e.Code, existing.MessageEN)
	}
	errnoRegistry[e.Code] = e

	return e, nil
}

// MustBuild creates and registers the Errno.
// Panics if registration fails.
func (b *ErrnoBuilder) MustBuild() *Errno {
	e, err := b.Build()
	if err != nil {
		panic(err)
	}
	return e
}

// ============================================================================
// Preset Builders for Common Categories
// ============================================================================

// NewRequestError creates a builder for request/validation errors (HTTP 400).
func NewRequestError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryRequest, sequence).
		HTTP(http.StatusBadRequest).
		GRPC(codes.InvalidArgument)
}

// NewAuthError creates a builder for authentication errors (HTTP 401).
func NewAuthError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryAuth, sequence).
		HTTP(http.StatusUnauthorized).
		GRPC(codes.Unauthenticated)
}

// NewPermissionError creates a builder for authorization errors (HTTP 403).
func NewPermissionError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryPermission, sequence).
		HTTP(http.StatusForbidden).
		GRPC(codes.PermissionDenied)
}

// NewNotFoundError creates a builder for resource not found errors (HTTP 404).
func NewNotFoundError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryResource, sequence).
		HTTP(http.StatusNotFound).
		GRPC(codes.NotFound)
}

// NewConflictError creates a builder for conflict errors (HTTP 409).
func NewConflictError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryConflict, sequence).
		HTTP(http.StatusConflict).
		GRPC(codes.AlreadyExists)
}

// NewRateLimitError creates a builder for rate limiting errors (HTTP 429).
func NewRateLimitError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryRateLimit, sequence).
		HTTP(http.StatusTooManyRequests).
		GRPC(codes.ResourceExhausted)
}

// NewInternalError creates a builder for internal errors (HTTP 500).
func NewInternalError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryInternal, sequence).
		HTTP(http.StatusInternalServerError).
		GRPC(codes.Internal)
}

// NewDatabaseError creates a builder for database errors (HTTP 500).
func NewDatabaseError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryDatabase, sequence).
		HTTP(http.StatusInternalServerError).
		GRPC(codes.Internal)
}

// NewCacheError creates a builder for cache errors (HTTP 500).
func NewCacheError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryCache, sequence).
		HTTP(http.StatusInternalServerError).
		GRPC(codes.Internal)
}

// NewNetworkError creates a builder for network errors (HTTP 503).
func NewNetworkError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryNetwork, sequence).
		HTTP(http.StatusServiceUnavailable).
		GRPC(codes.Unavailable)
}

// NewTimeoutError creates a builder for timeout errors (HTTP 504).
func NewTimeoutError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryTimeout, sequence).
		HTTP(http.StatusGatewayTimeout).
		GRPC(codes.DeadlineExceeded)
}

// NewConfigError creates a builder for configuration errors (HTTP 500).
func NewConfigError(service, sequence int) *ErrnoBuilder {
	return NewBuilder(service, CategoryConfig, sequence).
		HTTP(http.StatusInternalServerError).
		GRPC(codes.Internal)
}

// ============================================================================
// Quick Creation Functions
// ============================================================================

// NewRequestErr quickly creates and registers a request error.
func NewRequestErr(service, sequence int, en, zh string) *Errno {
	return NewRequestError(service, sequence).Message(en, zh).MustBuild()
}

// NewAuthErr quickly creates and registers an authentication error.
func NewAuthErr(service, sequence int, en, zh string) *Errno {
	return NewAuthError(service, sequence).Message(en, zh).MustBuild()
}

// NewPermissionErr quickly creates and registers a permission error.
func NewPermissionErr(service, sequence int, en, zh string) *Errno {
	return NewPermissionError(service, sequence).Message(en, zh).MustBuild()
}

// NewNotFoundErr quickly creates and registers a not found error.
func NewNotFoundErr(service, sequence int, en, zh string) *Errno {
	return NewNotFoundError(service, sequence).Message(en, zh).MustBuild()
}

// NewConflictErr quickly creates and registers a conflict error.
func NewConflictErr(service, sequence int, en, zh string) *Errno {
	return NewConflictError(service, sequence).Message(en, zh).MustBuild()
}

// NewInternalErr quickly creates and registers an internal error.
func NewInternalErr(service, sequence int, en, zh string) *Errno {
	return NewInternalError(service, sequence).Message(en, zh).MustBuild()
}
