# Error Builder Simplification

## Overview

The Builder pattern in `pkg/errors/builder.go` has been simplified to reduce complexity while maintaining full backward compatibility. The new API is cleaner and more direct, eliminating unnecessary method chaining.

## What Changed

### Before (Builder Pattern - Now Deprecated)

```go
// Complex method chaining
var ErrOrderNotFound = errors.NewNotFoundError(ServiceOrder, 1).
    Message("Order not found", "订单不存在").
    MustBuild()

// Custom HTTP/gRPC codes required full builder
var ErrOrderExpired = errors.NewBuilder(ServiceOrder, errors.CategoryConflict, 10).
    HTTP(http.StatusGone).
    GRPC(codes.FailedPrecondition).
    Message("Order has expired", "订单已过期").
    MustBuild()
```

### After (Simplified Functions - Recommended)

```go
// Simple function call
var ErrOrderNotFound = errors.NewNotFoundErr(ServiceOrder, 1,
    "Order not found", "订单不存在")

// Custom HTTP/gRPC codes use NewError
var ErrOrderExpired = errors.NewError(ServiceOrder, errors.CategoryConflict, 10,
    http.StatusGone, codes.FailedPrecondition,
    "Order has expired", "订单已过期")
```

## Benefits

1. **Simpler API**: Single function call instead of method chaining
2. **Less Code**: Fewer lines, easier to read
3. **Same Functionality**: All features preserved
4. **Backward Compatible**: Old code continues to work
5. **Better Performance**: No intermediate builder objects

## API Reference

### Recommended Functions

All error creation functions follow the same pattern: `NewXxxErr(service, sequence, en, zh)`

#### Request/Validation Errors (HTTP 400)

```go
NewRequestErr(service, sequence int, en, zh string) *Errno
```

#### Authentication Errors (HTTP 401)

```go
NewAuthErr(service, sequence int, en, zh string) *Errno
```

#### Permission Errors (HTTP 403)

```go
NewPermissionErr(service, sequence int, en, zh string) *Errno
```

#### Not Found Errors (HTTP 404)

```go
NewNotFoundErr(service, sequence int, en, zh string) *Errno
```

#### Conflict Errors (HTTP 409)

```go
NewConflictErr(service, sequence int, en, zh string) *Errno
```

#### Rate Limit Errors (HTTP 429)

```go
NewRateLimitErr(service, sequence int, en, zh string) *Errno
```

#### Internal Errors (HTTP 500)

```go
NewInternalErr(service, sequence int, en, zh string) *Errno
```

#### Database Errors (HTTP 500)

```go
NewDatabaseErr(service, sequence int, en, zh string) *Errno
```

#### Cache Errors (HTTP 500)

```go
NewCacheErr(service, sequence int, en, zh string) *Errno
```

#### Network Errors (HTTP 503)

```go
NewNetworkErr(service, sequence int, en, zh string) *Errno
```

#### Timeout Errors (HTTP 504)

```go
NewTimeoutErr(service, sequence int, en, zh string) *Errno
```

#### Configuration Errors (HTTP 500)

```go
NewConfigErr(service, sequence int, en, zh string) *Errno
```

### Advanced Function

For custom HTTP/gRPC status codes:

```go
NewError(service, category, sequence int, httpStatus int, grpcCode codes.Code, messageEN, messageZH string) *Errno
```

## Migration Guide

### Pattern 1: Standard Category Errors

**Before:**
```go
var ErrUserNotFound = errors.NewNotFoundError(ServiceUser, 1).
    Message("User not found", "用户不存在").
    MustBuild()
```

**After:**
```go
var ErrUserNotFound = errors.NewNotFoundErr(ServiceUser, 1,
    "User not found", "用户不存在")
```

### Pattern 2: Custom HTTP/gRPC Codes

**Before:**
```go
var ErrOrderExpired = errors.NewBuilder(ServiceOrder, errors.CategoryConflict, 10).
    HTTP(http.StatusGone).
    GRPC(codes.FailedPrecondition).
    Message("Order has expired", "订单已过期").
    MustBuild()
```

**After:**
```go
var ErrOrderExpired = errors.NewError(ServiceOrder, errors.CategoryConflict, 10,
    http.StatusGone, codes.FailedPrecondition,
    "Order has expired", "订单已过期")
```

### Pattern 3: Multiple Errors

**Before:**
```go
var (
    ErrInvalidAmount = errors.NewRequestError(ServiceOrder, 1).
        Message("Invalid order amount", "订单金额无效").
        MustBuild()

    ErrInvalidQuantity = errors.NewRequestError(ServiceOrder, 2).
        Message("Invalid order quantity", "订单数量无效").
        MustBuild()
)
```

**After:**
```go
var (
    ErrInvalidAmount = errors.NewRequestErr(ServiceOrder, 1,
        "Invalid order amount", "订单金额无效")

    ErrInvalidQuantity = errors.NewRequestErr(ServiceOrder, 2,
        "Invalid order quantity", "订单数量无效")
)
```

## Deprecated Functions

The following are deprecated but still work for backward compatibility:

- `NewBuilder(service, category, sequence)` → Use `NewError` instead
- `NewRequestError(service, sequence)` → Use `NewRequestErr` instead
- `NewAuthError(service, sequence)` → Use `NewAuthErr` instead
- `NewPermissionError(service, sequence)` → Use `NewPermissionErr` instead
- `NewNotFoundError(service, sequence)` → Use `NewNotFoundErr` instead
- `NewConflictError(service, sequence)` → Use `NewConflictErr` instead
- `NewRateLimitError(service, sequence)` → Use `NewRateLimitErr` instead
- `NewInternalError(service, sequence)` → Use `NewInternalErr` instead
- `NewDatabaseError(service, sequence)` → Use `NewDatabaseErr` instead
- `NewCacheError(service, sequence)` → Use `NewCacheErr` instead
- `NewNetworkError(service, sequence)` → Use `NewNetworkErr` instead
- `NewTimeoutError(service, sequence)` → Use `NewTimeoutErr` instead
- `NewConfigError(service, sequence)` → Use `NewConfigErr` instead

## Examples

### Complete Service Error Definition

```go
package myservice

import "github.com/kart-io/sentinel-x/pkg/errors"

const ServiceMyService = 25

func init() {
    errors.RegisterService(ServiceMyService, "my-service")
}

// Request Errors
var (
    ErrInvalidInput = errors.NewRequestErr(ServiceMyService, 1,
        "Invalid input", "输入无效")

    ErrMissingField = errors.NewRequestErr(ServiceMyService, 2,
        "Missing required field", "缺少必需字段")
)

// Resource Errors
var (
    ErrResourceNotFound = errors.NewNotFoundErr(ServiceMyService, 1,
        "Resource not found", "资源不存在")
)

// Conflict Errors
var (
    ErrResourceExists = errors.NewConflictErr(ServiceMyService, 1,
        "Resource already exists", "资源已存在")
)

// Custom Error with Special Status Codes
var (
    ErrResourceGone = errors.NewError(ServiceMyService, errors.CategoryResource, 2,
        http.StatusGone, codes.NotFound,
        "Resource permanently deleted", "资源已永久删除")
)
```

## Implementation Details

### Core Changes

1. **Added `NewError` function**: Direct error creation without builder
2. **Added helper functions**: `validateCodeParams`, `registerErrno`, `mustRegisterErrno`
3. **Simplified category functions**: All `NewXxxErr` functions now call `NewError` directly
4. **Preserved Builder**: Kept for backward compatibility but marked as deprecated
5. **No Breaking Changes**: All existing code continues to work

### File Structure

```
pkg/errors/builder.go
├── Service Registration (unchanged)
├── Core Error Creation Functions (new)
│   ├── validateCodeParams
│   ├── registerErrno
│   ├── mustRegisterErrno
│   └── NewError
├── Category-Specific Functions (simplified)
│   ├── NewRequestErr
│   ├── NewAuthErr
│   ├── NewPermissionErr
│   ├── NewNotFoundErr
│   ├── NewConflictErr
│   ├── NewRateLimitErr
│   ├── NewInternalErr
│   ├── NewDatabaseErr
│   ├── NewCacheErr
│   ├── NewNetworkErr
│   ├── NewTimeoutErr
│   └── NewConfigErr
└── Backward Compatibility (deprecated)
    ├── ErrnoBuilder type
    ├── NewBuilder
    ├── Builder methods (HTTP, GRPC, Message, etc.)
    └── NewXxxError functions
```

## Testing

All existing tests pass without modification:

```bash
go test ./pkg/errors/...
# PASS: All tests pass
```

All example code compiles without changes:

```bash
go build ./example/errors/...
# Success: No errors
```

## Performance Impact

The simplified API has better performance characteristics:

- **Before**: Creates intermediate Builder object, then Errno
- **After**: Creates Errno directly
- **Result**: Fewer allocations, faster execution

## Recommendations

1. **New Code**: Use `NewXxxErr` functions exclusively
2. **Existing Code**: Migrate gradually, no rush (backward compatible)
3. **Complex Cases**: Use `NewError` for custom HTTP/gRPC codes
4. **Code Reviews**: Suggest simplified API for new PRs

## Questions?

For questions or issues, please refer to:
- Error code design: `docs/design/error-code-design.md`
- Package documentation: `pkg/errors/errno.go`
- Examples: `example/errors/`
