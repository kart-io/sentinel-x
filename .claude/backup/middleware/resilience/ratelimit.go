package resilience

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/internal/pathutil"
	"github.com/kart-io/sentinel-x/pkg/infra/pool"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
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

// RateLimit returns a rate limiting middleware with default configuration.
func RateLimit() transport.MiddlewareFunc {
	opts := mwopts.NewRateLimitOptions()
	limiter := NewMemoryRateLimiter(opts.Limit, opts.GetWindow())
	return RateLimitWithOptions(*opts, limiter)
}

// RateLimitWithOptions returns a rate limiting middleware with custom options.
// 这是推荐的 API，使用纯配置选项和运行时依赖注入。
//
// 参数：
//   - opts: RateLimit 配置选项（纯配置，可 JSON 序列化）
//   - limiter: 限流器实现（运行时依赖注入）
//
// 示例：
//
//	opts := mwopts.NewRateLimitOptions()
//	opts.Limit = 200
//	limiter := resilience.NewMemoryRateLimiter(opts.Limit, opts.GetWindow())
//	middleware.RateLimitWithOptions(*opts, limiter)
func RateLimitWithOptions(opts mwopts.RateLimitOptions, limiter RateLimiter) transport.MiddlewareFunc {
	// 创建路径匹配器
	pathMatcher := pathutil.NewPathMatcher(opts.SkipPaths, nil)

	// 创建 key extraction 函数
	keyFunc := func(c transport.Context) string {
		return extractClientIP(c, opts)
	}

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			req := c.HTTPRequest()

			// Skip rate limiting for configured paths
			if pathMatcher(req.URL.Path) {
				next(c)
				return
			}

			// Extract rate limit key
			key := extractKey(c, keyFunc)

			// Check rate limit
			allowed, err := checkRateLimit(c.Request(), limiter, key)
			if err != nil {
				// Log error but allow request to proceed
				logRateLimitError(err, key)
				next(c)
				return
			}

			if !allowed {
				// Rate limit exceeded
				handleRateLimitExceeded(c)
				return
			}

			// Request allowed, proceed
			next(c)
		}
	}
}

// ============================================================================
// Key Extraction
// ============================================================================

// extractKey extracts the rate limit key using the configured KeyFunc.
// Falls back to RemoteAddr if the key function returns empty string.
func extractKey(c transport.Context, keyFunc func(c transport.Context) string) string {
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
// 1. TrustProxyHeaders is enabled in opts
// 2. The request comes from a trusted proxy IP/CIDR
// This prevents IP spoofing attacks via forged headers.
func extractClientIP(c transport.Context, opts mwopts.RateLimitOptions) string {
	req := c.HTTPRequest()
	remoteIP := getRemoteIP(req)

	// Only trust proxy headers if configured and request is from trusted proxy
	if opts.TrustProxyHeaders && isTrustedProxy(remoteIP, opts.TrustedProxies) {
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
func handleRateLimitExceeded(c transport.Context) {
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

// rateLimitEntry 存储单个键的限流数据（固定窗口计数器方案）
type rateLimitEntry struct {
	count       int        // 当前窗口内的请求计数
	windowStart time.Time  // 窗口起始时间
	mu          sync.Mutex // 保护并发访问
}

// NewMemoryRateLimiter 创建基于内存的限流器
func NewMemoryRateLimiter(limit int, window time.Duration) *MemoryRateLimiter {
	limiter := &MemoryRateLimiter{
		limit:       limit,
		window:      window,
		store:       &sync.Map{},
		stopCleanup: make(chan struct{}),
	}

	// 使用 ants 池提交清理任务，而非直接创建 goroutine
	if err := pool.SubmitToType(pool.BackgroundPool, func() {
		limiter.cleanupExpiredEntries()
	}); err != nil {
		// 池不可用时降级为直接启动 goroutine
		logger.Warnw("failed to submit cleanup task to pool, fallback to goroutine",
			"error", err.Error(),
		)
		go limiter.cleanupExpiredEntries()
	}

	return limiter
}

// Allow 检查给定键的请求是否被允许（固定窗口计数器方案）
func (m *MemoryRateLimiter) Allow(_ context.Context, key string) (bool, error) {
	now := time.Now()

	// 获取或创建限流条目
	value, _ := m.store.LoadOrStore(key, &rateLimitEntry{
		count:       0,
		windowStart: now,
	})

	entry := value.(*rateLimitEntry)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	// 检查窗口是否已过期，如果过期则重置计数器
	if now.Sub(entry.windowStart) >= m.window {
		entry.windowStart = now
		entry.count = 0
	}

	// 检查是否超过限制
	if entry.count >= m.limit {
		return false, nil
	}

	// 计数器加一
	entry.count++

	return true, nil
}

// Reset resets the rate limit counter for the given key.
func (m *MemoryRateLimiter) Reset(_ context.Context, key string) error {
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

// performCleanup 清理过期的限流条目
func (m *MemoryRateLimiter) performCleanup() {
	now := time.Now()

	m.store.Range(func(key, value interface{}) bool {
		entry := value.(*rateLimitEntry)
		entry.mu.Lock()
		windowStart := entry.windowStart
		entry.mu.Unlock()

		// 删除超过两个窗口周期未活动的条目
		if now.Sub(windowStart) > m.window*2 {
			m.store.Delete(key)
		}
		return true
	})
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
