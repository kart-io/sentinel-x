package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== Logger Performance Comparison ===\n")

	// Single-threaded performance
	demonstrateSingleThreadedPerformance()

	// Multi-threaded performance
	demonstrateMultiThreadedPerformance()

	// Memory allocation comparison
	demonstrateMemoryUsage()

	fmt.Println("\n=== Performance Comparison Complete ===")
}

// demonstrateSingleThreadedPerformance compares single-threaded performance
func demonstrateSingleThreadedPerformance() {
	fmt.Println("1. Single-threaded Performance Comparison")
	fmt.Println("==========================================")

	iterations := 10000

	// Test Slog engine with OTLP configuration
	slogOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"/dev/null"}, // Discard output for performance testing
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false), // Disabled for performance testing
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "performance-test-slog",
			},
		},
	}

	slogLogger, err := logger.New(slogOpt)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Testing Slog engine with %d iterations...\n", iterations)
	start := time.Now()

	for i := 0; i < iterations; i++ {
		slogLogger.Infow("Performance test message",
			"iteration", i,
			"timestamp", time.Now().Unix(),
			"worker_id", "worker_001",
			"batch_size", 100,
			"status", "processing",
		)
	}

	slogDuration := time.Since(start)
	fmt.Printf("Slog: %d logs in %v (%.2f logs/sec)\n", iterations, slogDuration, float64(iterations)/slogDuration.Seconds())

	// Test Zap engine with OTLP configuration
	zapOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"/dev/null"}, // Discard output for performance testing
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false), // Disabled for performance testing
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "performance-test-zap",
			},
		},
	}

	zapLogger, err := logger.New(zapOpt)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Testing Zap engine with %d iterations...\n", iterations)
	start = time.Now()

	for i := 0; i < iterations; i++ {
		zapLogger.Infow("Performance test message",
			"iteration", i,
			"timestamp", time.Now().Unix(),
			"worker_id", "worker_001",
			"batch_size", 100,
			"status", "processing",
		)
	}

	zapDuration := time.Since(start)
	fmt.Printf("Zap: %d logs in %v (%.2f logs/sec)\n", iterations, zapDuration, float64(iterations)/zapDuration.Seconds())

	// Performance comparison
	if zapDuration < slogDuration {
		speedup := float64(slogDuration) / float64(zapDuration)
		fmt.Printf("Result: Zap is %.2fx faster than Slog\n", speedup)
	} else {
		speedup := float64(zapDuration) / float64(slogDuration)
		fmt.Printf("Result: Slog is %.2fx faster than Zap\n", speedup)
	}

	fmt.Println()
}

// demonstrateMultiThreadedPerformance tests concurrent logging performance
func demonstrateMultiThreadedPerformance() {
	fmt.Println("2. Multi-threaded Performance Comparison")
	fmt.Println("=========================================")

	goroutines := 10
	iterationsPerGoroutine := 1000
	totalIterations := goroutines * iterationsPerGoroutine

	// Test Slog engine with concurrency
	slogOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"/dev/null"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false), // Disabled for performance testing
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "performance-test-slog-concurrent",
			},
		},
	}

	slogLogger, err := logger.New(slogOpt)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Testing Slog engine: %d goroutines x %d iterations = %d total logs\n",
		goroutines, iterationsPerGoroutine, totalIterations)

	start := time.Now()
	var wg sync.WaitGroup

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < iterationsPerGoroutine; i++ {
				slogLogger.Infow("Concurrent performance test",
					"goroutine_id", goroutineID,
					"iteration", i,
					"timestamp", time.Now().UnixNano(),
					"worker_type", "concurrent",
					"load_test", true,
				)
			}
		}(g)
	}

	wg.Wait()
	slogDuration := time.Since(start)
	fmt.Printf("Slog concurrent: %d logs in %v (%.2f logs/sec)\n",
		totalIterations, slogDuration, float64(totalIterations)/slogDuration.Seconds())

	// Test Zap engine with concurrency
	zapOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"/dev/null"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false), // Disabled for performance testing
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "performance-test-zap-concurrent",
			},
		},
	}

	zapLogger, err := logger.New(zapOpt)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Testing Zap engine: %d goroutines x %d iterations = %d total logs\n",
		goroutines, iterationsPerGoroutine, totalIterations)

	start = time.Now()
	wg = sync.WaitGroup{}

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < iterationsPerGoroutine; i++ {
				zapLogger.Infow("Concurrent performance test",
					"goroutine_id", goroutineID,
					"iteration", i,
					"timestamp", time.Now().UnixNano(),
					"worker_type", "concurrent",
					"load_test", true,
				)
			}
		}(g)
	}

	wg.Wait()
	zapDuration := time.Since(start)
	fmt.Printf("Zap concurrent: %d logs in %v (%.2f logs/sec)\n",
		totalIterations, zapDuration, float64(totalIterations)/zapDuration.Seconds())

	// Performance comparison
	if zapDuration < slogDuration {
		speedup := float64(slogDuration) / float64(zapDuration)
		fmt.Printf("Result: Zap is %.2fx faster than Slog in concurrent scenarios\n", speedup)
	} else {
		speedup := float64(zapDuration) / float64(slogDuration)
		fmt.Printf("Result: Slog is %.2fx faster than Zap in concurrent scenarios\n", speedup)
	}

	fmt.Println()
}

// demonstrateMemoryUsage shows memory allocation patterns
func demonstrateMemoryUsage() {
	fmt.Println("3. Memory Usage Patterns")
	fmt.Println("========================")

	// Create loggers that output to stdout for this demo
	slogOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false), // Disabled for memory usage demo
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "performance-test-memory-slog",
			},
		},
	}

	slogLogger, err := logger.New(slogOpt)
	if err != nil {
		panic(err)
	}

	zapOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false), // Disabled for memory usage demo
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "performance-test-memory-zap",
			},
		},
	}

	zapLogger, err := logger.New(zapOpt)
	if err != nil {
		panic(err)
	}

	// Demonstrate different logging patterns and their memory characteristics

	fmt.Println("3.1 Simple logging (minimal allocations):")
	slogLogger.Info("Simple slog message")
	zapLogger.Info("Simple zap message")

	fmt.Println("\n3.2 Structured logging with primitive types:")
	slogLogger.Infow("Slog structured with primitives",
		"int_field", 42,
		"float_field", 3.14,
		"bool_field", true,
		"string_field", "static_string",
	)

	zapLogger.Infow("Zap structured with primitives",
		"int_field", 42,
		"float_field", 3.14,
		"bool_field", true,
		"string_field", "static_string",
	)

	fmt.Println("\n3.3 Structured logging with complex types:")
	complexData := map[string]interface{}{
		"nested_map": map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		"slice":     []int{1, 2, 3, 4, 5},
		"timestamp": time.Now(),
	}

	slogLogger.Infow("Slog structured with complex types",
		"complex_data", complexData,
		"user_metadata", map[string]interface{}{
			"preferences": []string{"email", "sms"},
			"settings":    map[string]bool{"notifications": true},
		},
	)

	zapLogger.Infow("Zap structured with complex types",
		"complex_data", complexData,
		"user_metadata", map[string]interface{}{
			"preferences": []string{"email", "sms"},
			"settings":    map[string]bool{"notifications": true},
		},
	)

	fmt.Println("\n3.4 Logger with persistent fields (child loggers):")
	slogChild := slogLogger.With(
		"service", "user-service",
		"version", "1.0.0",
		"persistent_id", 12345,
	)

	zapChild := zapLogger.With(
		"service", "user-service",
		"version", "1.0.0",
		"persistent_id", 12345,
	)

	slogChild.Info("Slog child logger message")
	zapChild.Info("Zap child logger message")

	fmt.Println("\nMemory usage notes:")
	fmt.Println("- Zap generally has lower allocation overhead due to zero-allocation design")
	fmt.Println("- Slog uses more allocations but has simpler implementation")
	fmt.Println("- Both engines benefit from using primitive types over complex objects")
	fmt.Println("- Child loggers (With) reuse field allocations efficiently")

	fmt.Println()
}

// Helper function to create boolean pointers
func boolPtr(b bool) *bool {
	return &b
}
