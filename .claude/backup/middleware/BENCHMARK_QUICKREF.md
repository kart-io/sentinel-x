# Middleware Benchmark Quick Reference

## Quick Start

```bash
# Run all benchmarks
go test -bench=. -benchmem ./pkg/infra/middleware/

# Run specific middleware benchmark
go test -bench=BenchmarkLoggerMiddleware -benchmem ./pkg/infra/middleware/

# Compare before/after changes
go test -bench=. -benchmem ./pkg/infra/middleware/ > before.txt
# make changes...
go test -bench=. -benchmem ./pkg/infra/middleware/ > after.txt
benchcmp before.txt after.txt
```

## Available Benchmarks (21 total)

### Individual Middleware (12)

1. `BenchmarkLoggerMiddleware` - Logger performance
2. `BenchmarkLoggerMiddlewareWithSkip` - Logger with path skip
3. `BenchmarkRecoveryMiddleware` - Recovery normal operation
4. `BenchmarkRecoveryMiddlewareWithPanic` - Recovery panic handling
5. `BenchmarkRequestIDMiddleware` - RequestID generation
6. `BenchmarkRequestIDMiddlewareWithExisting` - RequestID with existing header
7. `BenchmarkRateLimitMiddleware` - Rate limiting
8. `BenchmarkRateLimitMiddlewareWithSkip` - Rate limit with path skip
9. `BenchmarkSecurityHeadersMiddleware` - Security headers
10. `BenchmarkSecurityHeadersMiddlewareWithHSTS` - Security headers with HSTS
11. `BenchmarkTimeoutMiddleware` - Timeout control
12. `BenchmarkTimeoutMiddlewareWithSkip` - Timeout with path skip

### Middleware Chains (4)

13. `BenchmarkMiddlewareChain` - Full middleware chain
14. `BenchmarkMiddlewareChainMinimal` - Minimal chain (3 middlewares)
15. `BenchmarkMiddlewareChainProduction` - Production-optimized chain
16. `BenchmarkMiddlewareChainConcurrent` - Concurrent load test

### Utility Functions (2)

17. `BenchmarkGenerateRequestID` - Request ID generation only
18. `BenchmarkMemoryRateLimiterAllow` - Rate limiter core logic

### Special Tests (3)

19. `BenchmarkTimeoutMiddlewareWithDelay` - Timeout with simulated delay
20. `BenchmarkMiddlewareMemoryAllocation` - Memory allocation patterns
21. `BenchmarkMiddlewareChainWithBody` - Request with JSON body

## Performance Targets

### Individual Middleware

| Middleware | Target ns/op | Target B/op | Target allocs/op |
|------------|--------------|-------------|------------------|
| Logger (skip) | < 2,000 | < 6,000 | < 20 |
| Recovery | < 2,000 | < 6,000 | < 20 |
| RequestID | < 4,000 | < 8,000 | < 30 |
| RateLimit | < 6,000 | < 32,000 | < 25 |
| SecurityHeaders | < 4,000 | < 7,000 | < 30 |
| Timeout | < 5,000 | < 8,000 | < 30 |

### Middleware Chains

| Chain | Target ns/op | Target B/op | Target allocs/op |
|-------|--------------|-------------|------------------|
| Minimal (3) | < 10,000 | < 28,000 | < 80 |
| Full (6) | < 15,000 | < 55,000 | < 100 |
| Production | < 13,000 | < 30,000 | < 85 |

## Reading Benchmark Results

```
BenchmarkLoggerMiddleware-28    1000    1526 ns/op    5864 B/op    18 allocs/op
```

- `BenchmarkLoggerMiddleware`: Test name
- `-28`: Number of CPUs used (GOMAXPROCS)
- `1000`: Number of iterations run
- `1526 ns/op`: Nanoseconds per operation (lower is better)
- `5864 B/op`: Bytes allocated per operation (lower is better)
- `18 allocs/op`: Number of allocations per operation (lower is better)

## Common Commands

### Run Specific Benchmark

```bash
# Single middleware
go test -bench=BenchmarkLoggerMiddleware$ -benchmem ./pkg/infra/middleware/

# Multiple benchmarks with pattern
go test -bench=BenchmarkMiddlewareChain -benchmem ./pkg/infra/middleware/
```

### Adjust Iterations

```bash
# Run for specific iterations
go test -bench=. -benchmem -benchtime=1000x ./pkg/infra/middleware/

# Run for specific duration
go test -bench=. -benchmem -benchtime=10s ./pkg/infra/middleware/
```

### CPU Profiling

```bash
# Generate CPU profile
go test -bench=BenchmarkMiddlewareChain -cpuprofile=cpu.prof ./pkg/infra/middleware/

# Analyze profile
go tool pprof cpu.prof
```

### Memory Profiling

```bash
# Generate memory profile
go test -bench=BenchmarkMiddlewareChain -memprofile=mem.prof ./pkg/infra/middleware/

# Analyze profile
go tool pprof mem.prof
```

### Trace Analysis

```bash
# Generate trace
go test -bench=BenchmarkMiddlewareChain -trace=trace.out ./pkg/infra/middleware/

# View trace
go tool trace trace.out
```

## Performance Regression Detection

### Compare Performance

```bash
# Benchmark before changes
go test -bench=. -benchmem ./pkg/infra/middleware/ | tee baseline.txt

# Make changes...

# Benchmark after changes
go test -bench=. -benchmem ./pkg/infra/middleware/ | tee current.txt

# Compare (using benchstat)
benchstat baseline.txt current.txt
```

### Install benchstat

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

### Acceptable Regression

- **Performance**: < 5% regression in ns/op
- **Memory**: < 10% increase in B/op
- **Allocations**: No increase in allocs/op (strict)

## CI/CD Integration

### Example CI Script

```bash
#!/bin/bash
set -e

# Run benchmarks
go test -bench=. -benchmem ./pkg/infra/middleware/ > current.txt

# Compare with baseline (if exists)
if [ -f baseline.txt ]; then
    benchstat baseline.txt current.txt

    # Check for significant regression (implement custom logic)
    # ... regression detection ...
fi

# Update baseline
cp current.txt baseline.txt
```

### GitHub Actions Example

```yaml
name: Benchmark
on: [push, pull_request]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.25'

      - name: Run benchmarks
        run: |
          go test -bench=. -benchmem ./pkg/infra/middleware/ > benchmark.txt
          cat benchmark.txt

      - name: Upload results
        uses: actions/upload-artifact@v2
        with:
          name: benchmark-results
          path: benchmark.txt
```

## Troubleshooting

### Unstable Results

```bash
# Increase iterations
go test -bench=. -benchmem -benchtime=10000x ./pkg/infra/middleware/

# Reduce system interference
sudo nice -n -20 go test -bench=. -benchmem ./pkg/infra/middleware/
```

### High Variance

```bash
# Run multiple times and average
for i in {1..5}; do
    go test -bench=BenchmarkLoggerMiddleware -benchmem ./pkg/infra/middleware/
done
```

### Memory Issues

```bash
# Increase available memory
GOGC=100 go test -bench=. -benchmem ./pkg/infra/middleware/

# Check for memory leaks
go test -bench=BenchmarkRateLimitMiddleware -memprofile=mem.prof ./pkg/infra/middleware/
go tool pprof -alloc_space mem.prof
```

## Documentation References

- Full Documentation: [BENCHMARK.md](./BENCHMARK.md)
- Performance Report: [PERFORMANCE_REPORT.md](./PERFORMANCE_REPORT.md)
- Test Code: [benchmark_test.go](./benchmark_test.go)

## Performance Goals Summary

### Current Performance (as of 2025-12-10)

- **Minimal Chain**: 9,463 ns/op (0.0095 ms)
- **Full Chain**: 13,845 ns/op (0.0138 ms)
- **Production Chain**: 12,118 ns/op (0.0121 ms)

### Target QPS (Single Core)

- **Minimal Chain**: ~105,000 QPS
- **Full Chain**: ~72,000 QPS
- **Production Chain**: ~82,000 QPS

### Latency Budget

- **Middleware Overhead**: < 0.015 ms (< 0.015% of 100ms budget)
- **Total API Latency**: < 100 ms (P99)

## Contact & Support

For questions about benchmarks:

1. Check [BENCHMARK.md](./BENCHMARK.md) for detailed usage
2. Review [PERFORMANCE_REPORT.md](./PERFORMANCE_REPORT.md) for analysis
3. Open an issue with benchmark results and questions
