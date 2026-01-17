package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/kart-io/logger"
	goredis "github.com/redis/go-redis/v9"
)

// EmbeddingCacheConfig Embedding 缓存配置。
type EmbeddingCacheConfig struct {
	// Enabled 是否启用缓存。
	Enabled bool
	// TTL 缓存过期时间。
	TTL time.Duration
	// KeyPrefix 缓存键前缀。
	KeyPrefix string
}

// DefaultEmbeddingCacheConfig 返回默认的 Embedding 缓存配置。
func DefaultEmbeddingCacheConfig() *EmbeddingCacheConfig {
	return &EmbeddingCacheConfig{
		Enabled:   true,
		TTL:       24 * time.Hour, // Embedding 结果相对稳定，可以缓存更长时间
		KeyPrefix: "emb:",
	}
}

// CachedEmbeddingProvider 提供 Embedding 缓存功能的包装器。
type CachedEmbeddingProvider struct {
	provider EmbeddingProvider
	redis    *goredis.Client
	config   *EmbeddingCacheConfig
}

// NewCachedEmbeddingProvider 创建带缓存的 Embedding Provider。
func NewCachedEmbeddingProvider(
	provider EmbeddingProvider,
	redis *goredis.Client,
	config *EmbeddingCacheConfig,
) *CachedEmbeddingProvider {
	if config == nil {
		config = DefaultEmbeddingCacheConfig()
	}
	return &CachedEmbeddingProvider{
		provider: provider,
		redis:    redis,
		config:   config,
	}
}

// generateCacheKey 基于文本生成缓存键（使用 SHA256 哈希）。
func (c *CachedEmbeddingProvider) generateCacheKey(text string) string {
	hash := sha256.Sum256([]byte(text))
	hashStr := hex.EncodeToString(hash[:])
	return c.config.KeyPrefix + hashStr
}

// EmbedSingle 生成单个文本的 Embedding（带缓存）。
func (c *CachedEmbeddingProvider) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	// 如果缓存未启用或 Redis 不可用，直接调用底层 provider
	if !c.config.Enabled || c.redis == nil {
		return c.provider.EmbedSingle(ctx, text)
	}

	cacheKey := c.generateCacheKey(text)

	// 1. 尝试从缓存获取
	data, err := c.redis.Get(ctx, cacheKey).Bytes()
	if err == nil {
		// 缓存命中
		var embedding []float32
		if err := json.Unmarshal(data, &embedding); err == nil {
			logger.Debugw("embedding cache hit", "text_length", len(text), "key", cacheKey)
			return embedding, nil
		}
		// 反序列化失败，删除损坏的缓存
		logger.Warnw("failed to unmarshal cached embedding, deleting", "error", err.Error(), "key", cacheKey)
		_ = c.redis.Del(ctx, cacheKey).Err()
	} else if err != goredis.Nil {
		// Redis 错误（非缓存未命中），记录但继续
		logger.Warnw("redis get error, falling back to provider", "error", err.Error())
	}

	// 2. 缓存未命中，调用底层 provider
	logger.Debugw("embedding cache miss", "text_length", len(text), "key", cacheKey)
	embedding, err := c.provider.EmbedSingle(ctx, text)
	if err != nil {
		return nil, err
	}

	// 3. 缓存结果
	data, err = json.Marshal(embedding)
	if err != nil {
		logger.Warnw("failed to marshal embedding for caching", "error", err.Error())
		return embedding, nil // 返回结果，缓存失败不影响功能
	}

	err = c.redis.Set(ctx, cacheKey, data, c.config.TTL).Err()
	if err != nil {
		logger.Warnw("failed to cache embedding", "error", err.Error(), "key", cacheKey)
	} else {
		logger.Debugw("embedding cached", "text_length", len(text), "key", cacheKey, "ttl", c.config.TTL)
	}

	return embedding, nil
}

// Embed 批量生成 Embedding（带缓存）。
func (c *CachedEmbeddingProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// 如果缓存未启用或 Redis 不可用，直接调用底层 provider
	if !c.config.Enabled || c.redis == nil {
		return c.provider.Embed(ctx, texts)
	}

	embeddings := make([][]float32, len(texts))
	uncachedIndices := []int{}
	uncachedTexts := []string{}

	// 1. 尝试从缓存获取
	for i, text := range texts {
		cacheKey := c.generateCacheKey(text)
		data, err := c.redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			// 缓存命中
			var embedding []float32
			if err := json.Unmarshal(data, &embedding); err == nil {
				embeddings[i] = embedding
				logger.Debugw("embedding cache hit (batch)", "index", i, "text_length", len(text))
				continue
			}
			// 反序列化失败，删除损坏的缓存
			_ = c.redis.Del(ctx, cacheKey).Err()
		}

		// 缓存未命中，记录索引
		uncachedIndices = append(uncachedIndices, i)
		uncachedTexts = append(uncachedTexts, text)
	}

	// 2. 批量计算未缓存的 Embedding
	if len(uncachedTexts) > 0 {
		logger.Infow("embedding cache miss (batch)", "total", len(texts), "uncached", len(uncachedTexts))
		uncachedEmbeddings, err := c.provider.Embed(ctx, uncachedTexts)
		if err != nil {
			return nil, err
		}

		// 3. 填充结果并缓存
		for i, idx := range uncachedIndices {
			embeddings[idx] = uncachedEmbeddings[i]

			// 缓存结果
			cacheKey := c.generateCacheKey(uncachedTexts[i])
			data, err := json.Marshal(uncachedEmbeddings[i])
			if err != nil {
				logger.Warnw("failed to marshal embedding for caching", "error", err.Error())
				continue
			}

			err = c.redis.Set(ctx, cacheKey, data, c.config.TTL).Err()
			if err != nil {
				logger.Warnw("failed to cache embedding", "error", err.Error(), "key", cacheKey)
			}
		}
	} else {
		logger.Infow("all embeddings from cache", "total", len(texts))
	}

	return embeddings, nil
}

// Name 返回底层 provider 的名称。
func (c *CachedEmbeddingProvider) Name() string {
	return c.provider.Name() + "-cached"
}

// ClearCache 清除所有 Embedding 缓存。
func (c *CachedEmbeddingProvider) ClearCache(ctx context.Context) error {
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

	logger.Infow("cleared embedding cache", "deleted_count", deletedCount)
	return nil
}

// GetCacheStats 获取缓存统计信息。
func (c *CachedEmbeddingProvider) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
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
		"provider":   c.provider.Name(),
	}, nil
}

// 确保 CachedEmbeddingProvider 实现了 EmbeddingProvider 接口。
var _ EmbeddingProvider = (*CachedEmbeddingProvider)(nil)
