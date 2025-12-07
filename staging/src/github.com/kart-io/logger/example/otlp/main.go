package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== OTLP Integration Testing ===\n")

	// Example 1: OTLP with gRPC protocol
	demonstrateOTLPgRPC()

	// Example 2: OTLP with HTTP protocol
	demonstrateOTLPHTTP()

	// Example 3: OTLP with custom timeout
	demonstrateOTLPTimeout()

	// Example 4: OTLP error handling
	demonstrateOTLPErrorHandling()

	// Example 5: OTLP with tracing context
	demonstrateOTLPWithTracing()

	fmt.Println("\n=== OTLP Testing Complete ===")
}

// demonstrateOTLPgRPC shows OTLP configuration with gRPC protocol
func demonstrateOTLPgRPC() {
	fmt.Println("1. OTLP with gRPC Protocol")
	fmt.Println("==========================")

	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4327",
			Protocol: "grpc",
			Timeout:  10 * time.Second,
		},
	}

	otlpLogger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("Warning: OTLP gRPC logger creation failed: %v\n", err)
		fmt.Println("This is expected if no OTLP collector is running at 127.0.0.1:4327")
		return
	}

	fmt.Println("OTLP gRPC logger created successfully")

	// Test logging with OTLP
	otlpLogger.Infow("OTLP gRPC test message",
		"protocol", "grpc",
		"endpoint", "127.0.0.1:4327",
		"test_id", "grpc_001",
		"timestamp", time.Now(),
	)

	otlpLogger.Infow("Application metrics via OTLP",
		"cpu_usage", 45.2,
		"memory_mb", 128,
		"active_connections", 15,
		"request_rate", 120.5,
	)

	fmt.Println()
}

// demonstrateOTLPHTTP shows OTLP configuration with HTTP protocol
func demonstrateOTLPHTTP() {
	fmt.Println("2. OTLP with HTTP Protocol")
	fmt.Println("==========================")

	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4328",
			Protocol: "http",
			Timeout:  5 * time.Second,
		},
	}

	otlpLogger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("Warning: OTLP HTTP logger creation failed: %v\n", err)
		fmt.Println("This is expected if no OTLP collector is running at 127.0.0.1:4327")
		return
	}

	fmt.Println("OTLP HTTP logger created successfully")

	// Test logging with HTTP OTLP
	otlpLogger.Infow("OTLP HTTP test message",
		"protocol", "http",
		"endpoint", "127.0.0.1:4328",
		"test_id", "http_001",
		"timestamp", time.Now(),
	)

	otlpLogger.Warnw("HTTP OTLP warning test",
		"warning_type", "rate_limit",
		"current_rate", 95.0,
		"limit", 100.0,
		"retry_after", 30,
	)

	fmt.Println()
}

// demonstrateOTLPTimeout shows OTLP with custom timeout configuration
func demonstrateOTLPTimeout() {
	fmt.Println("3. OTLP with Custom Timeout")
	fmt.Println("===========================")

	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "DEBUG",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4327",
			Protocol: "grpc",
			Timeout:  2 * time.Second, // Short timeout for testing
		},
	}

	otlpLogger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("Warning: OTLP timeout logger creation failed: %v\n", err)
		fmt.Println("This is expected if no OTLP collector is running at 127.0.0.1:4327")
		return
	}

	fmt.Println("OTLP logger with 2s timeout created successfully")

	// Test various log levels
	otlpLogger.Debug("OTLP debug message with short timeout")
	otlpLogger.Infow("OTLP info with timeout test",
		"timeout_seconds", 2,
		"test_purpose", "timeout_configuration",
		"batch_id", "timeout_001",
	)

	otlpLogger.Errorw("OTLP error handling test",
		"error_type", "timeout_test",
		"expected_behavior", "fallback_to_stdout",
		"fallback_enabled", true,
	)

	fmt.Println()
}

// demonstrateOTLPErrorHandling shows OTLP error handling scenarios
func demonstrateOTLPErrorHandling() {
	fmt.Println("4. OTLP Error Handling")
	fmt.Println("======================")

	// Test with invalid endpoint
	fmt.Println("4.1 Testing with invalid endpoint:")
	invalidOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "invalid.endpoint:9999",
			Protocol: "grpc",
			Timeout:  1 * time.Second,
		},
	}

	invalidLogger, err := logger.New(invalidOpt)
	if err != nil {
		fmt.Printf("Expected error with invalid endpoint: %v\n", err)
	} else {
		fmt.Println("Logger created despite invalid endpoint - will fallback to stdout")
		invalidLogger.Warnw("Testing invalid endpoint handling",
			"endpoint", "invalid.endpoint:9999",
			"expected", "fallback_to_stdout",
		)
	}

	// Test with valid config but no collector
	fmt.Println("\n4.2 Testing with valid config but no collector:")
	validOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4327",
			Protocol: "grpc",
			Timeout:  3 * time.Second,
		},
	}

	validLogger, err := logger.New(validOpt)
	if err != nil {
		fmt.Printf("Error creating logger: %v\n", err)
	} else {
		fmt.Println("Logger created - testing connection to non-existent collector")
		validLogger.Infow("Testing collector connection",
			"collector_endpoint", "127.0.0.1:4327",
			"status", "attempting_connection",
			"expected_result", "timeout_or_fallback",
		)
	}

	fmt.Println()
}

// demonstrateOTLPWithTracing shows OTLP integration with tracing context
func demonstrateOTLPWithTracing() {
	fmt.Println("5. OTLP with Distributed Tracing")
	fmt.Println("================================")

	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4327",
			Protocol: "grpc",
			Timeout:  10 * time.Second,
		},
	}

	baseLogger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("Warning: OTLP tracing logger creation failed: %v\n", err)
		fmt.Println("This is expected if no OTLP collector is running at 127.0.0.1:4327")
		return
	}

	fmt.Println("OTLP logger with tracing context created successfully")

	// Create traced logger with context
	ctx := context.Background()
	tracedLogger := baseLogger.WithCtx(ctx,
		"trace_id", "trace_abc123def456789",
		"span_id", "span_012345678901234",
		"parent_span_id", "span_parent123456",
		"service_name", "otlp-test-service",
		"service_version", "1.0.0",
	)

	// Simulate a traced operation
	tracedLogger.Infow("OTLP traced operation started",
		"operation", "process_user_request",
		"user_id", 98765,
		"request_method", "POST",
		"endpoint", "/api/users/create",
	)

	// Simulate some processing time
	time.Sleep(50 * time.Millisecond)

	tracedLogger.Infow("Database operation via OTLP",
		"operation", "user_create",
		"table", "users",
		"duration_ms", 45,
		"rows_affected", 1,
	)

	tracedLogger.Infow("OTLP traced operation completed",
		"operation", "process_user_request",
		"status", "success",
		"response_code", 201,
		"total_duration_ms", 125,
		"otlp_endpoint", "127.0.0.1:4327",
	)

	// Test error scenarios with tracing
	tracedLogger.Errorw("OTLP traced error example",
		"error_type", "validation_failed",
		"error_message", "invalid email format",
		"user_input", "invalid-email",
		"validation_rule", "email_regex",
	)

	fmt.Println()
}
