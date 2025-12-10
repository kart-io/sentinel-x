package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/redis"
)

// RedisStore implements the Store interface using Redis.
// It is suitable for distributed deployments.
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore creates a new Redis-backed token store.
func NewRedisStore(client *redis.Client, prefix string) *RedisStore {
	if prefix == "" {
		prefix = "jwt:blacklisted:"
	}
	return &RedisStore{
		client: client,
		prefix: prefix,
	}
}

// Revoke marks a token as revoked in Redis.
func (s *RedisStore) Revoke(ctx context.Context, token string, expiration time.Duration) error {
	key := s.prefix + token
	// Set the key with the remaining expiration time of the token
	return s.client.Client().Set(ctx, key, "revoked", expiration).Err()
}

// IsRevoked checks if a token exists in the Redis blacklist.
func (s *RedisStore) IsRevoked(ctx context.Context, token string) (bool, error) {
	key := s.prefix + token
	count, err := s.client.Client().Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis error: %w", err)
	}
	return count > 0, nil
}

// Close is a no-op for RedisStore as the client is managed externally.
func (s *RedisStore) Close() error {
	return nil
}
