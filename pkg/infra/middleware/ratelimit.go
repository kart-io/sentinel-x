package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
	"github.com/redis/go-redis/v9"
)

// RateLimiter defines the interface for rate limiting implementations.
type RateLimiter interface {
	// Allow checks if a request with the given key is allowed.
	// Returns true if allowed, false if rate limit exceeded.
	Allow(ctx context.Context, key string) (bool, error)

	// Reset resets the rate limit counter for the given key.
	Reset(ctx context.Context, key string) error
}

// RateLimitConfig defines the configuration for rate limiting middleware.
type RateLimitConfig struct {
	// Limit is the maximum number of requests allowed within the time window.
	// Default: 100
	Limit int

	// Window is the time window duration for rate limiting.
	// Default: 1 minute
	Window time.Duration

	// KeyFunc is a function to extract the rate limit key from the context.
	// Default: uses client IP address
	KeyFunc func(c transport.Context) string

	// SkipPaths is a list of paths to skip rate limiting.
	SkipPaths []string

	// OnLimitReached is called when rate limit is exceeded.
	// Can be used for custom logging or alerting.
	OnLimitReached func(c transport.Context)

	// Limiter is the rate limiter implementation to use.
	// If nil, a memory-based limiter will be created.
	Limiter RateLimiter

	// TrustedProxies is a list of trusted proxy IP addresses or CIDR ranges.
	// When empty, proxy headers (X-Forwarded-For, X-Real-IP) are not trusted.
	// Example: []string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12"}
	// Default: empty (do not trust proxy headers)
	TrustedProxies []string

	// TrustProxyHeaders controls whether to trust proxy headers for IP extraction.
	// Even if true, headers are only trusted when requests come from TrustedProxies.
	// Default: false (do not trust proxy headers)
	TrustProxyHeaders bool
}

// DefaultRateLimitConfig is the default rate limit configuration.
var DefaultRateLimitConfig = RateLimitConfig{
	Limit:             100,
	Window:            1 * time.Minute,
	KeyFunc:           nil, // Will use defaultKeyFunc
	SkipPaths:         []string{},
	OnLimitReached:    nil,
	Limiter:           nil,        // Will create memory limiter
	TrustedProxies:    []string{}, // Empty by default - do not trust proxy headers
	TrustProxyHeaders: false,      // Do not trust proxy headers by default
}

// RateLimit returns a rate limiting middleware with default configuration.
func RateLimit() transport.MiddlewareFunc {
	return RateLimitWithConfig(DefaultRateLimitConfig)
}

// RateLimitWithConfig returns a rate limiting middleware with custom configuration.
func RateLimitWithConfig(config RateLimitConfig) transport.MiddlewareFunc {
	// Validate and set defaults
	config = validateConfig(config)

	// Build skip paths map for fast lookup
	skipPaths := buildSkipPathsMap(config.SkipPaths)

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			req := c.HTTPRequest()

			// Skip rate limiting for configured paths
			if shouldSkipPath(req.URL.Path, skipPaths) {
				next(c)
				return
			}

			// Extract rate limit key
			key := extractKey(c, config.KeyFunc, config)

			// Check rate limit
			allowed, err := checkRateLimit(c.Request(), config.Limiter, key)
			if err != nil {
				// Log error but allow request to proceed
				logRateLimitError(err, key)
				next(c)
				return
			}

			if !allowed {
				// Rate limit exceeded
				handleRateLimitExceeded(c, config.OnLimitReached)
				return
			}

			// Request allowed, proceed
			next(c)
		}
	}
}

// ============================================================================
// Configuration Validation
// ============================================================================

// validateConfig validates and sets default values for the configuration.
func validateConfig(config RateLimitConfig) RateLimitConfig {
	if config.Limit <= 0 {
		config.Limit = DefaultRateLimitConfig.Limit
	}

	if config.Window <= 0 {
		config.Window = DefaultRateLimitConfig.Window
	}

	if config.KeyFunc == nil {
		// Use closure to capture config for IP extraction
		config.KeyFunc = func(c transport.Context) string {
			return extractClientIP(c, config)
		}
	}

	if config.SkipPaths == nil {
		config.SkipPaths = []string{}
	}

	if config.Limiter == nil {
		config.Limiter = NewMemoryRateLimiter(config.Limit, config.Window)
	}

	return config
}

// ============================================================================
// Key Extraction
// ============================================================================

// extractKey extracts the rate limit key using the configured KeyFunc.
// Falls back to RemoteAddr if the key function returns empty string.
func extractKey(c transport.Context, keyFunc func(c transport.Context) string, config RateLimitConfig) string {
	key := keyFunc(c)
	if key == "" {
		// Fallback to remote IP if key function returns empty string
		req := c.HTTPRequest()
		key = getRemoteIP(req)
	}
	return key
}

// extractClientIP extracts the real client IP from the request.
// It only trusts proxy headers (X-Forwarded-For, X-Real-IP) when:
// 1. TrustProxyHeaders is enabled in config
// 2. The request comes from a trusted proxy IP/CIDR
// This prevents IP spoofing attacks via forged headers.
func extractClientIP(c transport.Context, config RateLimitConfig) string {
	req := c.HTTPRequest()
	remoteIP := getRemoteIP(req)

	// Only trust proxy headers if configured and request is from trusted proxy
	if config.TrustProxyHeaders && isTrustedProxy(remoteIP, config.TrustedProxies) {
		// Check X-Forwarded-For header (most common)
		if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
			// X-Forwarded-For can contain multiple IPs (client, proxy1, proxy2, ...)
			// Use the first IP which should be the original client
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				ip := strings.TrimSpace(ips[0])
				if isValidIP(ip) {
					return ip
				}
			}
		}

		// Check X-Real-IP header (alternative to X-Forwarded-For)
		if xri := req.Header.Get("X-Real-IP"); xri != "" {
			xri = strings.TrimSpace(xri)
			if isValidIP(xri) {
				return xri
			}
		}
	}

	// Fall back to remote address (directly connected IP)
	// This is always safe as it cannot be spoofed
	return remoteIP
}

// getRemoteIP extracts the IP address from http.Request.RemoteAddr.
// RemoteAddr is in the form "IP:port", so we need to split it.
func getRemoteIP(req *http.Request) string {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		// If split fails, return the whole RemoteAddr
		return req.RemoteAddr
	}
	return ip
}

// isTrustedProxy checks if the given IP is in the list of trusted proxies.
// Supports both individual IPs and CIDR ranges.
func isTrustedProxy(ip string, trustedCIDRs []string) bool {
	// If no trusted proxies configured, don't trust any proxy
	if len(trustedCIDRs) == 0 {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		// Invalid IP, don't trust it
		return false
	}

	for _, cidr := range trustedCIDRs {
		// Support both single IP addresses and CIDR notation
		if !strings.Contains(cidr, "/") {
			// Single IP address - add /32 or /128 for exact match
			if cidr == ip {
				return true
			}
			continue
		}

		// CIDR range - parse and check if IP is in range
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			// Invalid CIDR, skip it
			logger.Warnw("invalid CIDR in trusted proxies",
				"cidr", cidr,
				"error", err.Error(),
			)
			continue
		}

		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// isValidIP validates that the given string is a valid IP address.
// This prevents injection of invalid data into rate limiting keys.
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ============================================================================
// Path Skipping
// ============================================================================

// buildSkipPathsMap builds a map for fast path lookup.
func buildSkipPathsMap(paths []string) map[string]bool {
	skipMap := make(map[string]bool, len(paths))
	for _, path := range paths {
		skipMap[path] = true
	}
	return skipMap
}

// shouldSkipPath checks if the given path should skip rate limiting.
func shouldSkipPath(path string, skipPaths map[string]bool) bool {
	return skipPaths[path]
}

// ============================================================================
// Rate Limit Checking
// ============================================================================

// checkRateLimit checks if a request is allowed by the rate limiter.
func checkRateLimit(ctx context.Context, limiter RateLimiter, key string) (bool, error) {
	return limiter.Allow(ctx, key)
}

// ============================================================================
// Response Handling
// ============================================================================

// handleRateLimitExceeded handles the case when rate limit is exceeded.
func handleRateLimitExceeded(c transport.Context, onLimitReached func(c transport.Context)) {
	// Call custom callback if provided
	if onLimitReached != nil {
		onLimitReached(c)
	}

	// Return rate limit error
	response.Fail(c, errors.ErrRateLimitExceeded)
}

// ============================================================================
// Error Logging
// ============================================================================

// logRateLimitError logs rate limiter errors.
func logRateLimitError(err error, key string) {
	logger.Errorw("rate limiter error",
		"error", err.Error(),
		"key", key,
	)
}

// ============================================================================
// Memory Rate Limiter Implementation
// ============================================================================

// MemoryRateLimiter implements rate limiting using in-memory storage.
// It uses a sliding window algorithm with bucketing for accurate rate limiting.
type MemoryRateLimiter struct {
	limit  int
	window time.Duration
	store  *sync.Map
	// cleanup goroutine cancellation
	stopCleanup chan struct{}
	cleanupOnce sync.Once
}

// rateLimitEntry stores rate limit data for a single key.
type rateLimitEntry struct {
	requests  []time.Time
	mu        sync.Mutex
	lastCheck time.Time
}

// NewMemoryRateLimiter creates a new memory-based rate limiter.
func NewMemoryRateLimiter(limit int, window time.Duration) *MemoryRateLimiter {
	limiter := &MemoryRateLimiter{
		limit:       limit,
		window:      window,
		store:       &sync.Map{},
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	go limiter.cleanupExpiredEntries()

	return limiter
}

// Allow checks if a request with the given key is allowed.
func (m *MemoryRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()

	// Get or create entry
	value, _ := m.store.LoadOrStore(key, &rateLimitEntry{
		requests:  make([]time.Time, 0, m.limit),
		lastCheck: now,
	})

	entry := value.(*rateLimitEntry)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Update last check time
	entry.lastCheck = now

	// Remove expired requests (outside the window)
	cutoff := now.Add(-m.window)
	entry.requests = filterExpiredRequests(entry.requests, cutoff)

	// Check if limit is exceeded
	if len(entry.requests) >= m.limit {
		return false, nil
	}

	// Add current request
	entry.requests = append(entry.requests, now)

	return true, nil
}

// Reset resets the rate limit counter for the given key.
func (m *MemoryRateLimiter) Reset(ctx context.Context, key string) error {
	m.store.Delete(key)
	return nil
}

// Stop stops the cleanup goroutine.
func (m *MemoryRateLimiter) Stop() {
	m.cleanupOnce.Do(func() {
		close(m.stopCleanup)
	})
}

// cleanupExpiredEntries periodically removes expired entries from memory.
func (m *MemoryRateLimiter) cleanupExpiredEntries() {
	ticker := time.NewTicker(m.window)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performCleanup()
		case <-m.stopCleanup:
			return
		}
	}
}

// performCleanup removes entries that haven't been accessed recently.
func (m *MemoryRateLimiter) performCleanup() {
	now := time.Now()
	threshold := now.Add(-m.window * 2) // Keep entries for 2x window duration

	m.store.Range(func(key, value interface{}) bool {
		entry := value.(*rateLimitEntry)
		entry.mu.Lock()
		lastCheck := entry.lastCheck
		entry.mu.Unlock()

		if lastCheck.Before(threshold) {
			m.store.Delete(key)
		}
		return true
	})
}

// filterExpiredRequests removes timestamps that are outside the time window.
func filterExpiredRequests(requests []time.Time, cutoff time.Time) []time.Time {
	// Find the first non-expired request
	validIdx := 0
	for i, t := range requests {
		if t.After(cutoff) {
			validIdx = i
			break
		}
	}

	// Return slice starting from first valid request
	if validIdx > 0 {
		return requests[validIdx:]
	}
	return requests
}

// ============================================================================
// Redis Rate Limiter Implementation
// ============================================================================

// RedisRateLimiter implements rate limiting using Redis.
// It uses Redis sorted sets for accurate sliding window rate limiting.
type RedisRateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	prefix string
}

// NewRedisRateLimiter creates a new Redis-based rate limiter.
func NewRedisRateLimiter(client *redis.Client, limit int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		limit:  limit,
		window: window,
		prefix: "ratelimit:",
	}
}

// Allow checks if a request with the given key is allowed using Redis.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	redisKey := r.prefix + key

	// Use Redis sorted set for sliding window
	// Score is timestamp, member is unique request ID
	pipe := r.client.Pipeline()

	// Remove old entries outside the window
	minScore := float64(now.Add(-r.window).UnixNano())
	pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%.0f", minScore))

	// Count current entries
	countCmd := pipe.ZCard(ctx, redisKey)

	// Add current request
	score := float64(now.UnixNano())
	member := fmt.Sprintf("%d", now.UnixNano())
	pipe.ZAdd(ctx, redisKey, redis.Z{Score: score, Member: member})

	// Set expiration
	pipe.Expire(ctx, redisKey, r.window*2)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis pipeline error: %w", err)
	}

	// Check if limit is exceeded
	count := countCmd.Val()
	if count >= int64(r.limit) {
		return false, nil
	}

	return true, nil
}

// Reset resets the rate limit counter for the given key in Redis.
func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := r.prefix + key
	return r.client.Del(ctx, redisKey).Err()
}
