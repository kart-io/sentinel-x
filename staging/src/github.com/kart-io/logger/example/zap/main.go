package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== Zap Engine Comprehensive Usage Examples ===\n")

	// Production Configuration Example
	demonstrateProductionConfig()

	// Development Configuration Example
	demonstrateDevelopmentConfig()

	// High Performance Logging Example
	demonstrateHighPerformance()

	// Error Handling with Stacktraces
	demonstrateErrorHandling()

	// Advanced Structured Logging
	demonstrateAdvancedStructured()

	// Performance Comparison
	demonstratePerformanceFeatures()

	fmt.Println("\n=== Zap Examples Complete ===")
}

// demonstrateProductionConfig shows production-ready Zap configuration
func demonstrateProductionConfig() {
	fmt.Println("1. Production Configuration")
	fmt.Println("===========================")

	opt := &option.LogOption{
		Engine:            "zap",
		Level:             "INFO",
		Format:            "json",
		OutputPaths:       []string{"stdout"},
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		OTLP:              &option.OTLPOption{},
	}

	prodLogger, err := logger.New(opt)
	if err != nil {
		panic(err)
	}

	// Production logging examples
	prodLogger.Infow("Application started",
		"app_name", "user-service",
		"version", "1.0.0",
		"environment", "production",
		"port", 8080,
		"startup_time", time.Now(),
	)

	prodLogger.Infow("Database connection established",
		"database", "postgresql",
		"host", "prod-db.example.com",
		"port", 5432,
		"connection_pool_size", 20,
		"ssl_enabled", true,
	)

	prodLogger.Warnw("High memory usage detected",
		"memory_usage_percent", 85.7,
		"memory_limit_mb", 2048,
		"gc_count", 15,
		"alert_threshold", 80.0,
	)

	fmt.Println()
}

// demonstrateDevelopmentConfig shows development-friendly Zap configuration
func demonstrateDevelopmentConfig() {
	fmt.Println("2. Development Configuration")
	fmt.Println("============================")

	opt := &option.LogOption{
		Engine:        "zap",
		Level:         "DEBUG",
		Format:        "console",
		OutputPaths:   []string{"stdout"},
		Development:   true,
		DisableCaller: false,
		OTLP:          &option.OTLPOption{},
	}

	devLogger, err := logger.New(opt)
	if err != nil {
		panic(err)
	}

	// Development logging examples
	devLogger.Debug("Debug information for development")
	devLogger.Debugw("Variable values",
		"user_id", 12345,
		"session_token", "dev_token_xyz",
		"debug_mode", true,
	)

	devLogger.Info("Processing user request")
	devLogger.Infof("User %s requested resource %s", "alice", "/api/profile")

	fmt.Println()
}

// demonstrateHighPerformance shows Zap's high-performance features
func demonstrateHighPerformance() {
	fmt.Println("3. High Performance Logging")
	fmt.Println("===========================")

	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
		OTLP:        &option.OTLPOption{},
	}

	perfLogger, err := logger.New(opt)
	if err != nil {
		panic(err)
	}

	// Simulate high-frequency logging
	start := time.Now()

	for i := 0; i < 10; i++ {
		perfLogger.Infow("High frequency log entry",
			"iteration", i,
			"timestamp", time.Now(),
			"worker_id", fmt.Sprintf("worker_%d", i%3),
			"batch_size", 100,
			"processing_time_ns", time.Since(start).Nanoseconds(),
		)
	}

	// Log performance metrics
	perfLogger.Infow("Performance test completed",
		"total_logs", 10,
		"total_time_ms", time.Since(start).Milliseconds(),
		"logs_per_second", float64(10)/time.Since(start).Seconds(),
	)

	fmt.Println()
}

// demonstrateErrorHandling shows error handling with rich context
func demonstrateErrorHandling() {
	fmt.Println("4. Error Handling and Stacktraces")
	fmt.Println("==================================")

	opt := &option.LogOption{
		Engine:            "zap",
		Level:             "INFO",
		Format:            "json",
		OutputPaths:       []string{"stdout"},
		DisableStacktrace: false,
		OTLP:              &option.OTLPOption{},
	}

	errorLogger, err := logger.New(opt)
	if err != nil {
		panic(err)
	}

	// Simulate various error scenarios
	err = processPayment(12345, 99.99)
	if err != nil {
		errorLogger.Errorw("Payment processing failed",
			"user_id", 12345,
			"amount", 99.99,
			"currency", "USD",
			"payment_method", "credit_card",
			"error", err.Error(),
			"retry_count", 0,
			"max_retries", 3,
		)
	}

	err = validateUser("invalid_user")
	if err != nil {
		errorLogger.Errorw("User validation failed",
			"username", "invalid_user",
			"validation_type", "authentication",
			"error", err.Error(),
			"ip_address", "192.168.1.100",
			"user_agent", "Mozilla/5.0...",
		)
	}

	fmt.Println()
}

// Helper functions for error simulation
func processPayment(userID int, amount float64) error {
	// Simulate payment processing
	time.Sleep(10 * time.Millisecond)
	return errors.New("insufficient funds")
}

func validateUser(username string) error {
	if username == "invalid_user" {
		return errors.New("user not found in database")
	}
	return nil
}

// demonstrateAdvancedStructured shows advanced structured logging patterns
func demonstrateAdvancedStructured() {
	fmt.Println("5. Advanced Structured Logging")
	fmt.Println("===============================")

	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP:        &option.OTLPOption{},
	}

	baseLogger, err := logger.New(opt)
	if err != nil {
		panic(err)
	}

	// Create service-specific logger with persistent fields
	serviceLogger := baseLogger.With(
		"service", "user-service",
		"version", "2.1.0",
		"environment", "production",
		"region", "us-west-2",
	)

	// Create request-specific logger
	ctx := context.WithValue(context.Background(), "request_id", "req_abc123")
	requestLogger := serviceLogger.WithCtx(ctx,
		"request_id", "req_abc123",
		"method", "POST",
		"path", "/api/users/profile",
		"user_id", 67890,
	)

	// Log request lifecycle
	requestLogger.Infow("Request started",
		"content_length", 512,
		"content_type", "application/json",
		"authorization", "Bearer",
	)

	// Simulate processing steps
	requestLogger.Infow("Validating request data",
		"validation_rules", []string{"required_fields", "data_types", "business_rules"},
	)

	requestLogger.Infow("Database query executing",
		"table", "user_profiles",
		"query_type", "UPDATE",
		"where_clause", "user_id = ?",
	)

	requestLogger.Infow("Request completed successfully",
		"response_code", 200,
		"response_size_bytes", 1024,
		"processing_time_ms", 145,
		"database_time_ms", 89,
		"validation_time_ms", 23,
	)

	fmt.Println()
}

// demonstratePerformanceFeatures shows Zap performance-oriented features
func demonstratePerformanceFeatures() {
	fmt.Println("6. Performance Features")
	fmt.Println("=======================")

	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
		OTLP:        &option.OTLPOption{},
	}

	perfLogger, err := logger.New(opt)
	if err != nil {
		panic(err)
	}

	// Zero-allocation logging patterns
	perfLogger.Infow("Zero-allocation logging example",
		"string_field", "static_string",
		"int_field", 42,
		"float_field", 3.14159,
		"bool_field", true,
		"time_field", time.Now(),
	)

	// Batch processing simulation
	batchLogger := perfLogger.With(
		"batch_id", "batch_2025_001",
		"processor", "data_pipeline_v2",
	)

	for batch := 1; batch <= 5; batch++ {
		start := time.Now()

		// Simulate batch processing
		time.Sleep(20 * time.Millisecond)

		batchLogger.Infow("Batch processed",
			"batch_number", batch,
			"records_processed", batch*1000,
			"processing_time_ms", time.Since(start).Milliseconds(),
			"throughput_rps", float64(batch*1000)/time.Since(start).Seconds(),
		)
	}

	// Memory and performance metrics
	perfLogger.Infow("Performance metrics",
		"total_batches", 5,
		"total_records", 15000,
		"avg_throughput_rps", 12500.0,
		"memory_usage_mb", 45.2,
		"gc_pause_ms", 2.1,
	)

	fmt.Println()
}
