package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/auth"
	"github.com/redis/go-redis/v9"
)

const KeyPrefix = "casbin:rules"

type Repository struct {
	client *redis.Client
	key    string
}

// NewRepository creates a new Redis repository
func NewRepository(client *redis.Client, key ...string) *Repository {
	k := KeyPrefix
	if len(key) > 0 {
		k = key[0]
	}
	return &Repository{
		client: client,
		key:    k,
	}
}

func (r *Repository) LoadPolicies(ctx context.Context) ([]*auth.Policy, error) {
	val, err := r.client.LRange(ctx, r.key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	policies := make([]*auth.Policy, 0, len(val))
	for _, v := range val {
		var p auth.Policy
		if err := json.Unmarshal([]byte(v), &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
		}
		policies = append(policies, &p)
	}
	return policies, nil
}

func (r *Repository) SavePolicies(ctx context.Context, policies []*auth.Policy) error {
	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.key)

	if len(policies) > 0 {
		args := make([]interface{}, len(policies))
		for i, p := range policies {
			b, err := json.Marshal(p)
			if err != nil {
				return err
			}
			args[i] = string(b)
		}
		pipe.RPush(ctx, r.key, args...)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *Repository) AddPolicy(ctx context.Context, policy *auth.Policy) error {
	b, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	return r.client.RPush(ctx, r.key, string(b)).Err()
}

func (r *Repository) RemovePolicy(ctx context.Context, policy *auth.Policy) error {
	b, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	// LREM count=1 (remove first occurrence) or 0 (all occurrences)?
	// Casbin usually implies unique rules, so 1 is fine, but 0 is safer if duplicates exist.
	return r.client.LRem(ctx, r.key, 0, string(b)).Err()
}

func (r *Repository) RemoveFilteredPolicy(ctx context.Context, ptype string, fieldIndex int, fieldValues ...string) error {
	// Redis doesn't support filtering list elements by content pattern efficiently.
	// We must load all, filter, and save back.
	// Using a Watch/Multi transaction to ensure atomicity.

	return r.client.Watch(ctx, func(tx *redis.Tx) error {
		val, err := tx.LRange(ctx, r.key, 0, -1).Result()
		if err != nil {
			return err
		}

		var keep []interface{}
		for _, v := range val {
			var p auth.Policy
			if err := json.Unmarshal([]byte(v), &p); err != nil {
				return err
			}

			if !matchFilter(&p, ptype, fieldIndex, fieldValues...) {
				keep = append(keep, v)
			}
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, r.key)
			if len(keep) > 0 {
				pipe.RPush(ctx, r.key, keep...)
			}
			return nil
		})
		return err
	}, r.key)
}

func matchFilter(p *auth.Policy, ptype string, fieldIndex int, fieldValues ...string) bool {
	if p.PType != ptype {
		return false
	}

	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) {
		if p.V0 != fieldValues[0-fieldIndex] {
			return false
		}
	}
	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) {
		if p.V1 != fieldValues[1-fieldIndex] {
			return false
		}
	}
	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) {
		if p.V2 != fieldValues[2-fieldIndex] {
			return false
		}
	}
	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) {
		if p.V3 != fieldValues[3-fieldIndex] {
			return false
		}
	}
	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) {
		if p.V4 != fieldValues[4-fieldIndex] {
			return false
		}
	}
	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) {
		if p.V5 != fieldValues[5-fieldIndex] {
			return false
		}
	}
	return true
}
