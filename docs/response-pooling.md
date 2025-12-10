# Response Object Pooling

## Overview

This document describes the Response object pooling implementation designed to reduce memory allocations and GC pressure in high-throughput scenarios (10K+ RPS).

## Implementation Details

### Core Components

1. **sync.Pool**: Thread-safe object pool for Response instances
2. **Acquire()**: Retrieves a Response from the pool
3. **Release()**: Returns a Response to the pool after resetting fields

### Architecture

```go
var responsePool = sync.Pool{
    New: func() interface{} {
        return &Response{}
    },
}

// Acquire retrieves a Response from the pool
func Acquire() *Response {
    return responsePool.Get().(*Response)
}

// Release returns a Response to the pool
func Release(r *Response) {
    if r == nil {
        return
    }
    // Reset all fields to prevent data leakage
    r.Code = 0
    r.HTTPCode = 0
    r.Message = ""
    r.Data = nil
    r.RequestID = ""
    r.Timestamp = 0
    responsePool.Put(r)
}
```

## Usage

### Automatic Release (Recommended)

The `Writer` type automatically manages pooling for you:

```go
// response.NewWriter automatically handles pooling
w := response.NewWriter(ctx)
w.OK(data)  // Response is automatically released after JSON encoding
```

### Manual Pooling (Advanced)

When creating responses directly:

```go
// Acquire from pool
resp := response.Acquire()

// Use the response
resp.Code = 0
resp.Message = "success"
resp.Data = myData

// IMPORTANT: Release back to pool
defer response.Release(resp)

// Write to client
ctx.JSON(200, resp)
```

### Using Helper Functions

All helper functions now use pooling internally:

```go
// These automatically acquire from pool
resp := response.Success(data)
defer response.Release(resp)

resp := response.Err(errCode)
defer response.Release(resp)

resp := response.Page(list, total, page, pageSize)
defer response.Release(resp)
```

## Performance Characteristics

### Benchmark Results

Based on benchmark testing on Apple M4 Pro:

#### Concurrent Access (16 goroutines)
```
BenchmarkConcurrentPool/Concurrent_16-14    0.8038 ns/op    0 B/op    0 allocs/op
```

#### Success Response Creation
```
BenchmarkSuccessResponse/Success-14         7.119 ns/op     0 B/op    0 allocs/op
```

### Key Benefits

1. **Zero Allocations**: In steady state, no new Response objects are allocated
2. **Thread-Safe**: sync.Pool is safe for concurrent use
3. **GC Pressure Reduction**: Significantly reduces garbage collection overhead
4. **Performance**: Minimal overhead compared to direct allocation

## Thread Safety

### Pool Safety
- `sync.Pool` is thread-safe by design
- Multiple goroutines can safely call `Acquire()` and `Release()` concurrently
- No locks required in application code

### Data Isolation
- `Release()` resets all fields to zero values
- Prevents data leakage between requests
- Safe for reuse in different goroutines

## Best Practices

### DO

✅ **Use Writer methods** - Automatic pooling management
```go
response.NewWriter(ctx).OK(data)
```

✅ **Release after use** - When using manual pooling
```go
resp := response.Acquire()
defer response.Release(resp)
```

✅ **Check for nil** - Before releasing
```go
if resp != nil {
    response.Release(resp)
}
```

### DON'T

❌ **Don't hold references** - After release
```go
resp := response.Acquire()
response.Release(resp)
// DON'T use resp here - it may be reused
```

❌ **Don't release twice** - Double release
```go
resp := response.Acquire()
response.Release(resp)
response.Release(resp)  // BUG: Don't do this
```

❌ **Don't modify after JSON encoding** - Race condition risk
```go
ctx.JSON(200, resp)
resp.Message = "changed"  // BUG: May be in use
```

## Migration Guide

### Before (Without Pooling)

```go
func (h *Handler) GetUser(ctx transport.Context) {
    user := h.service.GetUser(id)

    // Direct allocation
    resp := &response.Response{
        Code:    0,
        Message: "success",
        Data:    user,
    }
    ctx.JSON(200, resp)
}
```

### After (With Pooling)

```go
func (h *Handler) GetUser(ctx transport.Context) {
    user := h.service.GetUser(id)

    // Use Writer (automatic pooling)
    response.NewWriter(ctx).OK(user)
}

// OR manual pooling if needed
func (h *Handler) GetUserManual(ctx transport.Context) {
    user := h.service.GetUser(id)

    resp := response.Success(user)
    defer response.Release(resp)

    ctx.JSON(resp.HTTPStatus(), resp)
}
```

## Testing

### Pool Safety Tests

```bash
# Run pool safety tests
go test -v ./pkg/utils/response/ -run TestPoolSafety

# Run concurrent safety tests
go test -v ./pkg/utils/response/ -run TestConcurrentSafety
```

### Benchmarks

```bash
# Basic pool benchmark
go test -bench=BenchmarkResponsePool -benchmem ./pkg/utils/response/

# Concurrent access benchmark
go test -bench=BenchmarkConcurrentPool -benchmem ./pkg/utils/response/

# High throughput simulation
go test -bench=BenchmarkHighThroughput -benchmem ./pkg/utils/response/

# All benchmarks
go test -bench=. -benchmem ./pkg/utils/response/
```

## Implementation Notes

### Memory Management

1. **Pool Behavior**: sync.Pool may discard objects during GC
2. **No Size Limit**: Pool grows as needed under high load
3. **Auto-Scaling**: Pool automatically adjusts to workload

### Field Reset Order

Fields are reset in a specific order to ensure safety:
```go
r.Code = 0          // Integer - fast
r.HTTPCode = 0      // Integer - fast
r.Message = ""      // String - needs GC
r.Data = nil        // Interface - needs GC
r.RequestID = ""    // String - needs GC
r.Timestamp = 0     // Integer - fast
```

### When NOT to Use Pooling

- **Low traffic** (< 100 RPS): Overhead not justified
- **Long-lived responses**: Responses kept in memory for caching
- **Response mutation**: If responses are modified after creation

## Monitoring

### Metrics to Track

1. **Allocation Rate**: Monitor `allocs/op` in benchmarks
2. **GC Frequency**: Reduced with effective pooling
3. **Memory Usage**: Should stabilize under load
4. **Response Time**: P50, P95, P99 latencies

### Performance Indicators

Good pooling performance shows:
- 0 B/op in steady state
- 0 allocs/op for pooled operations
- Stable memory usage under sustained load
- Reduced GC pause times

## Future Improvements

Potential enhancements:
1. Per-CPU pools for better cache locality
2. Size classes for different response types
3. Pool statistics and monitoring hooks
4. Automatic leak detection in development mode

## References

- [Go sync.Pool documentation](https://pkg.go.dev/sync#Pool)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
- Project Location: `/Users/costalong/code/go/src/github.com/kart/sentinel-x/pkg/utils/response/`
