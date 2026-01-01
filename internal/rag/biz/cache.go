package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	goredis "github.com/redis/go-redis/v9"
)

// QueryCacheConfig 查询缓存配置。
type QueryCacheConfig struct {
	// Enabled 是否启用缓存。
	Enabled bool
	// TTL 缓存过期时间。
	TTL time.Duration
	// KeyPrefix 缓存键前缀。
	KeyPrefix string
}

// QueryCache 查询结果缓存。
type QueryCache struct {
	redis  *goredis.Client
	config *QueryCacheConfig
}

// NewQueryCache 创建查询缓存实例。
func NewQueryCache(redis *goredis.Client, config *QueryCacheConfig) *QueryCache {
	if config == nil {
		config = &QueryCacheConfig{
			Enabled:   false,
			TTL:       1 * time.Hour,
			KeyPrefix: "rag:query:",
		}
	}
	return &QueryCache{
		redis:  redis,
		config: config,
	}
}

// generateCacheKey 基于问题生成缓存键（使用 SHA256 哈希）。
func (c *QueryCache) generateCacheKey(question string) string {
	hash := sha256.Sum256([]byte(question))
	hashStr := hex.EncodeToString(hash[:])
	return c.config.KeyPrefix + hashStr
}

// Get 从缓存获取查询结果。
func (c *QueryCache) Get(ctx context.Context, question string) (*model.QueryResult, error) {
	if !c.config.Enabled || c.redis == nil {
		return nil, fmt.Errorf("cache not enabled or redis not available")
	}

	cacheKey := c.generateCacheKey(question)

	// 从 Redis 获取缓存数据
	data, err := c.redis.Get(ctx, cacheKey).Bytes()
	if err != nil {
		if err == goredis.Nil {
			// 缓存未命中
			logger.Debugw("cache miss", "question", question, "key", cacheKey)
			return nil, nil
		}
		logger.Warnw("failed to get from cache", "error", err.Error(), "key", cacheKey)
		return nil, err
	}

	// 反序列化
	var result model.QueryResult
	if err := json.Unmarshal(data, &result); err != nil {
		logger.Warnw("failed to unmarshal cached result", "error", err.Error(), "key", cacheKey)
		// 删除损坏的缓存
		_ = c.redis.Del(ctx, cacheKey).Err()
		return nil, err
	}

	logger.Infow("cache hit", "question", question, "key", cacheKey, "answer_length", len(result.Answer))
	return &result, nil
}

// Set 将查询结果写入缓存。
func (c *QueryCache) Set(ctx context.Context, question string, result *model.QueryResult) error {
	if !c.config.Enabled || c.redis == nil {
		return nil
	}

	cacheKey := c.generateCacheKey(question)

	// 序列化
	data, err := json.Marshal(result)
	if err != nil {
		logger.Warnw("failed to marshal result for caching", "error", err.Error())
		return err
	}

	// 写入 Redis
	err = c.redis.Set(ctx, cacheKey, data, c.config.TTL).Err()
	if err != nil {
		logger.Warnw("failed to set cache", "error", err.Error(), "key", cacheKey)
		return err
	}

	logger.Infow("cached query result", "question", question, "key", cacheKey, "ttl", c.config.TTL)
	return nil
}

// Clear 清除所有 RAG 查询缓存。
func (c *QueryCache) Clear(ctx context.Context) error {
	if !c.config.Enabled || c.redis == nil {
		return nil
	}

	// 使用 SCAN 命令查找所有匹配的键
	pattern := c.config.KeyPrefix + "*"
	iter := c.redis.Scan(ctx, 0, pattern, 0).Iterator()

	deletedCount := 0
	for iter.Next(ctx) {
		if err := c.redis.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Warnw("failed to delete cache key", "error", err.Error(), "key", iter.Val())
		} else {
			deletedCount++
		}
	}

	if err := iter.Err(); err != nil {
		logger.Warnw("error during cache scan", "error", err.Error())
		return err
	}

	logger.Infow("cleared query cache", "deleted_count", deletedCount)
	return nil
}

// GetStats 获取缓存统计信息。
func (c *QueryCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if !c.config.Enabled || c.redis == nil {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	// 统计缓存键数量
	pattern := c.config.KeyPrefix + "*"
	iter := c.redis.Scan(ctx, 0, pattern, 0).Iterator()

	keyCount := 0
	for iter.Next(ctx) {
		keyCount++
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"enabled":    true,
		"key_count":  keyCount,
		"ttl":        c.config.TTL.String(),
		"key_prefix": c.config.KeyPrefix,
	}, nil
}
