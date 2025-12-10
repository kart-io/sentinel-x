// Package authz provides caching support for authorization decisions.
package authz

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

// CachedAuthorizer wraps an Authorizer with caching support.
// This improves performance by caching authorization decisions
// for repeated requests with the same subject/resource/action.
type CachedAuthorizer struct {
	delegate Authorizer
	mu       sync.RWMutex
	cache    map[string]cacheEntry
	ttl      time.Duration
	maxSize  int

	// cleanupInterval is the interval for cleanup of expired entries
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

type cacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

// cacheItem represents a cache entry with its key for sorting purposes.
type cacheItem struct {
	key       string
	expiresAt time.Time
}

// CacheOption is a functional option for CachedAuthorizer.
type CacheOption func(*CachedAuthorizer)

// WithCacheTTL sets the cache TTL.
func WithCacheTTL(ttl time.Duration) CacheOption {
	return func(c *CachedAuthorizer) {
		c.ttl = ttl
	}
}

// WithCacheMaxSize sets the maximum cache size.
func WithCacheMaxSize(size int) CacheOption {
	return func(c *CachedAuthorizer) {
		c.maxSize = size
	}
}

// WithCacheCleanupInterval sets the cleanup interval.
func WithCacheCleanupInterval(d time.Duration) CacheOption {
	return func(c *CachedAuthorizer) {
		c.cleanupInterval = d
	}
}

// NewCachedAuthorizer creates a new cached authorizer.
func NewCachedAuthorizer(delegate Authorizer, opts ...CacheOption) *CachedAuthorizer {
	c := &CachedAuthorizer{
		delegate:        delegate,
		cache:           make(map[string]cacheEntry),
		ttl:             5 * time.Minute,
		maxSize:         10000,
		cleanupInterval: time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	go c.cleanup()

	return c
}

// Authorize checks authorization with caching.
func (c *CachedAuthorizer) Authorize(ctx context.Context, subject, resource, action string) (bool, error) {
	key := c.cacheKey(subject, resource, action)

	// Check cache first
	c.mu.RLock()
	entry, found := c.cache[key]
	c.mu.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		return entry.allowed, nil
	}

	// Cache miss or expired, call delegate
	allowed, err := c.delegate.Authorize(ctx, subject, resource, action)
	if err != nil {
		return false, err
	}

	// Update cache
	c.mu.Lock()
	// Check size limit
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}
	c.cache[key] = cacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return allowed, nil
}

// AuthorizeWithContext checks authorization with context (not cached).
func (c *CachedAuthorizer) AuthorizeWithContext(ctx context.Context, subject, resource, action string, context map[string]interface{}) (bool, error) {
	// Don't cache context-based authorization as context can vary
	return c.delegate.AuthorizeWithContext(ctx, subject, resource, action, context)
}

// Invalidate removes a specific entry from the cache.
func (c *CachedAuthorizer) Invalidate(subject, resource, action string) {
	key := c.cacheKey(subject, resource, action)
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

// InvalidateSubject removes all entries for a subject from the cache.
func (c *CachedAuthorizer) InvalidateSubject(subject string) {
	prefix := subject + ":"
	c.mu.Lock()
	for key := range c.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.cache, key)
		}
	}
	c.mu.Unlock()
}

// Clear removes all entries from the cache.
func (c *CachedAuthorizer) Clear() {
	c.mu.Lock()
	c.cache = make(map[string]cacheEntry)
	c.mu.Unlock()
}

// Close stops the cleanup goroutine.
func (c *CachedAuthorizer) Close() error {
	close(c.stopCleanup)
	return nil
}

// Size returns the current cache size.
func (c *CachedAuthorizer) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// cacheKey generates a cache key.
func (c *CachedAuthorizer) cacheKey(subject, resource, action string) string {
	return buildCacheKey(subject, resource, action)
}

// buildCacheKey builds cache key with optimized string concatenation.
func buildCacheKey(subject, resource, action string) string {
	var b strings.Builder
	b.Grow(len(subject) + len(resource) + len(action) + 2)
	b.WriteString(subject)
	b.WriteByte(':')
	b.WriteString(resource)
	b.WriteByte(':')
	b.WriteString(action)
	return b.String()
}

// evictOldest removes the oldest entries when cache is full.
// Must be called with lock held.
// This method implements a two-phase eviction strategy:
// 1. First, remove all expired entries
// 2. If insufficient, remove oldest (earliest expiring) active entries
func (c *CachedAuthorizer) evictOldest() {
	toRemove := c.maxSize / 10
	if toRemove < 1 {
		toRemove = 1
	}

	now := time.Now()

	// Categorize entries into expired and active
	expired, active := c.categorizeEntries(now)

	// Build final eviction list
	toDelete := c.buildEvictionList(expired, active, toRemove)

	// Execute deletions
	c.deleteEntries(toDelete)
}

// categorizeEntries separates cache entries into expired and active categories.
// This function is responsible solely for classification, not deletion.
func (c *CachedAuthorizer) categorizeEntries(now time.Time) (expired []string, active []cacheItem) {
	expired = make([]string, 0)
	active = make([]cacheItem, 0)

	for key, entry := range c.cache {
		if now.After(entry.expiresAt) {
			// Entry has expired
			expired = append(expired, key)
		} else {
			// Entry is still active
			active = append(active, cacheItem{
				key:       key,
				expiresAt: entry.expiresAt,
			})
		}
	}

	return expired, active
}

// buildEvictionList constructs the final list of keys to delete.
// It prioritizes expired entries first, then adds oldest active entries if needed.
func (c *CachedAuthorizer) buildEvictionList(expired []string, active []cacheItem, limit int) []string {
	toDelete := make([]string, 0, limit)

	// First, include all expired entries (up to limit)
	for i := 0; i < len(expired) && len(toDelete) < limit; i++ {
		toDelete = append(toDelete, expired[i])
	}

	// If we haven't reached the limit, add oldest active entries
	if len(toDelete) < limit {
		remaining := limit - len(toDelete)
		oldestActive := c.collectOldestEntries(active, remaining)
		toDelete = append(toDelete, oldestActive...)
	}

	return toDelete
}

// collectOldestEntries returns the keys of the oldest entries by expiration time.
// Entries are sorted by expiresAt in ascending order (earliest first).
func (c *CachedAuthorizer) collectOldestEntries(items []cacheItem, count int) []string {
	if len(items) == 0 {
		return []string{}
	}

	// Sort by expiration time (earliest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].expiresAt.Before(items[j].expiresAt)
	})

	// Collect the oldest 'count' entries
	limit := count
	if limit > len(items) {
		limit = len(items)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = items[i].key
	}

	return result
}

// deleteEntries removes the specified keys from the cache.
// This function is responsible solely for deletion, not selection logic.
func (c *CachedAuthorizer) deleteEntries(keys []string) {
	for _, key := range keys {
		delete(c.cache, key)
	}
}

// cleanup periodically removes expired entries.
func (c *CachedAuthorizer) cleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.doCleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// doCleanup removes expired entries in batches.
func (c *CachedAuthorizer) doCleanup() {
	// First pass: collect expired keys under read lock
	c.mu.RLock()
	var expired []string
	now := time.Now()
	for key, entry := range c.cache {
		if now.After(entry.expiresAt) {
			expired = append(expired, key)
		}
	}
	c.mu.RUnlock()

	if len(expired) == 0 {
		return
	}

	// Second pass: delete in batches under write lock
	const batchSize = 100
	for i := 0; i < len(expired); i += batchSize {
		end := i + batchSize
		if end > len(expired) {
			end = len(expired)
		}
		batch := expired[i:end]

		c.mu.Lock()
		for _, key := range batch {
			if entry, exists := c.cache[key]; exists && now.After(entry.expiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}
