package jwt

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryStore_Basic(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	token := "test-token-123"

	// 测试未撤销的令牌
	revoked, err := store.IsRevoked(ctx, token)
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if revoked {
		t.Error("Token should not be revoked initially")
	}

	// 测试撤销令牌
	err = store.Revoke(ctx, token, time.Hour)
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	// 验证令牌已撤销
	revoked, err = store.IsRevoked(ctx, token)
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if !revoked {
		t.Error("Token should be revoked")
	}
}

func TestMemoryStore_MultipleTokens(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	tokens := []string{
		"token-1",
		"token-2",
		"token-3",
	}

	// 撤销所有令牌
	for _, token := range tokens {
		err := store.Revoke(ctx, token, time.Hour)
		if err != nil {
			t.Fatalf("Failed to revoke token %s: %v", token, err)
		}
	}

	// 验证所有令牌都已撤销
	for _, token := range tokens {
		revoked, err := store.IsRevoked(ctx, token)
		if err != nil {
			t.Fatalf("IsRevoked error for %s: %v", token, err)
		}
		if !revoked {
			t.Errorf("Token %s should be revoked", token)
		}
	}

	// 检查 store 大小
	size := store.Size()
	if size != len(tokens) {
		t.Errorf("Expected size %d, got %d", len(tokens), size)
	}
}

func TestMemoryStore_Expiration(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	token := "expiring-token"

	// 撤销令牌，设置很短的过期时间
	err := store.Revoke(ctx, token, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	// 立即检查应该是已撤销
	revoked, err := store.IsRevoked(ctx, token)
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if !revoked {
		t.Error("Token should be revoked immediately after revocation")
	}

	// 等待令牌过期
	time.Sleep(100 * time.Millisecond)

	// 检查令牌是否已过期（不再被视为已撤销）
	revoked, err = store.IsRevoked(ctx, token)
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if revoked {
		t.Error("Token should not be revoked after expiration")
	}
}

func TestMemoryStore_Cleanup(t *testing.T) {
	// 创建清理间隔很短的 store
	store := NewMemoryStore(WithCleanupInterval(50 * time.Millisecond))
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// 添加多个令牌，设置短过期时间
	for i := 0; i < 10; i++ {
		token := "token-" + string(rune(i))
		err := store.Revoke(ctx, token, 30*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to revoke token: %v", err)
		}
	}

	// 验证所有令牌都在 store 中
	initialSize := store.Size()
	if initialSize != 10 {
		t.Errorf("Expected initial size 10, got %d", initialSize)
	}

	// 等待令牌过期和清理
	time.Sleep(150 * time.Millisecond)

	// 检查 store 大小应该减少（清理已过期的令牌）
	finalSize := store.Size()
	if finalSize != 0 {
		t.Errorf("Expected final size 0 after cleanup, got %d", finalSize)
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	const numGoroutines = 100
	var wg sync.WaitGroup

	// 并发撤销令牌
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			token := "token-" + string(rune(id))
			err := store.Revoke(ctx, token, time.Hour)
			if err != nil {
				t.Errorf("Concurrent revoke failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// 并发检查令牌
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			token := "token-" + string(rune(id))
			revoked, err := store.IsRevoked(ctx, token)
			if err != nil {
				t.Errorf("Concurrent IsRevoked failed: %v", err)
			}
			if !revoked {
				t.Errorf("Token %s should be revoked", token)
			}
		}(i)
	}
	wg.Wait()
}

func TestMemoryStore_UpdateExpiration(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	token := "renewable-token"

	// 第一次撤销，设置短过期时间
	err := store.Revoke(ctx, token, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	// 等待一段时间
	time.Sleep(50 * time.Millisecond)

	// 再次撤销同一令牌，延长过期时间
	err = store.Revoke(ctx, token, time.Hour)
	if err != nil {
		t.Fatalf("Second revoke error: %v", err)
	}

	// 等待超过第一次的过期时间
	time.Sleep(100 * time.Millisecond)

	// 令牌应该仍然被撤销（因为过期时间被延长了）
	revoked, err := store.IsRevoked(ctx, token)
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if !revoked {
		t.Error("Token should still be revoked after expiration update")
	}
}

func TestMemoryStore_CleanupWithMixedExpiration(t *testing.T) {
	store := NewMemoryStore(WithCleanupInterval(50 * time.Millisecond))
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// 添加一些很快过期的令牌
	for i := 0; i < 5; i++ {
		token := "short-token-" + string(rune(i))
		err := store.Revoke(ctx, token, 30*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to revoke short token: %v", err)
		}
	}

	// 添加一些较长过期的令牌
	for i := 0; i < 5; i++ {
		token := "long-token-" + string(rune(i))
		err := store.Revoke(ctx, token, time.Hour)
		if err != nil {
			t.Fatalf("Failed to revoke long token: %v", err)
		}
	}

	// 初始大小应该是 10
	if size := store.Size(); size != 10 {
		t.Errorf("Expected initial size 10, got %d", size)
	}

	// 等待短期令牌过期和清理
	time.Sleep(150 * time.Millisecond)

	// 大小应该减少到 5（只剩长期令牌）
	finalSize := store.Size()
	if finalSize != 5 {
		t.Errorf("Expected final size 5, got %d", finalSize)
	}

	// 验证长期令牌仍然存在
	for i := 0; i < 5; i++ {
		token := "long-token-" + string(rune(i))
		revoked, err := store.IsRevoked(ctx, token)
		if err != nil {
			t.Fatalf("IsRevoked error: %v", err)
		}
		if !revoked {
			t.Errorf("Long token %s should still be revoked", token)
		}
	}
}

func TestMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore()

	// 添加一些令牌
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		token := "token-" + string(rune('0'+i))
		err := store.Revoke(ctx, token, time.Hour)
		if err != nil {
			t.Fatalf("Failed to revoke token: %v", err)
		}
	}

	// 验证令牌已撤销
	revoked, err := store.IsRevoked(ctx, "token-0")
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if !revoked {
		t.Error("Token should be revoked before close")
	}

	// 关闭 store
	err = store.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}

	// 关闭后仍然可以查询（清理 goroutine 已停止，但数据仍在）
	revoked, err = store.IsRevoked(ctx, "token-0")
	if err != nil {
		t.Fatalf("IsRevoked error after close: %v", err)
	}
	if !revoked {
		t.Error("Token should still be revoked after close")
	}
}

func TestNoopStore(t *testing.T) {
	store := NewNoopStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	token := "test-token"

	// NoopStore 不应该撤销令牌
	err := store.Revoke(ctx, token, time.Hour)
	if err != nil {
		t.Fatalf("NoopStore.Revoke should not error: %v", err)
	}

	// NoopStore 总是返回 false（未撤销）
	revoked, err := store.IsRevoked(ctx, token)
	if err != nil {
		t.Fatalf("NoopStore.IsRevoked should not error: %v", err)
	}
	if revoked {
		t.Error("NoopStore should always return false for IsRevoked")
	}

	// Close 应该成功
	err = store.Close()
	if err != nil {
		t.Fatalf("NoopStore.Close should not error: %v", err)
	}
}

func TestNoopStore_Multiple(t *testing.T) {
	store := NewNoopStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// 撤销多个令牌
	tokens := []string{"token-1", "token-2", "token-3"}
	for _, token := range tokens {
		err := store.Revoke(ctx, token, time.Hour)
		if err != nil {
			t.Fatalf("NoopStore.Revoke error: %v", err)
		}
	}

	// 所有令牌都应该显示为未撤销
	for _, token := range tokens {
		revoked, err := store.IsRevoked(ctx, token)
		if err != nil {
			t.Fatalf("NoopStore.IsRevoked error: %v", err)
		}
		if revoked {
			t.Errorf("Token %s should not be revoked in NoopStore", token)
		}
	}
}

func BenchmarkMemoryStore_Revoke(b *testing.B) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token := "token-" + string(rune(i))
		_ = store.Revoke(ctx, token, time.Hour)
	}
}

func BenchmarkMemoryStore_IsRevoked(b *testing.B) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// 预先撤销一些令牌
	for i := 0; i < 1000; i++ {
		token := "token-" + string(rune(i))
		_ = store.Revoke(ctx, token, time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token := "token-" + string(rune(i%1000))
		_, _ = store.IsRevoked(ctx, token)
	}
}

func BenchmarkMemoryStore_ConcurrentRevoke(b *testing.B) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			token := "token-" + string(rune(i))
			_ = store.Revoke(ctx, token, time.Hour)
			i++
		}
	})
}

func BenchmarkMemoryStore_ConcurrentIsRevoked(b *testing.B) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// 预先撤销一些令牌
	for i := 0; i < 1000; i++ {
		token := "token-" + string(rune(i))
		_ = store.Revoke(ctx, token, time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			token := "token-" + string(rune(i%1000))
			_, _ = store.IsRevoked(ctx, token)
			i++
		}
	})
}

func BenchmarkNoopStore_Revoke(b *testing.B) {
	store := NewNoopStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token := "token-" + string(rune(i))
		_ = store.Revoke(ctx, token, time.Hour)
	}
}

func BenchmarkNoopStore_IsRevoked(b *testing.B) {
	store := NewNoopStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token := "token-" + string(rune(i))
		_, _ = store.IsRevoked(ctx, token)
	}
}
