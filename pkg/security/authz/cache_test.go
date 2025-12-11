package authz

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockAuthorizer is a mock implementation of Authorizer for testing.
type mockAuthorizer struct {
	mu        sync.Mutex
	decisions map[string]bool
	calls     int
}

// newMockAuthorizer creates a new mock authorizer.
func newMockAuthorizer(decisions map[string]bool) *mockAuthorizer {
	return &mockAuthorizer{
		decisions: decisions,
	}
}

// Authorize implements Authorizer interface.
func (m *mockAuthorizer) Authorize(ctx context.Context, subject, resource, action string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	key := subject + ":" + resource + ":" + action
	return m.decisions[key], nil
}

// AuthorizeWithContext implements Authorizer interface.
func (m *mockAuthorizer) AuthorizeWithContext(ctx context.Context, subject, resource, action string, context map[string]interface{}) (bool, error) {
	return m.Authorize(ctx, subject, resource, action)
}

// getCalls returns the number of times Authorize was called.
func (m *mockAuthorizer) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// resetCalls resets the call counter.
func (m *mockAuthorizer) resetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = 0
}

// TestCachedAuthorizer tests basic caching functionality.
func TestCachedAuthorizer(t *testing.T) {
	// Create mock authorizer
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read":   true,
		"user-1:posts:delete": false,
		"user-2:posts:read":   true,
	})

	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// First call should invoke delegate
	allowed, err := cached.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Expected allowed=true")
	}
	if mockAuth.getCalls() != 1 {
		t.Errorf("Expected 1 delegate call, got %d", mockAuth.getCalls())
	}

	// Second call should use cache
	allowed, err = cached.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Expected allowed=true from cache")
	}
	if mockAuth.getCalls() != 1 {
		t.Errorf("Expected 1 delegate call (cached), got %d", mockAuth.getCalls())
	}

	// Different request should invoke delegate again
	allowed, err = cached.Authorize(ctx, "user-1", "posts", "delete")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("Expected allowed=false")
	}
	if mockAuth.getCalls() != 2 {
		t.Errorf("Expected 2 delegate calls, got %d", mockAuth.getCalls())
	}
}

// TestCacheExpiration tests cache expiration.
func TestCacheExpiration(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read": true,
	})

	// Set very short TTL for testing
	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(100*time.Millisecond),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// First call
	allowed, err := cached.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Expected allowed=true")
	}
	if mockAuth.getCalls() != 1 {
		t.Errorf("Expected 1 delegate call, got %d", mockAuth.getCalls())
	}

	// Second call immediately - should use cache
	allowed, err = cached.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Expected allowed=true")
	}
	if mockAuth.getCalls() != 1 {
		t.Errorf("Expected 1 delegate call (cached), got %d", mockAuth.getCalls())
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call after expiration - should invoke delegate
	allowed, err = cached.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Expected allowed=true")
	}
	if mockAuth.getCalls() != 2 {
		t.Errorf("Expected 2 delegate calls (expired), got %d", mockAuth.getCalls())
	}
}

// TestCacheEviction tests cache eviction when max size is reached.
func TestCacheEviction(t *testing.T) {
	mockAuth := newMockAuthorizer(make(map[string]bool))

	// Populate decisions
	for i := 0; i < 150; i++ {
		key := "user-1:resource-" + string(rune(i)) + ":read"
		mockAuth.decisions[key] = true
	}

	// Set small max size
	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// Fill cache beyond max size
	for i := 0; i < 150; i++ {
		resource := "resource-" + string(rune(i))
		_, err := cached.Authorize(ctx, "user-1", resource, "read")
		if err != nil {
			t.Fatalf("Authorize error: %v", err)
		}
	}

	// Cache size should not exceed max size
	size := cached.Size()
	if size > 100 {
		t.Errorf("Cache size %d exceeds max size 100", size)
	}

	// Cache should still contain some entries
	if size == 0 {
		t.Error("Cache size should not be 0 after eviction")
	}
}

// TestCacheInvalidate tests cache invalidation.
func TestCacheInvalidate(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read": true,
	})

	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// First call - populate cache
	_, err := cached.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}

	// Verify cache is populated
	if cached.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cached.Size())
	}

	// Invalidate the entry
	cached.Invalidate("user-1", "posts", "read")

	// Cache should be empty
	if cached.Size() != 0 {
		t.Errorf("Expected cache size 0 after invalidation, got %d", cached.Size())
	}

	// Next call should invoke delegate
	mockAuth.resetCalls()
	_, err = cached.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if mockAuth.getCalls() != 1 {
		t.Errorf("Expected 1 delegate call after invalidation, got %d", mockAuth.getCalls())
	}
}

// TestCacheInvalidateSubject tests subject-based cache invalidation.
func TestCacheInvalidateSubject(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read":    true,
		"user-1:posts:write":   true,
		"user-1:comments:read": true,
		"user-2:posts:read":    true,
	})

	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// Populate cache with multiple entries
	_, _ = cached.Authorize(ctx, "user-1", "posts", "read")
	_, _ = cached.Authorize(ctx, "user-1", "posts", "write")
	_, _ = cached.Authorize(ctx, "user-1", "comments", "read")
	_, _ = cached.Authorize(ctx, "user-2", "posts", "read")

	// Verify cache size
	if cached.Size() != 4 {
		t.Errorf("Expected cache size 4, got %d", cached.Size())
	}

	// Invalidate all entries for user-1
	cached.InvalidateSubject("user-1")

	// Cache should only contain user-2's entry
	if cached.Size() != 1 {
		t.Errorf("Expected cache size 1 after subject invalidation, got %d", cached.Size())
	}

	// user-1's requests should invoke delegate
	mockAuth.resetCalls()
	_, _ = cached.Authorize(ctx, "user-1", "posts", "read")
	if mockAuth.getCalls() != 1 {
		t.Errorf("Expected 1 delegate call for user-1, got %d", mockAuth.getCalls())
	}

	// user-2's request should use cache
	_, _ = cached.Authorize(ctx, "user-2", "posts", "read")
	if mockAuth.getCalls() != 1 {
		t.Errorf("Expected 1 delegate call (user-2 cached), got %d", mockAuth.getCalls())
	}
}

// TestCacheClear tests clearing the entire cache.
func TestCacheClear(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read": true,
		"user-2:posts:read": true,
	})

	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// Populate cache
	_, _ = cached.Authorize(ctx, "user-1", "posts", "read")
	_, _ = cached.Authorize(ctx, "user-2", "posts", "read")

	if cached.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cached.Size())
	}

	// Clear cache
	cached.Clear()

	// Cache should be empty
	if cached.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cached.Size())
	}
}

// TestCacheAuthorizeWithContext tests that context-based authorization is not cached.
func TestCacheAuthorizeWithContext(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read": true,
	})

	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// Call AuthorizeWithContext twice
	_, err := cached.AuthorizeWithContext(ctx, "user-1", "posts", "read", map[string]interface{}{
		"owner": "user-1",
	})
	if err != nil {
		t.Fatalf("AuthorizeWithContext error: %v", err)
	}

	_, err = cached.AuthorizeWithContext(ctx, "user-1", "posts", "read", map[string]interface{}{
		"owner": "user-1",
	})
	if err != nil {
		t.Fatalf("AuthorizeWithContext error: %v", err)
	}

	// Both calls should invoke delegate (not cached)
	if mockAuth.getCalls() != 2 {
		t.Errorf("Expected 2 delegate calls (context not cached), got %d", mockAuth.getCalls())
	}
}

// TestCacheConcurrency tests concurrent access to the cache.
func TestCacheConcurrency(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read": true,
		"user-2:posts:read": true,
		"user-3:posts:read": true,
	})

	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			user := "user-" + string(rune(idx%3+1))
			_, _ = cached.Authorize(ctx, user, "posts", "read")
		}(i)
	}

	wg.Wait()

	// Should have 3 entries in cache (one for each user)
	size := cached.Size()
	if size != 3 {
		t.Errorf("Expected cache size 3, got %d", size)
	}
}

// TestCacheCleanup tests automatic cleanup of expired entries.
func TestCacheCleanup(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read": true,
		"user-2:posts:read": true,
	})

	// Set short TTL and cleanup interval
	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(100*time.Millisecond),
		WithCacheMaxSize(100),
		WithCacheCleanupInterval(50*time.Millisecond),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// Populate cache
	_, _ = cached.Authorize(ctx, "user-1", "posts", "read")
	_, _ = cached.Authorize(ctx, "user-2", "posts", "read")

	if cached.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cached.Size())
	}

	// Wait for entries to expire and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Cache should be cleaned up
	size := cached.Size()
	if size != 0 {
		t.Errorf("Expected cache size 0 after cleanup, got %d", size)
	}
}

// TestCacheOptions tests cache option configurations.
func TestCacheOptions(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{})

	tests := []struct {
		name   string
		opts   []CacheOption
		verify func(*testing.T, *CachedAuthorizer)
	}{
		{
			name: "default options",
			opts: nil,
			verify: func(t *testing.T, c *CachedAuthorizer) {
				if c.ttl != 5*time.Minute {
					t.Errorf("Expected default TTL 5m, got %v", c.ttl)
				}
				if c.maxSize != 10000 {
					t.Errorf("Expected default maxSize 10000, got %d", c.maxSize)
				}
				if c.cleanupInterval != time.Minute {
					t.Errorf("Expected default cleanupInterval 1m, got %v", c.cleanupInterval)
				}
			},
		},
		{
			name: "custom TTL",
			opts: []CacheOption{WithCacheTTL(10 * time.Minute)},
			verify: func(t *testing.T, c *CachedAuthorizer) {
				if c.ttl != 10*time.Minute {
					t.Errorf("Expected TTL 10m, got %v", c.ttl)
				}
			},
		},
		{
			name: "custom max size",
			opts: []CacheOption{WithCacheMaxSize(500)},
			verify: func(t *testing.T, c *CachedAuthorizer) {
				if c.maxSize != 500 {
					t.Errorf("Expected maxSize 500, got %d", c.maxSize)
				}
			},
		},
		{
			name: "custom cleanup interval",
			opts: []CacheOption{WithCacheCleanupInterval(30 * time.Second)},
			verify: func(t *testing.T, c *CachedAuthorizer) {
				if c.cleanupInterval != 30*time.Second {
					t.Errorf("Expected cleanupInterval 30s, got %v", c.cleanupInterval)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := NewCachedAuthorizer(mockAuth, tt.opts...)
			defer func() { _ = cached.Close() }()
			tt.verify(t, cached)
		})
	}
}

// TestCacheSize tests the Size method.
func TestCacheSize(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read":  true,
		"user-1:posts:write": true,
		"user-2:posts:read":  true,
	})

	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// Initially empty
	if cached.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", cached.Size())
	}

	// Add entries
	_, _ = cached.Authorize(ctx, "user-1", "posts", "read")
	if cached.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cached.Size())
	}

	_, _ = cached.Authorize(ctx, "user-1", "posts", "write")
	if cached.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cached.Size())
	}

	_, _ = cached.Authorize(ctx, "user-2", "posts", "read")
	if cached.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cached.Size())
	}

	// Duplicate request should not increase size
	_, _ = cached.Authorize(ctx, "user-1", "posts", "read")
	if cached.Size() != 3 {
		t.Errorf("Expected size 3 (cached), got %d", cached.Size())
	}
}

// TestCacheDifferentDecisions tests caching of both allow and deny decisions.
func TestCacheDifferentDecisions(t *testing.T) {
	mockAuth := newMockAuthorizer(map[string]bool{
		"user-1:posts:read":   true,
		"user-1:posts:write":  false,
		"user-1:posts:delete": false,
	})

	cached := NewCachedAuthorizer(mockAuth,
		WithCacheTTL(time.Minute),
		WithCacheMaxSize(100),
	)
	defer func() { _ = cached.Close() }()

	ctx := context.Background()

	// Cache allow decision
	allowed, _ := cached.Authorize(ctx, "user-1", "posts", "read")
	if !allowed {
		t.Error("Expected allowed=true")
	}

	// Cache deny decision
	allowed, _ = cached.Authorize(ctx, "user-1", "posts", "write")
	if allowed {
		t.Error("Expected allowed=false")
	}

	// Verify both are cached
	mockAuth.resetCalls()
	_, _ = cached.Authorize(ctx, "user-1", "posts", "read")
	_, _ = cached.Authorize(ctx, "user-1", "posts", "write")
	if mockAuth.getCalls() != 0 {
		t.Errorf("Expected 0 delegate calls (cached), got %d", mockAuth.getCalls())
	}
}
