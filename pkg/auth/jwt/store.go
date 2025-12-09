package jwt

import (
	"context"
	"sync"
	"time"
)

// Store defines the interface for token storage/revocation.
// Implementations can use Redis, database, or in-memory storage.
type Store interface {
	// Revoke marks a token as revoked.
	// The token should be stored until its natural expiration (TTL).
	Revoke(ctx context.Context, token string, expiration time.Duration) error

	// IsRevoked checks if a token has been revoked.
	IsRevoked(ctx context.Context, token string) (bool, error)

	// Close releases any resources used by the store.
	Close() error
}

// MemoryStore is an in-memory implementation of Store.
// Suitable for single-instance deployments or testing.
// For distributed systems, use Redis or database-backed store.
type MemoryStore struct {
	mu     sync.RWMutex
	tokens map[string]time.Time // token -> expiration time

	// cleanupInterval is the interval for cleanup of expired tokens
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// MemoryStoreOption is a functional option for MemoryStore.
type MemoryStoreOption func(*MemoryStore)

// NewMemoryStore creates a new in-memory token store.
func NewMemoryStore(opts ...MemoryStoreOption) *MemoryStore {
	s := &MemoryStore{
		tokens:          make(map[string]time.Time),
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Start cleanup goroutine
	go s.cleanup()

	return s
}

// WithCleanupInterval sets the cleanup interval.
func WithCleanupInterval(d time.Duration) MemoryStoreOption {
	return func(s *MemoryStore) {
		s.cleanupInterval = d
	}
}

// Revoke marks a token as revoked.
func (s *MemoryStore) Revoke(ctx context.Context, token string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token] = time.Now().Add(expiration)
	return nil
}

// IsRevoked checks if a token has been revoked.
func (s *MemoryStore) IsRevoked(ctx context.Context, token string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exp, exists := s.tokens[token]
	if !exists {
		return false, nil
	}

	// Check if the revocation entry has expired
	if time.Now().After(exp) {
		return false, nil
	}

	return true, nil
}

// Close stops the cleanup goroutine.
func (s *MemoryStore) Close() error {
	close(s.stopCleanup)
	return nil
}

// cleanup periodically removes expired entries.
func (s *MemoryStore) cleanup() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.doCleanup()
		case <-s.stopCleanup:
			return
		}
	}
}

// doCleanup removes expired entries.
func (s *MemoryStore) doCleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for token, exp := range s.tokens {
		if now.After(exp) {
			delete(s.tokens, token)
		}
	}
}

// Size returns the number of revoked tokens in the store.
func (s *MemoryStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tokens)
}

// NoopStore is a no-op implementation of Store.
// Useful when token revocation is not needed.
type NoopStore struct{}

// NewNoopStore creates a new no-op store.
func NewNoopStore() *NoopStore {
	return &NoopStore{}
}

// Revoke does nothing.
func (s *NoopStore) Revoke(ctx context.Context, token string, expiration time.Duration) error {
	return nil
}

// IsRevoked always returns false.
func (s *NoopStore) IsRevoked(ctx context.Context, token string) (bool, error) {
	return false, nil
}

// Close does nothing.
func (s *NoopStore) Close() error {
	return nil
}
