package resilience

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// BenchmarkMemoryRateLimiterAllow 已在 ratelimit.go 中定义 (line 244-256)
// 这里添加 Redis 限流器的 benchmark

// BenchmarkRedisRateLimiterAllow 测试 Redis 限流器的性能
//
// 注意: 需要本地运行 Redis 服务器
//   docker run -d -p 6379:6379 redis:7-alpine
func BenchmarkRedisRateLimiterAllow(b *testing.B) {
	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	// 验证连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis 不可用,跳过测试: %v", err)
		return
	}

	// 清理测试数据
	client.FlushDB(ctx)
	defer client.FlushDB(ctx)

	limiter := NewRedisRateLimiter(client, 10000, 1*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(ctx, "test-key")
	}
}

// BenchmarkRedisRateLimiterAllow_Parallel 测试 Redis 限流器的并发性能
func BenchmarkRedisRateLimiterAllow_Parallel(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis 不可用,跳过测试: %v", err)
		return
	}

	client.FlushDB(ctx)
	defer client.FlushDB(ctx)

	limiter := NewRedisRateLimiter(client, 10000, 1*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = limiter.Allow(ctx, "test-key")
		}
	})
}

// BenchmarkRateLimiterComparison 对比内存和 Redis 限流器的性能
func BenchmarkRateLimiterComparison(b *testing.B) {
	ctx := context.Background()

	b.Run("Memory", func(b *testing.B) {
		limiter := NewMemoryRateLimiter(10000, 1*time.Minute)
		defer limiter.Stop()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = limiter.Allow(ctx, "test-key")
		}
	})

	b.Run("Redis", func(b *testing.B) {
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
		defer client.Close()

		if err := client.Ping(ctx).Err(); err != nil {
			b.Skipf("Redis 不可用,跳过测试: %v", err)
			return
		}

		client.FlushDB(ctx)
		defer client.FlushDB(ctx)

		limiter := NewRedisRateLimiter(client, 10000, 1*time.Minute)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = limiter.Allow(ctx, "test-key")
		}
	})
}

// TestRedisRateLimiter_MultiInstance 测试多实例场景下的 Redis 限流器
//
// 模拟微服务环境中多个实例共享限流配额
func TestRedisRateLimiter_MultiInstance(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis 不可用,跳过测试: %v", err)
		return
	}

	// 清理测试数据
	client.FlushDB(ctx)
	defer client.FlushDB(ctx)

	// 配置: 全局每分钟 100 次请求
	limit := 100
	window := 1 * time.Minute

	// 模拟 3 个服务实例
	instance1 := NewRedisRateLimiter(client, limit, window)
	instance2 := NewRedisRateLimiter(client, limit, window)
	instance3 := NewRedisRateLimiter(client, limit, window)

	// 每个实例尝试 50 次请求
	key := "user:12345"
	var allowedCount int

	// Instance 1: 50 次请求
	for i := 0; i < 50; i++ {
		allowed, err := instance1.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Instance 1 error: %v", err)
		}
		if allowed {
			allowedCount++
		}
	}

	// Instance 2: 50 次请求
	for i := 0; i < 50; i++ {
		allowed, err := instance2.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Instance 2 error: %v", err)
		}
		if allowed {
			allowedCount++
		}
	}

	// Instance 3: 50 次请求 (应该全部被拒绝)
	for i := 0; i < 50; i++ {
		allowed, err := instance3.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Instance 3 error: %v", err)
		}
		if allowed {
			allowedCount++
		}
	}

	// 验证: 总共允许 100 次请求 (全局限制)
	if allowedCount != limit {
		t.Errorf("预期允许 %d 次请求,实际 %d 次", limit, allowedCount)
	}

	t.Logf("✅ 多实例限流测试通过")
	t.Logf("   3 个实例共发起 150 次请求")
	t.Logf("   全局限制: %d 次/分钟", limit)
	t.Logf("   实际允许: %d 次", allowedCount)
	t.Logf("   拒绝: %d 次", 150-allowedCount)
}

// TestRedisRateLimiter_Reset 测试重置限流计数
func TestRedisRateLimiter_Reset(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis 不可用,跳过测试: %v", err)
		return
	}

	client.FlushDB(ctx)
	defer client.FlushDB(ctx)

	limiter := NewRedisRateLimiter(client, 10, 1*time.Minute)
	key := "test-key"

	// 消耗所有配额
	for i := 0; i < 10; i++ {
		allowed, _ := limiter.Allow(ctx, key)
		if !allowed {
			t.Errorf("第 %d 次请求应该被允许", i+1)
		}
	}

	// 第 11 次应该被拒绝
	allowed, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Error("第 11 次请求应该被拒绝")
	}

	// 重置计数
	if err := limiter.Reset(ctx, key); err != nil {
		t.Fatalf("Reset 失败: %v", err)
	}

	// 重置后应该可以再次请求
	allowed, _ = limiter.Allow(ctx, key)
	if !allowed {
		t.Error("重置后的第一次请求应该被允许")
	}

	t.Log("✅ Reset 功能测试通过")
}

// TestRedisRateLimiter_SlidingWindow 测试滑动窗口特性
func TestRedisRateLimiter_SlidingWindow(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis 不可用,跳过测试: %v", err)
		return
	}

	client.FlushDB(ctx)
	defer client.FlushDB(ctx)

	// 配置: 5 秒窗口,最多 5 次请求
	limiter := NewRedisRateLimiter(client, 5, 5*time.Second)
	key := "test-sliding"

	// 快速发送 5 次请求
	for i := 0; i < 5; i++ {
		allowed, _ := limiter.Allow(ctx, key)
		if !allowed {
			t.Errorf("第 %d 次请求应该被允许", i+1)
		}
	}

	// 第 6 次应该被拒绝
	allowed, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Error("第 6 次请求应该被拒绝")
	}

	// 等待窗口过期
	t.Log("等待 6 秒让窗口过期...")
	time.Sleep(6 * time.Second)

	// 窗口过期后应该可以再次请求
	allowed, _ = limiter.Allow(ctx, key)
	if !allowed {
		t.Error("窗口过期后的请求应该被允许")
	}

	t.Log("✅ 滑动窗口测试通过")
}
