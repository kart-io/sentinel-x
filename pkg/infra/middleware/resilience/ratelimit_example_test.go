package resilience_test

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/redis/go-redis/v9"
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

// Example_rateLimitWithConfig demonstrates custom rate limit configuration.
func Example_rateLimitWithConfig() {
	// Configure rate limiting
	config := resilience.RateLimitConfig{
		Limit:  50,              // 50 requests
		Window: 1 * time.Minute, // per minute
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
	}

	rateLimitMiddleware := resilience.RateLimitWithConfig(config)
	_ = rateLimitMiddleware
	// Output:
}

// Example_rateLimitByUserID demonstrates rate limiting by user ID.
func Example_rateLimitByUserID() {
	config := resilience.RateLimitConfig{
		Limit:  10,
		Window: 1 * time.Minute,
		// Custom key function to rate limit by user ID
		KeyFunc: func(c transport.Context) string {
			userID := c.Header("X-User-ID")
			if userID == "" {
				// Fallback to IP if no user ID
				return extractClientIP(c)
			}
			return fmt.Sprintf("user:%s", userID)
		},
	}

	rateLimitMiddleware := resilience.RateLimitWithConfig(config)
	_ = rateLimitMiddleware
	// Output:
}

// Example_rateLimitWithCallback demonstrates rate limit callback.
func Example_rateLimitWithCallback() {
	config := resilience.RateLimitConfig{
		Limit:  100,
		Window: 1 * time.Minute,
		OnLimitReached: func(c transport.Context) {
			// Log or alert when rate limit is exceeded
			req := c.HTTPRequest()
			fmt.Printf("Rate limit exceeded for %s\n", req.RemoteAddr)
		},
	}

	rateLimitMiddleware := resilience.RateLimitWithConfig(config)
	_ = rateLimitMiddleware
	// Output:
}

// Example_memoryRateLimiter demonstrates memory-based rate limiting.
func Example_memoryRateLimiter() {
	// Create memory-based rate limiter
	limiter := resilience.NewMemoryRateLimiter(100, 1*time.Minute)
	defer limiter.Stop()

	config := resilience.RateLimitConfig{
		Limit:   100,
		Window:  1 * time.Minute,
		Limiter: limiter,
	}

	rateLimitMiddleware := resilience.RateLimitWithConfig(config)
	_ = rateLimitMiddleware
	// Output:
}

// Example_redisRateLimiter demonstrates Redis-based rate limiting.
func Example_redisRateLimiter() {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Create Redis-based rate limiter
	limiter := resilience.NewRedisRateLimiter(redisClient, 100, 1*time.Minute)

	config := resilience.RateLimitConfig{
		Limit:   100,
		Window:  1 * time.Minute,
		Limiter: limiter,
	}

	rateLimitMiddleware := resilience.RateLimitWithConfig(config)
	_ = rateLimitMiddleware
	// Output:
}

// Example_rateLimitByAPIKey demonstrates rate limiting by API key.
func Example_rateLimitByAPIKey() {
	config := resilience.RateLimitConfig{
		Limit:  1000, // Higher limit for API keys
		Window: 1 * time.Minute,
		KeyFunc: func(c transport.Context) string {
			apiKey := c.Header("X-API-Key")
			if apiKey == "" {
				// No API key, use IP and lower limit
				return fmt.Sprintf("ip:%s", extractClientIP(c))
			}
			return fmt.Sprintf("apikey:%s", apiKey)
		},
	}

	rateLimitMiddleware := resilience.RateLimitWithConfig(config)
	_ = rateLimitMiddleware
	// Output:
}

// Example_rateLimitPerEndpoint demonstrates different limits per endpoint.
func Example_rateLimitPerEndpoint() {
	config := resilience.RateLimitConfig{
		Limit:  50,
		Window: 1 * time.Minute,
		KeyFunc: func(c transport.Context) string {
			req := c.HTTPRequest()
			// Combine IP and path for per-endpoint rate limiting
			return fmt.Sprintf("%s:%s", extractClientIP(c), req.URL.Path)
		},
	}

	rateLimitMiddleware := resilience.RateLimitWithConfig(config)
	_ = rateLimitMiddleware
	// Output:
}

// customLimiter is a custom rate limiter implementation
type customLimiter struct{}

func (c *customLimiter) Allow(_ context.Context, _ string) (bool, error) {
	// Custom rate limiting logic
	return true, nil
}

func (c *customLimiter) Reset(_ context.Context, _ string) error {
	// Custom reset logic
	return nil
}

// Example_customRateLimiter demonstrates implementing a custom rate limiter.
func Example_customRateLimiter() {
	// Create instance of custom limiter
	limiter := &customLimiter{}

	config := resilience.RateLimitConfig{
		Limit:   100,
		Window:  1 * time.Minute,
		Limiter: limiter,
	}

	rateLimitMiddleware := resilience.RateLimitWithConfig(config)
	_ = rateLimitMiddleware
	// Output:
}

// Helper function to extract client IP
func extractClientIP(c transport.Context) string {
	req := c.HTTPRequest()
	xff := req.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}
	return req.RemoteAddr
}
