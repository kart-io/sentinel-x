package errors

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

// ============================================================================
// Success
// ============================================================================

// OK represents a successful operation.
var OK = Register(&Errno{
	Code:      0,
	HTTP:      http.StatusOK,
	GRPCCode:  codes.OK,
	MessageEN: "Success",
	MessageZH: "成功",
})

// ============================================================================
// Request Errors (Category: 01)
// ============================================================================

var (
	// ErrBadRequest indicates a malformed request.
	ErrBadRequest = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRequest, 0),
		HTTP:      http.StatusBadRequest,
		GRPCCode:  codes.InvalidArgument,
		MessageEN: "Bad request",
		MessageZH: "请求错误",
	})

	// ErrInvalidParam indicates an invalid parameter.
	ErrInvalidParam = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRequest, 1),
		HTTP:      http.StatusBadRequest,
		GRPCCode:  codes.InvalidArgument,
		MessageEN: "Invalid parameter",
		MessageZH: "参数无效",
	})

	// ErrMissingParam indicates a missing required parameter.
	ErrMissingParam = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRequest, 2),
		HTTP:      http.StatusBadRequest,
		GRPCCode:  codes.InvalidArgument,
		MessageEN: "Missing required parameter",
		MessageZH: "缺少必需参数",
	})

	// ErrInvalidFormat indicates an invalid format.
	ErrInvalidFormat = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRequest, 3),
		HTTP:      http.StatusBadRequest,
		GRPCCode:  codes.InvalidArgument,
		MessageEN: "Invalid format",
		MessageZH: "格式无效",
	})

	// ErrValidationFailed indicates validation failure.
	ErrValidationFailed = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRequest, 4),
		HTTP:      http.StatusBadRequest,
		GRPCCode:  codes.InvalidArgument,
		MessageEN: "Validation failed",
		MessageZH: "验证失败",
	})

	// ErrRequestTooLarge indicates the request body is too large.
	ErrRequestTooLarge = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRequest, 5),
		HTTP:      http.StatusRequestEntityTooLarge,
		GRPCCode:  codes.InvalidArgument,
		MessageEN: "Request entity too large",
		MessageZH: "请求体过大",
	})

	// ErrUnsupportedMediaType indicates unsupported media type.
	ErrUnsupportedMediaType = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRequest, 6),
		HTTP:      http.StatusUnsupportedMediaType,
		GRPCCode:  codes.InvalidArgument,
		MessageEN: "Unsupported media type",
		MessageZH: "不支持的媒体类型",
	})
)

// ============================================================================
// Authentication Errors (Category: 02)
// ============================================================================

var (
	// ErrUnauthorized indicates the request is not authenticated.
	ErrUnauthorized = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryAuth, 0),
		HTTP:      http.StatusUnauthorized,
		GRPCCode:  codes.Unauthenticated,
		MessageEN: "Unauthorized",
		MessageZH: "未认证",
	})

	// ErrInvalidToken indicates the token is invalid.
	ErrInvalidToken = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryAuth, 1),
		HTTP:      http.StatusUnauthorized,
		GRPCCode:  codes.Unauthenticated,
		MessageEN: "Invalid token",
		MessageZH: "令牌无效",
	})

	// ErrTokenExpired indicates the token has expired.
	ErrTokenExpired = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryAuth, 2),
		HTTP:      http.StatusUnauthorized,
		GRPCCode:  codes.Unauthenticated,
		MessageEN: "Token expired",
		MessageZH: "令牌已过期",
	})

	// ErrInvalidCredentials indicates invalid credentials.
	ErrInvalidCredentials = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryAuth, 3),
		HTTP:      http.StatusUnauthorized,
		GRPCCode:  codes.Unauthenticated,
		MessageEN: "Invalid credentials",
		MessageZH: "凭证无效",
	})

	// ErrTokenRevoked indicates the token has been revoked.
	ErrTokenRevoked = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryAuth, 4),
		HTTP:      http.StatusUnauthorized,
		GRPCCode:  codes.Unauthenticated,
		MessageEN: "Token revoked",
		MessageZH: "令牌已撤销",
	})

	// ErrSessionExpired indicates the session has expired.
	ErrSessionExpired = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryAuth, 5),
		HTTP:      http.StatusUnauthorized,
		GRPCCode:  codes.Unauthenticated,
		MessageEN: "Session expired",
		MessageZH: "会话已过期",
	})
)

// ============================================================================
// Authorization Errors (Category: 03)
// ============================================================================

var (
	// ErrForbidden indicates the request is forbidden.
	ErrForbidden = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryPermission, 0),
		HTTP:      http.StatusForbidden,
		GRPCCode:  codes.PermissionDenied,
		MessageEN: "Forbidden",
		MessageZH: "禁止访问",
	})

	// ErrNoPermission indicates no permission for the operation.
	ErrNoPermission = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryPermission, 1),
		HTTP:      http.StatusForbidden,
		GRPCCode:  codes.PermissionDenied,
		MessageEN: "No permission",
		MessageZH: "无权限",
	})

	// ErrResourceLocked indicates the resource is locked.
	ErrResourceLocked = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryPermission, 2),
		HTTP:      http.StatusLocked,
		GRPCCode:  codes.PermissionDenied,
		MessageEN: "Resource locked",
		MessageZH: "资源已锁定",
	})

	// ErrAccountDisabled indicates the account is disabled.
	ErrAccountDisabled = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryPermission, 3),
		HTTP:      http.StatusForbidden,
		GRPCCode:  codes.PermissionDenied,
		MessageEN: "Account disabled",
		MessageZH: "账号已禁用",
	})

	// ErrIPBlocked indicates the IP is blocked.
	ErrIPBlocked = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryPermission, 4),
		HTTP:      http.StatusForbidden,
		GRPCCode:  codes.PermissionDenied,
		MessageEN: "IP blocked",
		MessageZH: "IP 已被封禁",
	})
)

// ============================================================================
// Resource Errors (Category: 04)
// ============================================================================

var (
	// ErrNotFound indicates the resource is not found.
	ErrNotFound = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryResource, 0),
		HTTP:      http.StatusNotFound,
		GRPCCode:  codes.NotFound,
		MessageEN: "Resource not found",
		MessageZH: "资源不存在",
	})

	// ErrUserNotFound indicates the user is not found.
	ErrUserNotFound = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryResource, 1),
		HTTP:      http.StatusNotFound,
		GRPCCode:  codes.NotFound,
		MessageEN: "User not found",
		MessageZH: "用户不存在",
	})

	// ErrRecordNotFound indicates the record is not found.
	ErrRecordNotFound = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryResource, 2),
		HTTP:      http.StatusNotFound,
		GRPCCode:  codes.NotFound,
		MessageEN: "Record not found",
		MessageZH: "记录不存在",
	})

	// ErrFileNotFound indicates the file is not found.
	ErrFileNotFound = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryResource, 3),
		HTTP:      http.StatusNotFound,
		GRPCCode:  codes.NotFound,
		MessageEN: "File not found",
		MessageZH: "文件不存在",
	})

	// ErrRouteNotFound indicates the route is not found.
	ErrRouteNotFound = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryResource, 4),
		HTTP:      http.StatusNotFound,
		GRPCCode:  codes.NotFound,
		MessageEN: "Route not found",
		MessageZH: "路由不存在",
	})
)

// ============================================================================
// Conflict Errors (Category: 05)
// ============================================================================

var (
	// ErrConflict indicates a resource conflict.
	ErrConflict = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryConflict, 0),
		HTTP:      http.StatusConflict,
		GRPCCode:  codes.AlreadyExists,
		MessageEN: "Resource conflict",
		MessageZH: "资源冲突",
	})

	// ErrAlreadyExists indicates the resource already exists.
	ErrAlreadyExists = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryConflict, 1),
		HTTP:      http.StatusConflict,
		GRPCCode:  codes.AlreadyExists,
		MessageEN: "Resource already exists",
		MessageZH: "资源已存在",
	})

	// ErrDuplicateKey indicates a duplicate key.
	ErrDuplicateKey = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryConflict, 2),
		HTTP:      http.StatusConflict,
		GRPCCode:  codes.AlreadyExists,
		MessageEN: "Duplicate key",
		MessageZH: "键值重复",
	})

	// ErrVersionConflict indicates a version conflict.
	ErrVersionConflict = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryConflict, 3),
		HTTP:      http.StatusConflict,
		GRPCCode:  codes.AlreadyExists,
		MessageEN: "Version conflict",
		MessageZH: "版本冲突",
	})
)

// ============================================================================
// Rate Limit Errors (Category: 06)
// ============================================================================

var (
	// ErrTooManyRequests indicates too many requests.
	ErrTooManyRequests = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRateLimit, 0),
		HTTP:      http.StatusTooManyRequests,
		GRPCCode:  codes.ResourceExhausted,
		MessageEN: "Too many requests",
		MessageZH: "请求过于频繁",
	})

	// ErrRateLimitExceeded indicates rate limit exceeded.
	ErrRateLimitExceeded = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRateLimit, 1),
		HTTP:      http.StatusTooManyRequests,
		GRPCCode:  codes.ResourceExhausted,
		MessageEN: "Rate limit exceeded",
		MessageZH: "超出速率限制",
	})

	// ErrQuotaExceeded indicates quota exceeded.
	ErrQuotaExceeded = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryRateLimit, 2),
		HTTP:      http.StatusTooManyRequests,
		GRPCCode:  codes.ResourceExhausted,
		MessageEN: "Quota exceeded",
		MessageZH: "配额已用尽",
	})
)

// ============================================================================
// Internal Errors (Category: 07)
// ============================================================================

var (
	// ErrInternal indicates an internal server error.
	ErrInternal = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryInternal, 0),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Internal server error",
		MessageZH: "服务器内部错误",
	})

	// ErrUnknown indicates an unknown error.
	ErrUnknown = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryInternal, 1),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Unknown,
		MessageEN: "Unknown error",
		MessageZH: "未知错误",
	})

	// ErrPanic indicates a service panic.
	ErrPanic = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryInternal, 2),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Service panic",
		MessageZH: "服务崩溃",
	})

	// ErrNotImplemented indicates the feature is not implemented.
	ErrNotImplemented = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryInternal, 3),
		HTTP:      http.StatusNotImplemented,
		GRPCCode:  codes.Unimplemented,
		MessageEN: "Not implemented",
		MessageZH: "功能未实现",
	})
)

// ============================================================================
// Database Errors (Category: 08)
// ============================================================================

var (
	// ErrDatabase indicates a database error.
	ErrDatabase = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryDatabase, 0),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Database error",
		MessageZH: "数据库错误",
	})

	// ErrDBConnection indicates database connection failure.
	ErrDBConnection = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryDatabase, 1),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Unavailable,
		MessageEN: "Database connection failed",
		MessageZH: "数据库连接失败",
	})

	// ErrDBQuery indicates database query failure.
	ErrDBQuery = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryDatabase, 2),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Database query failed",
		MessageZH: "数据库查询失败",
	})

	// ErrDBTransaction indicates database transaction failure.
	ErrDBTransaction = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryDatabase, 3),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Database transaction failed",
		MessageZH: "数据库事务失败",
	})

	// ErrDBDeadlock indicates database deadlock.
	ErrDBDeadlock = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryDatabase, 4),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Database deadlock",
		MessageZH: "数据库死锁",
	})
)

// ============================================================================
// Cache Errors (Category: 09)
// ============================================================================

var (
	// ErrCache indicates a cache error.
	ErrCache = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryCache, 0),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Cache error",
		MessageZH: "缓存错误",
	})

	// ErrCacheConnection indicates cache connection failure.
	ErrCacheConnection = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryCache, 1),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Unavailable,
		MessageEN: "Cache connection failed",
		MessageZH: "缓存连接失败",
	})

	// ErrCacheMiss indicates cache miss.
	ErrCacheMiss = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryCache, 2),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.NotFound,
		MessageEN: "Cache miss",
		MessageZH: "缓存未命中",
	})

	// ErrCacheExpired indicates cache expired.
	ErrCacheExpired = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryCache, 3),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.NotFound,
		MessageEN: "Cache expired",
		MessageZH: "缓存已过期",
	})
)

// ============================================================================
// Network Errors (Category: 10)
// ============================================================================

var (
	// ErrNetwork indicates a network error.
	ErrNetwork = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryNetwork, 0),
		HTTP:      http.StatusBadGateway,
		GRPCCode:  codes.Unavailable,
		MessageEN: "Network error",
		MessageZH: "网络错误",
	})

	// ErrServiceUnavailable indicates the service is unavailable.
	ErrServiceUnavailable = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryNetwork, 1),
		HTTP:      http.StatusServiceUnavailable,
		GRPCCode:  codes.Unavailable,
		MessageEN: "Service unavailable",
		MessageZH: "服务不可用",
	})

	// ErrConnectionRefused indicates connection refused.
	ErrConnectionRefused = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryNetwork, 2),
		HTTP:      http.StatusBadGateway,
		GRPCCode:  codes.Unavailable,
		MessageEN: "Connection refused",
		MessageZH: "连接被拒绝",
	})

	// ErrDNSResolution indicates DNS resolution failure.
	ErrDNSResolution = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryNetwork, 3),
		HTTP:      http.StatusBadGateway,
		GRPCCode:  codes.Unavailable,
		MessageEN: "DNS resolution failed",
		MessageZH: "DNS 解析失败",
	})
)

// ============================================================================
// Timeout Errors (Category: 11)
// ============================================================================

var (
	// ErrTimeout indicates operation timeout.
	ErrTimeout = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryTimeout, 0),
		HTTP:      http.StatusGatewayTimeout,
		GRPCCode:  codes.DeadlineExceeded,
		MessageEN: "Operation timeout",
		MessageZH: "操作超时",
	})

	// ErrRequestTimeout indicates request timeout.
	ErrRequestTimeout = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryTimeout, 1),
		HTTP:      http.StatusRequestTimeout,
		GRPCCode:  codes.DeadlineExceeded,
		MessageEN: "Request timeout",
		MessageZH: "请求超时",
	})

	// ErrGatewayTimeout indicates gateway timeout.
	ErrGatewayTimeout = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryTimeout, 2),
		HTTP:      http.StatusGatewayTimeout,
		GRPCCode:  codes.DeadlineExceeded,
		MessageEN: "Gateway timeout",
		MessageZH: "网关超时",
	})

	// ErrContextCanceled indicates context canceled.
	ErrContextCanceled = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryTimeout, 3),
		HTTP:      499, // Client Closed Request
		GRPCCode:  codes.Canceled,
		MessageEN: "Context canceled",
		MessageZH: "上下文已取消",
	})
)

// ============================================================================
// Configuration Errors (Category: 12)
// ============================================================================

var (
	// ErrConfig indicates a configuration error.
	ErrConfig = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryConfig, 0),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Configuration error",
		MessageZH: "配置错误",
	})

	// ErrConfigNotFound indicates configuration not found.
	ErrConfigNotFound = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryConfig, 1),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Configuration not found",
		MessageZH: "配置不存在",
	})

	// ErrConfigInvalid indicates invalid configuration.
	ErrConfigInvalid = Register(&Errno{
		Code:      MakeCode(ServiceCommon, CategoryConfig, 2),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Invalid configuration",
		MessageZH: "配置无效",
	})
)
