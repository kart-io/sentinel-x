package biz

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 辅助函数：创建测试用 Redis 客户端
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试专用数据库
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis 不可用，跳过测试")
	}

	// 清空测试数据库
	client.FlushDB(ctx)

	return client
}

func TestNewQueryCache(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)
	assert.NotNil(t, cache)
	assert.Equal(t, config, cache.config)
	assert.Equal(t, client, cache.redis)
}

func TestNewQueryCache_WithNilConfig(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	cache := NewQueryCache(client, nil)
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.config)
	assert.False(t, cache.config.Enabled) // 默认禁用
	assert.Equal(t, 1*time.Hour, cache.config.TTL)
	assert.Equal(t, "rag:query:", cache.config.KeyPrefix)
}

func TestQueryCache_GenerateCacheKey(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)

	question1 := "什么是 RAG？"
	question2 := "什么是 RAG？" // 相同问题
	question3 := "RAG 是什么？" // 不同问题

	key1 := cache.generateCacheKey(question1)
	key2 := cache.generateCacheKey(question2)
	key3 := cache.generateCacheKey(question3)

	// 相同问题应生成相同的键
	assert.Equal(t, key1, key2)
	// 不同问题应生成不同的键
	assert.NotEqual(t, key1, key3)
	// 键应包含前缀
	assert.Contains(t, key1, config.KeyPrefix)
}

func TestQueryCache_SetAndGet(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)
	ctx := context.Background()

	question := "什么是向量数据库？"
	result := &model.QueryResult{
		Answer: "向量数据库是一种专门用于存储和检索向量嵌入的数据库。",
		Sources: []model.ChunkSource{
			{
				DocumentID:   "doc1",
				DocumentName: "vector_db.md",
				Section:      "Introduction",
				Content:      "向量数据库介绍...",
				Score:        0.95,
			},
		},
	}

	// 测试写入缓存
	err := cache.Set(ctx, question, result)
	require.NoError(t, err)

	// 测试读取缓存
	cached, err := cache.Get(ctx, question)
	require.NoError(t, err)
	require.NotNil(t, cached)
	assert.Equal(t, result.Answer, cached.Answer)
	assert.Equal(t, len(result.Sources), len(cached.Sources))
	assert.Equal(t, result.Sources[0].DocumentID, cached.Sources[0].DocumentID)
}

func TestQueryCache_GetMiss(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)
	ctx := context.Background()

	// 查询不存在的缓存
	result, err := cache.Get(ctx, "不存在的问题")
	require.NoError(t, err)
	assert.Nil(t, result) // 缓存未命中应返回 nil
}

func TestQueryCache_Disabled(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   false, // 禁用缓存
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)
	ctx := context.Background()

	question := "测试问题"
	result := &model.QueryResult{Answer: "测试答案"}

	// 禁用缓存时，Set 应该不报错但不写入
	err := cache.Set(ctx, question, result)
	assert.NoError(t, err)

	// 禁用缓存时，Get 应该返回错误
	cached, err := cache.Get(ctx, question)
	assert.Error(t, err)
	assert.Nil(t, cached)
}

func TestQueryCache_Clear(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)
	ctx := context.Background()

	// 写入多个缓存项
	for i := 0; i < 5; i++ {
		question := "问题" + string(rune('A'+i))
		result := &model.QueryResult{Answer: "答案" + string(rune('A'+i))}
		err := cache.Set(ctx, question, result)
		require.NoError(t, err)
	}

	// 清空缓存
	err := cache.Clear(ctx)
	require.NoError(t, err)

	// 验证所有缓存已清空
	for i := 0; i < 5; i++ {
		question := "问题" + string(rune('A'+i))
		cached, err := cache.Get(ctx, question)
		require.NoError(t, err)
		assert.Nil(t, cached) // 应该全部被清空
	}
}

func TestQueryCache_GetStats(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)
	ctx := context.Background()

	// 写入一些缓存项
	for i := 0; i < 3; i++ {
		question := "测试问题" + string(rune('1'+i))
		result := &model.QueryResult{Answer: "测试答案"}
		err := cache.Set(ctx, question, result)
		require.NoError(t, err)
	}

	// 获取统计信息
	stats, err := cache.GetStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.True(t, stats["enabled"].(bool))
	assert.Equal(t, 3, stats["key_count"].(int))
	assert.Equal(t, config.TTL.String(), stats["ttl"].(string))
	assert.Equal(t, config.KeyPrefix, stats["key_prefix"].(string))
}

func TestQueryCache_GetStats_Disabled(t *testing.T) {
	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   false,
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)
	ctx := context.Background()

	stats, err := cache.GetStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.False(t, stats["enabled"].(bool))
}

func TestQueryCache_TTLExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过耗时测试")
	}

	client := setupTestRedis(t)
	defer func() { _ = client.Close() }()

	config := &QueryCacheConfig{
		Enabled:   true,
		TTL:       2 * time.Second, // 短 TTL 用于测试
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(client, config)
	ctx := context.Background()

	question := "临时问题"
	result := &model.QueryResult{Answer: "临时答案"}

	// 写入缓存
	err := cache.Set(ctx, question, result)
	require.NoError(t, err)

	// 立即读取应该成功
	cached, err := cache.Get(ctx, question)
	require.NoError(t, err)
	require.NotNil(t, cached)

	// 等待 TTL 过期
	time.Sleep(3 * time.Second)

	// 再次读取应该缓存未命中
	cached, err = cache.Get(ctx, question)
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestQueryCache_NilRedis(t *testing.T) {
	config := &QueryCacheConfig{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "test:rag:",
	}

	cache := NewQueryCache(nil, config)
	ctx := context.Background()

	// Redis 为 nil 时，Get 应返回错误
	_, err := cache.Get(ctx, "问题")
	assert.Error(t, err)

	// Redis 为 nil 时，Set 应不报错（优雅降级）
	err = cache.Set(ctx, "问题", &model.QueryResult{Answer: "答案"})
	assert.NoError(t, err)

	// Redis 为 nil 时，Clear 应不报错
	err = cache.Clear(ctx)
	assert.NoError(t, err)

	// Redis 为 nil 时，GetStats 应返回禁用状态
	stats, err := cache.GetStats(ctx)
	require.NoError(t, err)
	assert.False(t, stats["enabled"].(bool))
}
