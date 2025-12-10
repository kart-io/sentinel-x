package id

import (
	"sync"
	"testing"
)

// TestSnowflakeClockDrift tests various clock drift scenarios.
func TestSnowflakeClockDrift(t *testing.T) {
	t.Run("SmallClockDrift", func(t *testing.T) {
		// Test small clock drift (< 5 seconds)
		var currentTime int64 = 1704067200000 + 10000 // 10 seconds after epoch
		var mu sync.Mutex
		var callCount int

		timeFunc := func() int64 {
			mu.Lock()
			defer mu.Unlock()
			callCount++
			// Simulate time recovery: after drift, time gradually catches up
			if callCount > 2 {
				currentTime += 1 // Time moves forward to catch up
			}
			return currentTime
		}

		gen, err := NewSnowflakeGenerator(
			WithNodeID(1),
			WithTimeFunc(timeFunc),
		)
		if err != nil {
			t.Fatalf("failed to create generator: %v", err)
		}

		// Generate first ID
		id1 := gen.GenerateInt64()
		if id1 == 0 {
			t.Error("expected non-zero ID")
		}

		// Simulate small clock drift backward (100ms)
		mu.Lock()
		currentTime -= 100
		callCount = 0 // Reset call count for recovery simulation
		mu.Unlock()

		// Should handle clock drift and generate new ID
		// The time function will gradually advance time back
		id2 := gen.GenerateInt64()
		if id2 == 0 {
			t.Error("expected non-zero ID")
		}

		// IDs should be different
		if id1 == id2 {
			t.Error("expected different IDs")
		}
	})

	t.Run("LargeClockDrift", func(t *testing.T) {
		// Test large clock drift (> 5 seconds) - should panic
		var currentTime int64 = 1704067200000 + 10000
		var mu sync.Mutex

		timeFunc := func() int64 {
			mu.Lock()
			defer mu.Unlock()
			return currentTime
		}

		gen, err := NewSnowflakeGenerator(
			WithNodeID(1),
			WithTimeFunc(timeFunc),
		)
		if err != nil {
			t.Fatalf("failed to create generator: %v", err)
		}

		// Generate first ID
		id1 := gen.GenerateInt64()
		if id1 == 0 {
			t.Error("expected non-zero ID")
		}

		// Simulate large clock drift backward (6 seconds)
		mu.Lock()
		currentTime -= 6000
		mu.Unlock()

		// Should panic with clock drift error
		defer func() {
			if r := recover(); r != nil {
				if r != ErrClockMovedBackward {
					t.Errorf("expected ErrClockMovedBackward, got %v", r)
				}
			} else {
				t.Error("expected panic for large clock drift")
			}
		}()

		gen.GenerateInt64()
	})

	t.Run("NoClockDrift", func(t *testing.T) {
		// Test normal operation without clock drift
		var currentTime int64 = 1704067200000
		var mu sync.Mutex

		timeFunc := func() int64 {
			mu.Lock()
			defer mu.Unlock()
			currentTime++ // Normal time progression
			return currentTime
		}

		gen, err := NewSnowflakeGenerator(
			WithNodeID(1),
			WithTimeFunc(timeFunc),
		)
		if err != nil {
			t.Fatalf("failed to create generator: %v", err)
		}

		// Generate multiple IDs
		const numIDs = 100
		ids := make(map[int64]bool)

		for i := 0; i < numIDs; i++ {
			id := gen.GenerateInt64()
			if id == 0 {
				t.Error("expected non-zero ID")
			}
			if ids[id] {
				t.Errorf("duplicate ID: %d", id)
			}
			ids[id] = true
		}

		if len(ids) != numIDs {
			t.Errorf("expected %d unique IDs, got %d", numIDs, len(ids))
		}
	})

	t.Run("SequenceOverflow", func(t *testing.T) {
		// Test that sequence overflow waits for next millisecond
		var currentTime int64 = 1704067200000
		var mu sync.Mutex
		var overflowDetected bool

		timeFunc := func() int64 {
			mu.Lock()
			defer mu.Unlock()
			// After sequence overflow is detected, advance time
			if overflowDetected {
				currentTime++
			}
			return currentTime
		}

		gen, err := NewSnowflakeGenerator(
			WithNodeID(1),
			WithTimeFunc(timeFunc),
		)
		if err != nil {
			t.Fatalf("failed to create generator: %v", err)
		}

		// Generate IDs until sequence overflows (4096 IDs in same millisecond)
		for i := 0; i <= snowflakeMaxSeq; i++ {
			gen.GenerateInt64()
		}

		// Mark overflow as detected so time can advance
		mu.Lock()
		overflowDetected = true
		mu.Unlock()

		// Next ID should be generated successfully after time advances
		id := gen.GenerateInt64()
		if id == 0 {
			t.Error("expected non-zero ID after sequence overflow")
		}

		parsed := ParseSnowflake(id)
		if parsed.Sequence != 0 {
			t.Errorf("expected sequence to reset after overflow, got %d", parsed.Sequence)
		}
	})

	t.Run("ClockForwardProgress", func(t *testing.T) {
		// Test that time moving forward works correctly
		var currentTime int64 = 1704067200000
		var mu sync.Mutex

		timeFunc := func() int64 {
			mu.Lock()
			defer mu.Unlock()
			return currentTime
		}

		gen, err := NewSnowflakeGenerator(
			WithNodeID(1),
			WithTimeFunc(timeFunc),
		)
		if err != nil {
			t.Fatalf("failed to create generator: %v", err)
		}

		// Generate first ID
		id1 := gen.GenerateInt64()
		parsed1 := ParseSnowflake(id1)

		// Move time forward
		mu.Lock()
		currentTime += 1000 // 1 second forward
		mu.Unlock()

		// Generate second ID
		id2 := gen.GenerateInt64()
		parsed2 := ParseSnowflake(id2)

		// Second ID timestamp should be greater
		if parsed2.Timestamp <= parsed1.Timestamp {
			t.Errorf("expected timestamp to increase, got %d <= %d", parsed2.Timestamp, parsed1.Timestamp)
		}

		// Sequence should be reset
		if parsed2.Sequence != 0 {
			t.Errorf("expected sequence to reset to 0, got %d", parsed2.Sequence)
		}
	})

	t.Run("ConcurrentGeneration", func(t *testing.T) {
		// Test concurrent ID generation without clock drift
		gen, err := NewSnowflakeGenerator(WithNodeID(1))
		if err != nil {
			t.Fatalf("failed to create generator: %v", err)
		}

		const numGoroutines = 10
		const numIDsPerGoroutine = 100

		var wg sync.WaitGroup
		idMap := sync.Map{}

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numIDsPerGoroutine; j++ {
					id := gen.GenerateInt64()
					if id == 0 {
						t.Error("expected non-zero ID")
					}
					// Check for duplicates
					if _, exists := idMap.LoadOrStore(id, true); exists {
						t.Errorf("duplicate ID detected: %d", id)
					}
				}
			}()
		}

		wg.Wait()
	})
}

// BenchmarkSnowflakeNormal benchmarks normal ID generation without clock drift.
func BenchmarkSnowflakeNormal(b *testing.B) {
	gen, _ := NewSnowflakeGenerator(WithNodeID(1))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gen.GenerateInt64()
		}
	})
}
