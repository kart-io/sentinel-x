# Authentication Flow Integration Tests

## Overview

This document describes the comprehensive integration tests for the Sentinel-X authentication and authorization system. The tests cover the complete authentication flow from token generation to revocation, as well as middleware integration with RBAC authorization.

## Test Location

File: `pkg/auth/integration_test.go`

Build Tag: `integration` (tests are only run when explicitly enabled with `-tags=integration`)

## Running Integration Tests

### Run all integration tests

```bash
go test -tags=integration -v ./pkg/auth/...
```

### Run specific test

```bash
go test -tags=integration -v -run TestAuthFlow_CompleteTokenLifecycle ./pkg/auth/...
```

### Run with timeout

```bash
go test -tags=integration -v -timeout 30s ./pkg/auth/...
```

## Test Coverage

### 1. Complete Token Lifecycle Test

**Test**: `TestAuthFlow_CompleteTokenLifecycle`

Tests the full lifecycle of a JWT token:

- **Sign**: Creates a new token with custom claims
- **Verify**: Validates the token and extracts claims
- **Refresh**: Creates a new token from an existing valid token
- **Revoke**: Invalidates a token and prevents future use

**Key Assertions**:
- Token is properly generated with correct metadata
- Claims are correctly extracted and validated
- Refreshed tokens preserve original claims
- Revoked tokens fail verification

### 2. Error Scenarios Test

**Test**: `TestAuthFlow_ErrorScenarios`

Tests various error conditions:

- **ExpiredToken**: Tokens that have passed their expiration time
- **InvalidToken**: Malformed, empty, or incorrectly signed tokens
- **RevokedToken**: Tokens that have been explicitly revoked
- **RefreshExpiredBeyondMaxRefresh**: Tokens beyond the refresh window

**Key Assertions**:
- Correct error codes are returned (ErrTokenExpired, ErrInvalidToken, ErrTokenRevoked, ErrSessionExpired)
- Error messages are informative
- Invalid tokens cannot be verified or refreshed

### 3. RBAC Integration Test

**Test**: `TestAuthFlow_WithRBAC`

Tests Role-Based Access Control without middleware:

**Roles Defined**:
- `admin`: Full access to all resources (`*:*`)
- `editor`: Can read, create, and update posts
- `viewer`: Can only read posts

**Test Cases**:
- Admin can perform any action
- Editor can update but not delete
- Viewer can read but not create
- Unknown users have no permissions

### 4. Authentication Middleware Test

**Test**: `TestMiddleware_AuthIntegration`

Tests the HTTP authentication middleware:

**Features Tested**:
- Valid token authentication
- Missing token rejection
- Invalid token rejection
- Skip path functionality (public paths)
- Context injection of claims and subject

**Key Assertions**:
- Valid tokens pass authentication
- Invalid/missing tokens return 401
- Skip paths bypass authentication
- User information is correctly injected into context

### 5. Authorization Middleware Test

**Test**: `TestMiddleware_AuthzIntegration`

Tests the HTTP authorization middleware with RBAC:

**Features Tested**:
- Permission checking based on HTTP method
- Resource and action extraction from request
- Skip path functionality
- Denial of unauthorized requests

**HTTP Method Mapping**:
- GET → read
- POST → create
- PUT/PATCH → update
- DELETE → delete

**Key Assertions**:
- Users with permissions can access resources
- Users without permissions receive 403
- Skip paths bypass authorization
- Correct error logging for denied access

### 6. Full Authentication Flow Test

**Test**: `TestMiddleware_FullAuthFlow`

Tests the complete middleware chain: Auth → Authz → Handler

**Scenarios**:
- Admin accessing any resource
- Viewer reading posts (allowed)
- Viewer deleting posts (denied)

**Key Assertions**:
- Request passes through both middlewares correctly
- Authorization is checked only after authentication succeeds
- Proper HTTP status codes are returned

### 7. Concurrent Access Test

**Test**: `TestConcurrentAccess`

Tests thread safety of authentication components:

**Concurrent Operations**:
- Token generation (50 goroutines × 10 operations)
- Token verification (50 goroutines × 10 operations)
- RBAC operations (role assignment and authorization)
- Token revocation (50 concurrent revocations)

**Key Assertions**:
- No race conditions occur
- All operations complete successfully
- Revoked tokens are properly tracked

## Test Configuration

### Test Constants

```go
const (
    testKey     = "test-secret-key-at-least-32-chars-long!!"
    testSubject = "user-123"
    testIssuer  = "sentinel-x-test"
)
```

### Test Components

- **JWT Authenticator**: Configured with HS256 signing method
- **Memory Store**: In-memory token revocation store
- **RBAC Authorizer**: In-memory role and permission storage
- **Mock Context**: Minimal transport.Context implementation for testing

## Design Decisions

### 1. Build Tag Usage

Integration tests are separated using build tags to:
- Prevent running during normal unit tests
- Allow selective test execution
- Clearly distinguish integration vs unit tests

### 2. Test Isolation

Each test:
- Creates its own authenticator and authorizer instances
- Uses independent token stores
- Cleans up resources after completion

### 3. Error Validation

Tests verify both:
- Error occurrence (non-nil error)
- Specific error codes (using errors.IsCode)
- Error messages for user-facing scenarios

### 4. Timing Considerations

Tests that depend on token expiration use:
- Short expiration times (100-200ms) for quick execution
- Appropriate sleep durations to ensure expiration
- Separate authenticators to avoid interference

## Mock Implementation

### mockContext

A lightweight implementation of `transport.Context` for testing middleware:

**Features**:
- Thread-safe (uses sync.RWMutex)
- Tracks JSON responses and status codes
- Supports context injection and retrieval
- Minimal overhead for fast test execution

## Common Issues and Solutions

### Issue: Race Conditions in Concurrent Tests

**Solution**: Use proper synchronization primitives (sync.WaitGroup, channels) to coordinate goroutines.

### Issue: Flaky Expiration Tests

**Solution**: Use appropriate sleep durations and check for both expired and session expired errors.

### Issue: Mock Context Complexity

**Solution**: Only implement methods actually used by tests; return empty/nil for unused methods.

## Integration with CI/CD

### Example GitHub Actions Workflow

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run Integration Tests
        run: go test -tags=integration -v -timeout 30s ./pkg/auth/...
```

### Example Makefile Target

```makefile
.PHONY: test-integration
test-integration:
	go test -tags=integration -v -timeout 30s ./pkg/auth/... ./pkg/authz/... ./pkg/middleware/...
```

## Best Practices

1. **Always use build tags** for integration tests to separate them from unit tests
2. **Test both success and failure paths** for comprehensive coverage
3. **Use table-driven tests** for testing multiple scenarios
4. **Clean up resources** (close stores, clear RBAC state) after tests
5. **Verify error codes** in addition to error presence
6. **Test concurrent access** to ensure thread safety
7. **Use realistic test data** (proper key lengths, valid JWT formats)

## Future Enhancements

Potential areas for expansion:

1. **Database-backed stores**: Test with real Redis/PostgreSQL
2. **Token refresh chains**: Test multiple refresh operations
3. **Rate limiting**: Test authentication rate limits
4. **Custom claims validation**: Test complex claim structures
5. **OAuth2 integration**: Test third-party authentication providers
6. **gRPC interceptors**: Test gRPC authentication/authorization
7. **Performance benchmarks**: Measure throughput and latency

## References

- [JWT Specification](https://tools.ietf.org/html/rfc7519)
- [RBAC Concepts](https://en.wikipedia.org/wiki/Role-based_access_control)
- [Go Testing Best Practices](https://golang.org/doc/code.html#Testing)
- [Sentinel-X Auth/Authz Design](../../docs/design/auth-authz.md)
