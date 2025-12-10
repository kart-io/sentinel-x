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
// Core Error Creation Functions
// ============================================================================

// validateCodeParams validates service, category, and sequence parameters.
func validateCodeParams(service, category, sequence int) {
	if service < 0 || service > 99 {
		panic(fmt.Sprintf("errors: service code must be 0-99, got %d", service))
	}
	if category < 0 || category > 99 {
		panic(fmt.Sprintf("errors: category code must be 0-99, got %d", category))
	}
	if sequence < 0 || sequence > 999 {
		panic(fmt.Sprintf("errors: sequence must be 0-999, got %d", sequence))
	}
}

// registerErrno registers an Errno in the global registry.
// Returns error if code is already registered.
func registerErrno(e *Errno) (*Errno, error) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if existing, ok := errnoRegistry[e.Code]; ok {
		return nil, fmt.Errorf("errno code %d already registered: %s", e.Code, existing.MessageEN)
	}
	errnoRegistry[e.Code] = e
	return e, nil
}

// mustRegisterErrno registers an Errno and panics on failure.
func mustRegisterErrno(e *Errno) *Errno {
	registered, err := registerErrno(e)
	if err != nil {
		panic(err)
	}
	return registered
}

// NewError creates and registers a new Errno with the given parameters.
// This is the most flexible function for custom error definitions.
// Panics if registration fails or if messageEN is empty.
//
// Example:
//
//	var ErrCustom = errors.NewError(25, errors.CategoryRequest, 1,
//	    http.StatusBadRequest, codes.InvalidArgument,
//	    "Custom error", "自定义错误")
func NewError(service, category, sequence int, httpStatus int, grpcCode codes.Code, messageEN, messageZH string) *Errno {
	validateCodeParams(service, category, sequence)
	if messageEN == "" {
		panic("errors: english message is required")
	}

	e := &Errno{
		Code:      MakeCode(service, category, sequence),
		HTTP:      httpStatus,
		GRPCCode:  grpcCode,
		MessageEN: messageEN,
		MessageZH: messageZH,
	}

	return mustRegisterErrno(e)
}

// ============================================================================
// Category-Specific Error Creation Functions (Recommended API)
// ============================================================================

// NewRequestErr creates and registers a request/validation error (HTTP 400).
// This is the recommended way to create request errors.
//
// Example:
//
//	var ErrInvalidInput = errors.NewRequestErr(ServiceOrder, 1,
//	    "Invalid input", "输入无效")
func NewRequestErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryRequest, sequence, http.StatusBadRequest, codes.InvalidArgument, en, zh)
}

// NewAuthErr creates and registers an authentication error (HTTP 401).
//
// Example:
//
//	var ErrLoginFailed = errors.NewAuthErr(ServiceUser, 1,
//	    "Login failed", "登录失败")
func NewAuthErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryAuth, sequence, http.StatusUnauthorized, codes.Unauthenticated, en, zh)
}

// NewPermissionErr creates and registers a permission/authorization error (HTTP 403).
//
// Example:
//
//	var ErrNoAccess = errors.NewPermissionErr(ServiceUser, 1,
//	    "No permission", "无权限")
func NewPermissionErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryPermission, sequence, http.StatusForbidden, codes.PermissionDenied, en, zh)
}

// NewNotFoundErr creates and registers a not found error (HTTP 404).
//
// Example:
//
//	var ErrOrderNotFound = errors.NewNotFoundErr(ServiceOrder, 1,
//	    "Order not found", "订单不存在")
func NewNotFoundErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryResource, sequence, http.StatusNotFound, codes.NotFound, en, zh)
}

// NewConflictErr creates and registers a conflict error (HTTP 409).
//
// Example:
//
//	var ErrAlreadyExists = errors.NewConflictErr(ServiceOrder, 1,
//	    "Order already exists", "订单已存在")
func NewConflictErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryConflict, sequence, http.StatusConflict, codes.AlreadyExists, en, zh)
}

// NewRateLimitErr creates and registers a rate limit error (HTTP 429).
//
// Example:
//
//	var ErrTooManyRequests = errors.NewRateLimitErr(ServiceAPI, 1,
//	    "Too many requests", "请求过于频繁")
func NewRateLimitErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryRateLimit, sequence, http.StatusTooManyRequests, codes.ResourceExhausted, en, zh)
}

// NewInternalErr creates and registers an internal error (HTTP 500).
//
// Example:
//
//	var ErrProcessFailed = errors.NewInternalErr(ServiceOrder, 1,
//	    "Process failed", "处理失败")
func NewInternalErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryInternal, sequence, http.StatusInternalServerError, codes.Internal, en, zh)
}

// NewDatabaseErr creates and registers a database error (HTTP 500).
//
// Example:
//
//	var ErrDBQuery = errors.NewDatabaseErr(ServiceOrder, 1,
//	    "Database query failed", "数据库查询失败")
func NewDatabaseErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryDatabase, sequence, http.StatusInternalServerError, codes.Internal, en, zh)
}

// NewCacheErr creates and registers a cache error (HTTP 500).
//
// Example:
//
//	var ErrCacheFailed = errors.NewCacheErr(ServiceOrder, 1,
//	    "Cache operation failed", "缓存操作失败")
func NewCacheErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryCache, sequence, http.StatusInternalServerError, codes.Internal, en, zh)
}

// NewNetworkErr creates and registers a network error (HTTP 503).
//
// Example:
//
//	var ErrConnectionFailed = errors.NewNetworkErr(ServiceAPI, 1,
//	    "Connection failed", "连接失败")
func NewNetworkErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryNetwork, sequence, http.StatusServiceUnavailable, codes.Unavailable, en, zh)
}

// NewTimeoutErr creates and registers a timeout error (HTTP 504).
//
// Example:
//
//	var ErrOperationTimeout = errors.NewTimeoutErr(ServiceAPI, 1,
//	    "Operation timeout", "操作超时")
func NewTimeoutErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryTimeout, sequence, http.StatusGatewayTimeout, codes.DeadlineExceeded, en, zh)
}

// NewConfigErr creates and registers a configuration error (HTTP 500).
//
// Example:
//
//	var ErrInvalidConfig = errors.NewConfigErr(ServiceAPI, 1,
//	    "Invalid configuration", "配置无效")
func NewConfigErr(service, sequence int, en, zh string) *Errno {
	return NewError(service, CategoryConfig, sequence, http.StatusInternalServerError, codes.Internal, en, zh)
}
