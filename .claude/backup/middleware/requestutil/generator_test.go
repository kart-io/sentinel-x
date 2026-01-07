package requestutil

import (
	"crypto/rand"
	"fmt"
	"sync"
	"testing"

	"github.com/oklog/ulid/v2"
)

// ============================================================================
// Benchmark: ID 生成器性能对比
// ============================================================================

// BenchmarkRandomHexGenerator 测试当前默认的随机十六进制生成器
func BenchmarkRandomHexGenerator(b *testing.B) {
	gen := &RandomHexGenerator{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = gen.Generate()
	}
}

// BenchmarkULIDGenerator 测试 ULID 生成器
func BenchmarkULIDGenerator(b *testing.B) {
	gen := NewULIDGenerator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = gen.Generate()
	}
}

// BenchmarkULIDGeneratorParallel 测试 ULID 生成器并发性能
func BenchmarkULIDGeneratorParallel(b *testing.B) {
	gen := NewULIDGenerator()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = gen.Generate()
		}
	})
}

// BenchmarkRandomHexGeneratorParallel 测试随机十六进制生成器并发性能
func BenchmarkRandomHexGeneratorParallel(b *testing.B) {
	gen := &RandomHexGenerator{}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = gen.Generate()
		}
	})
}

// ============================================================================
// Benchmark: 与原生 ULID 库对比
// ============================================================================

// BenchmarkULIDNative 测试直接使用 oklog/ulid 库 (无锁,不保证单调性)
func BenchmarkULIDNative(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ulid.MustNew(ulid.Now(), rand.Reader).String()
	}
}

// BenchmarkULIDNativeParallel 测试原生 ULID 并发性能
func BenchmarkULIDNativeParallel(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = ulid.MustNew(ulid.Now(), rand.Reader).String()
		}
	})
}

// ============================================================================
// Benchmark: 实际使用场景 (中间件集成)
// ============================================================================

// BenchmarkGeneratorInMiddleware 模拟在中间件中使用的场景
func BenchmarkGeneratorInMiddleware(b *testing.B) {
	tests := []struct {
		name      string
		generator IDGenerator
	}{
		{
			name:      "RandomHex",
			generator: &RandomHexGenerator{},
		},
		{
			name:      "ULID",
			generator: NewULIDGenerator(),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// 模拟中间件操作:生成 ID + 存储到 map
				requestID := tt.generator.Generate()
				_ = map[string]string{"request_id": requestID}
			}
		})
	}
}

// ============================================================================
// Benchmark: 唯一性压力测试
// ============================================================================

// BenchmarkGeneratorUniqueness 测试生成器在高并发下的唯一性
func BenchmarkGeneratorUniqueness(b *testing.B) {
	const concurrency = 100

	tests := []struct {
		name      string
		generator IDGenerator
	}{
		{
			name:      "RandomHex",
			generator: &RandomHexGenerator{},
		},
		{
			name:      "ULID",
			generator: NewULIDGenerator(),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var mu sync.Mutex
			seen := make(map[string]bool)
			collisions := 0

			b.ResetTimer()

			var wg sync.WaitGroup
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < b.N/concurrency; j++ {
						id := tt.generator.Generate()
						mu.Lock()
						if seen[id] {
							collisions++
						} else {
							seen[id] = true
						}
						mu.Unlock()
					}
				}()
			}
			wg.Wait()

			if collisions > 0 {
				b.Errorf("发现 %d 次碰撞 (生成了 %d 个 ID)", collisions, len(seen))
			}
		})
	}
}

// ============================================================================
// Benchmark: 字符串长度对比
// ============================================================================

// BenchmarkGeneratorStringLength 对比不同生成器的字符串长度
func BenchmarkGeneratorStringLength(b *testing.B) {
	generators := map[string]IDGenerator{
		"RandomHex": &RandomHexGenerator{},
		"ULID":      NewULIDGenerator(),
	}

	for name, gen := range generators {
		id := gen.Generate()
		b.Logf("%s 长度: %d 字符, 示例: %s", name, len(id), id)
	}

	b.SkipNow() // 这个测试只是为了显示信息,不需要实际运行
}

// ============================================================================
// Test: 生成器正确性
// ============================================================================

// TestRandomHexGenerator 测试随机十六进制生成器的正确性
func TestRandomHexGenerator(t *testing.T) {
	gen := &RandomHexGenerator{}

	// 测试基本生成
	id := gen.Generate()
	if id == "" {
		t.Error("生成的 ID 不应为空")
	}

	// 测试长度 (16 字节 = 32 字符十六进制)
	if len(id) != 32 {
		t.Errorf("期望长度 32,实际 %d", len(id))
	}

	// 测试唯一性 (生成 1000 个 ID)
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := gen.Generate()
		if seen[id] {
			t.Errorf("发现重复 ID: %s", id)
		}
		seen[id] = true
	}
}

// TestULIDGenerator 测试 ULID 生成器的正确性
func TestULIDGenerator(t *testing.T) {
	gen := NewULIDGenerator()

	// 测试基本生成
	id := gen.Generate()
	if id == "" {
		t.Error("生成的 ID 不应为空")
	}

	// 测试长度 (ULID 固定 26 字符)
	if len(id) != 26 {
		t.Errorf("期望长度 26,实际 %d", len(id))
	}

	// 测试时间可排序性 (连续生成的 ID 应该是递增的)
	prev := gen.Generate()
	for i := 0; i < 100; i++ {
		current := gen.Generate()
		if current <= prev {
			t.Errorf("ULID 应该是单调递增的,但 %s <= %s", current, prev)
		}
		prev = current
	}

	// 测试唯一性
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := gen.Generate()
		if seen[id] {
			t.Errorf("发现重复 ID: %s", id)
		}
		seen[id] = true
	}
}

// TestULIDMonotonicity 测试 ULID 单调性 (同一毫秒内生成的 ID 也是递增的)
func TestULIDMonotonicity(t *testing.T) {
	gen := NewULIDGenerator()

	// 在极短时间内生成大量 ID (可能在同一毫秒)
	ids := make([]string, 10000)
	for i := 0; i < len(ids); i++ {
		ids[i] = gen.Generate()
	}

	// 验证所有 ID 都是递增的
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("ID 不是单调递增的: %s <= %s (索引 %d)", ids[i], ids[i-1], i)
		}
	}
}

// TestNewGenerator 测试工厂函数
func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name          string
		generatorType string
		expectedType  string
	}{
		{"ULID 类型", "ulid", "*requestutil.ULIDGenerator"},
		{"Random 类型", "random", "*requestutil.RandomHexGenerator"},
		{"Hex 类型", "hex", "*requestutil.RandomHexGenerator"},
		{"空字符串", "", "*requestutil.RandomHexGenerator"},
		{"未知类型", "unknown", "*requestutil.RandomHexGenerator"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.generatorType)
			if gen == nil {
				t.Error("生成器不应为 nil")
			}

			actualType := fmt.Sprintf("%T", gen)
			if actualType != tt.expectedType {
				t.Errorf("期望类型 %s,实际 %s", tt.expectedType, actualType)
			}

			// 测试生成功能
			id := gen.Generate()
			if id == "" {
				t.Error("生成的 ID 不应为空")
			}
		})
	}
}

// TestGeneratorConcurrency 测试生成器的并发安全性
func TestGeneratorConcurrency(t *testing.T) {
	generators := []struct {
		name string
		gen  IDGenerator
	}{
		{"RandomHex", &RandomHexGenerator{}},
		{"ULID", NewULIDGenerator()},
	}

	for _, gen := range generators {
		t.Run(gen.name, func(t *testing.T) {
			const goroutines = 100
			const idsPerGoroutine = 100

			var mu sync.Mutex
			seen := make(map[string]bool)
			collisions := 0

			var wg sync.WaitGroup
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < idsPerGoroutine; j++ {
						id := gen.gen.Generate()
						mu.Lock()
						if seen[id] {
							collisions++
						} else {
							seen[id] = true
						}
						mu.Unlock()
					}
				}()
			}
			wg.Wait()

			if collisions > 0 {
				t.Errorf("%s: 发现 %d 次碰撞 (总共生成 %d 个 ID)", gen.name, collisions, len(seen))
			}

			expectedTotal := goroutines * idsPerGoroutine
			if len(seen) != expectedTotal-collisions {
				t.Errorf("%s: 期望 %d 个唯一 ID,实际 %d", gen.name, expectedTotal, len(seen))
			}
		})
	}
}
