package response

import (
	"testing"

	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// BenchmarkResponsePool compares pooled vs non-pooled Response allocation
func BenchmarkResponsePool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp := Acquire()
			resp.Code = 0
			resp.Message = "success"
			resp.Data = map[string]string{"key": "value"}
			Release(resp)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = &Response{
				Code:    0,
				Message: "success",
				Data:    map[string]string{"key": "value"},
			}
		}
	})
}

// BenchmarkSuccessResponse benchmarks Success function
func BenchmarkSuccessResponse(b *testing.B) {
	testData := map[string]interface{}{
		"user_id": 12345,
		"name":    "John Doe",
		"email":   "john@example.com",
	}

	b.Run("Success", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp := Success(testData)
			Release(resp)
		}
	})

	b.Run("SuccessWithMessage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp := SuccessWithMessage("operation completed", testData)
			Release(resp)
		}
	})
}

// BenchmarkErrorResponse benchmarks error response functions
func BenchmarkErrorResponse(b *testing.B) {
	testErr := errors.ErrInternal

	b.Run("Err", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp := Err(testErr)
			Release(resp)
		}
	})

	b.Run("ErrWithLang", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp := ErrWithLang(testErr, "en")
			Release(resp)
		}
	})

	b.Run("ErrorWithCode", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp := ErrorWithCode(400, "bad request")
			Release(resp)
		}
	})

	b.Run("ErrorWithData", func(b *testing.B) {
		b.ReportAllocs()
		data := map[string]string{"field": "validation error"}
		for i := 0; i < b.N; i++ {
			resp := ErrorWithData(400, "validation failed", data)
			Release(resp)
		}
	})
}

// BenchmarkPageResponse benchmarks paginated response
func BenchmarkPageResponse(b *testing.B) {
	testList := []map[string]interface{}{
		{"id": 1, "name": "item1"},
		{"id": 2, "name": "item2"},
		{"id": 3, "name": "item3"},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp := Page(testList, 100, 1, 10)
		Release(resp)
	}
}

// BenchmarkConcurrentPool benchmarks concurrent access to the pool
func BenchmarkConcurrentPool(b *testing.B) {
	b.Run("Concurrent_4", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(4)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp := Acquire()
				resp.Code = 0
				resp.Message = "success"
				Release(resp)
			}
		})
	})

	b.Run("Concurrent_8", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(8)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp := Acquire()
				resp.Code = 0
				resp.Message = "success"
				Release(resp)
			}
		})
	})

	b.Run("Concurrent_16", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(16)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp := Acquire()
				resp.Code = 0
				resp.Message = "success"
				Release(resp)
			}
		})
	})
}

// BenchmarkHighThroughput simulates 10K RPS scenario
func BenchmarkHighThroughput(b *testing.B) {
	testData := map[string]interface{}{
		"user_id": 12345,
		"status":  "active",
	}

	b.Run("HighThroughput_WithPool", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(100) // Simulate high concurrency
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp := Success(testData)
				// Simulate some work
				_ = resp.HTTPStatus()
				Release(resp)
			}
		})
	})

	b.Run("HighThroughput_WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(100)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp := &Response{
					Code:    0,
					Message: "success",
					Data:    testData,
				}
				// Simulate some work
				_ = resp.HTTPStatus()
			}
		})
	})
}

// TestPoolSafety ensures pool operations are safe
func TestPoolSafety(t *testing.T) {
	t.Run("AcquireAndRelease", func(t *testing.T) {
		resp := Acquire()
		if resp == nil {
			t.Fatal("Acquire returned nil")
		}

		resp.Code = 200
		resp.Message = "test"
		resp.Data = "data"
		resp.RequestID = "req-123"
		resp.Timestamp = 123456789

		Release(resp)

		// Verify fields are reset
		if resp.Code != 0 {
			t.Errorf("Code not reset: got %d, want 0", resp.Code)
		}
		if resp.Message != "" {
			t.Errorf("Message not reset: got %s, want empty", resp.Message)
		}
		if resp.Data != nil {
			t.Errorf("Data not reset: got %v, want nil", resp.Data)
		}
		if resp.RequestID != "" {
			t.Errorf("RequestID not reset: got %s, want empty", resp.RequestID)
		}
		if resp.Timestamp != 0 {
			t.Errorf("Timestamp not reset: got %d, want 0", resp.Timestamp)
		}
	})

	t.Run("ReleaseNil", func(_ *testing.T) {
		// Should not panic
		Release(nil)
	})

	t.Run("MultipleAcquireRelease", func(_ *testing.T) {
		for i := 0; i < 100; i++ {
			resp := Acquire()
			resp.Code = i
			Release(resp)
		}
	})
}

// TestConcurrentSafety tests concurrent pool access
func TestConcurrentSafety(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	done := make(chan bool, goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			for i := 0; i < iterations; i++ {
				resp := Acquire()
				resp.Code = id
				resp.Message = "test"
				// Simulate some work
				_ = resp.HTTPStatus()
				Release(resp)
			}
			done <- true
		}(g)
	}

	for g := 0; g < goroutines; g++ {
		<-done
	}
}
