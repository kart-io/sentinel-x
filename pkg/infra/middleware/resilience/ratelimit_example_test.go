package resilience_test

import (
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// Example_rateLimit demonstrates basic rate limiting with default configuration.
func Example_rateLimit() {
	// Create middleware with default configuration:
	// - 100 requests per minute
	// - Rate limiting by client IP
	rateLimitMiddleware := resilience.RateLimit()

	// Use with server
	_ = rateLimitMiddleware
	// Output:
}

// Example_rateLimitWithOptions demonstrates custom rate limit configuration.
func Example_rateLimitWithOptions() {
	// Configure rate limiting
	opts := mwopts.RateLimitOptions{
		Limit:  50, // 50 requests
		Window: 60, // per 60 seconds
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
	}

	limiter := resilience.NewMemoryRateLimiter(opts.Limit, time.Duration(opts.Window)*time.Second)
	defer limiter.Stop()

	rateLimitMiddleware := resilience.RateLimitWithOptions(opts, limiter)
	_ = rateLimitMiddleware
	// Output:
}

// Example_memoryRateLimiter demonstrates memory-based rate limiting.
func Example_memoryRateLimiter() {
	// Create memory-based rate limiter
	limiter := resilience.NewMemoryRateLimiter(100, 1*time.Minute)
	defer limiter.Stop()

	opts := mwopts.RateLimitOptions{
		Limit:  100,
		Window: 60,
	}

	rateLimitMiddleware := resilience.RateLimitWithOptions(opts, limiter)
	_ = rateLimitMiddleware
	// Output:
}
