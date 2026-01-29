package biz

import (
	"context"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/redis"
)

// RedisCaptchaStore implements base64Captcha.Store interface using Redis
type RedisCaptchaStore struct {
	redis  *redis.Client
	prefix string
	expiration time.Duration
}

// NewRedisCaptchaStore creates a new RedisCaptchaStore
func NewRedisCaptchaStore(redis *redis.Client, expiration time.Duration) *RedisCaptchaStore {
	return &RedisCaptchaStore{
		redis:      redis,
		prefix:     "captcha:",
		expiration: expiration,
	}
}

// Set sets the digits for the captcha id
func (s *RedisCaptchaStore) Set(id string, value string) error {
	ctx := context.Background()
	return s.redis.Client().Set(ctx, s.prefix+id, value, s.expiration).Err()
}

// Get returns stored digits for the captcha id. Clear indicates whether the captcha must be deleted from the store.
func (s *RedisCaptchaStore) Get(id string, clear bool) string {
	ctx := context.Background()
	key := s.prefix + id
	val, err := s.redis.Client().Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	if clear {
		s.redis.Client().Del(ctx, key)
	}
	return val
}

// Verify verifies the key and the answer.
func (s *RedisCaptchaStore) Verify(id, answer string, clear bool) bool {
	v := s.Get(id, clear)
	return v == answer
}
