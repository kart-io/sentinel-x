package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/integrations/gorm"
	"github.com/kart-io/logger/integrations/kratos"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== Framework Integrations Example ===\n")

	// Create our unified logger
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "DEBUG",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP:        &option.OTLPOption{},
	}

	coreLogger, err := logger.New(opt)
	if err != nil {
		panic(err)
	}

	// Example 1: GORM Integration
	demonstrateGormIntegration(coreLogger)

	// Example 2: Kratos Integration
	demonstrateKratosIntegration(coreLogger)

	fmt.Println("\n=== All Integration Examples Complete ===")
}

func demonstrateGormIntegration(coreLogger core.Logger) {
	fmt.Println("1. GORM Integration Example")
	fmt.Println("===========================")

	// Create GORM adapter
	gormAdapter := gorm.NewGormAdapter(coreLogger)

	// Configure GORM logging settings
	gormAdapter.SetSlowThreshold(100 * time.Millisecond) // 100ms slow query threshold
	gormAdapter.SetIgnoreRecordNotFoundError(true)

	// Simulate GORM logging scenarios

	// 1. Normal query
	fmt.Println("\n1.1 Normal Query:")
	ctx := context.Background()
	begin := time.Now().Add(-50 * time.Millisecond) // 50ms query
	gormAdapter.Trace(ctx, begin, func() (string, int64) {
		return "SELECT * FROM users WHERE active = true", 25
	}, nil)

	// 2. Slow query
	fmt.Println("\n1.2 Slow Query:")
	slowBegin := time.Now().Add(-200 * time.Millisecond) // 200ms query (slow)
	gormAdapter.Trace(ctx, slowBegin, func() (string, int64) {
		return "SELECT u.*, p.* FROM users u JOIN profiles p ON u.id = p.user_id WHERE u.created_at > ?", 1000
	}, nil)

	// 3. Error query
	fmt.Println("\n1.3 Error Query:")
	errorBegin := time.Now().Add(-30 * time.Millisecond)
	gormAdapter.Trace(ctx, errorBegin, func() (string, int64) {
		return "INSERT INTO users (email, name) VALUES (?, ?)", 0
	}, errors.New("duplicate key error"))

	// 4. Record not found (should be ignored by default)
	fmt.Println("\n1.4 Record Not Found (ignored):")
	notFoundBegin := time.Now().Add(-20 * time.Millisecond)
	gormAdapter.Trace(ctx, notFoundBegin, func() (string, int64) {
		return "SELECT * FROM users WHERE id = ?", 0
	}, errors.New("record not found"))

	// 5. Direct logging methods
	fmt.Println("\n1.5 Direct GORM Logging:")
	gormAdapter.Info(ctx, "GORM connection established to database: %s", "postgres")
	gormAdapter.Warn(ctx, "GORM: Connection pool is %d%% full", 85)
	gormAdapter.Error(ctx, "GORM migration failed: %v", errors.New("table already exists"))

	// 6. Use different log levels
	fmt.Println("\n1.6 Different Log Levels:")
	silentAdapter := gormAdapter.LogMode(gorm.Silent)
	errorAdapter := gormAdapter.LogMode(gorm.Error)

	// Silent mode - should not log
	silentAdapter.Info(ctx, "This should not appear")

	// Error mode - should only log errors
	errorAdapter.Info(ctx, "This should not appear either")
	errorAdapter.Error(ctx, "This error should appear")

	fmt.Println()
}

func demonstrateKratosIntegration(coreLogger core.Logger) {
	fmt.Println("2. Kratos Integration Example")
	fmt.Println("=============================")

	// Create Kratos adapter
	kratosAdapter := kratos.NewKratosAdapter(coreLogger)

	// 1. Basic Kratos logging
	fmt.Println("\n2.1 Basic Kratos Logging:")
	kratosAdapter.Log(kratos.LevelInfo, "msg", "Service started", "port", 8080, "version", "1.0.0")
	kratosAdapter.Log(kratos.LevelWarn, "msg", "High memory usage", "memory_percent", 85.5)
	kratosAdapter.Log(kratos.LevelError, "msg", "Database connection failed", "error", "timeout", "retry_count", 3)

	// 2. Kratos logger with persistent fields
	fmt.Println("\n2.2 Kratos Logger With Persistent Fields:")
	serviceLogger := kratosAdapter.With("service", "user-service", "instance_id", "us-west-1-i-123456")
	serviceLogger.Log(kratos.LevelInfo, "msg", "Processing user request", "user_id", "user_789", "action", "login")
	serviceLogger.Log(kratos.LevelInfo, "msg", "Request completed", "user_id", "user_789", "duration_ms", 150)

	// 3. HTTP request logging
	fmt.Println("\n2.3 HTTP Request Logging:")
	kratosAdapter.LogRequest("GET", "/api/users", 200, 1500000000, "user_456")     // 1.5 seconds
	kratosAdapter.LogRequest("POST", "/api/users", 201, 2500000000, "user_789")    // 2.5 seconds
	kratosAdapter.LogRequest("GET", "/api/users/999", 404, 100000000, "user_123")  // 100ms, not found
	kratosAdapter.LogRequest("DELETE", "/api/users/1", 500, 5000000000, "admin_1") // 5 seconds, error

	// 4. Middleware logging
	fmt.Println("\n2.4 Middleware Logging:")
	kratosAdapter.LogMiddleware("auth-middleware", 5000000)        // 5ms
	kratosAdapter.LogMiddleware("cors-middleware", 1000000)        // 1ms
	kratosAdapter.LogMiddleware("rate-limit-middleware", 15000000) // 15ms

	// 5. Error logging
	fmt.Println("\n2.5 HTTP Error Logging:")
	kratosAdapter.LogError(errors.New("database connection lost"), "GET", "/api/users", 500)
	kratosAdapter.LogError(errors.New("validation failed"), "POST", "/api/users", 400)
	kratosAdapter.LogError(errors.New("unauthorized"), "DELETE", "/api/users/1", 401)

	// 6. Using Kratos helper
	fmt.Println("\n2.6 Kratos Helper:")
	helper := kratos.NewHelper(kratosAdapter)
	helper.Info("Helper info message")
	helper.Warn("Helper warning message")
	helper.Error("Helper error message")

	// 7. Using Kratos filter
	fmt.Println("\n2.7 Kratos Filter (only errors):")
	errorOnlyFilter := func(level kratos.Level, keyvals ...interface{}) bool {
		return level >= kratos.LevelError
	}
	filteredAdapter := kratos.NewKratosAdapter(coreLogger)
	filteredLogger := kratos.NewFilter(filteredAdapter, errorOnlyFilter)
	filteredLogger.Log(kratos.LevelInfo, "msg", "This should be filtered out")
	filteredLogger.Log(kratos.LevelError, "msg", "This error should pass through")

	// 8. Standard logger integration
	fmt.Println("\n2.8 Standard Logger Integration:")
	stdLogger := kratos.NewStdLogger(kratosAdapter)
	stdLogger.Print("Standard logger print message")
	stdLogger.Printf("Standard logger printf: %s = %d", "count", 42)
	stdLogger.Println("Standard logger println message")

	fmt.Println()
}
