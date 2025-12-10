// Package main demonstrates the enhanced structured logging with context propagation
package main

import (
	"context"
	"fmt"

	applogger "github.com/kart-io/sentinel-x/pkg/infra/logger"
)

func main() {
	// Initialize logger
	opts := applogger.NewOptions()
	opts.Level = "DEBUG"
	opts.Format = "json"
	opts.Development = true

	if err := opts.Init(); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}

	fmt.Println("=== Enhanced Structured Logging Demo ===")

	// Example 1: Basic context fields
	fmt.Println("1. Basic Context Fields:")
	ctx := context.Background()
	ctx = applogger.WithRequestID(ctx, "req-12345")
	ctx = applogger.WithUserID(ctx, "user-67890")
	ctx = applogger.WithTenantID(ctx, "tenant-001")

	log := applogger.GetLogger(ctx)
	log.Infow("User request processed", "action", "login", "duration_ms", 123)

	// Example 2: Trace correlation
	fmt.Println("\n2. Trace Correlation:")
	ctx2 := context.Background()
	ctx2 = applogger.WithTraceID(ctx2, "trace-abc123")
	ctx2 = applogger.WithSpanID(ctx2, "span-xyz789")

	log2 := applogger.GetLogger(ctx2)
	log2.Infow("Database query executed", "query", "SELECT * FROM users", "rows", 42)

	// Example 3: Error logging
	fmt.Println("\n3. Error Logging:")
	ctx3 := context.Background()
	ctx3 = applogger.WithRequestID(ctx3, "req-99999")

	err := fmt.Errorf("database connection failed")
	ctx3 = applogger.WithError(ctx3, err)
	ctx3 = applogger.WithErrorCode(ctx3, "DB_CONN_ERR")

	log3 := applogger.GetLogger(ctx3)
	log3.Errorw("Failed to process request")

	// Example 4: Multiple fields at once
	fmt.Println("\n4. Multiple Fields:")
	ctx4 := context.Background()
	ctx4 = applogger.WithFields(ctx4,
		"request_id", "req-11111",
		"user_id", "user-22222",
		"api_version", "v2",
		"region", "us-east-1",
	)

	log4 := applogger.GetLogger(ctx4)
	log4.Infow("API request completed", "status", 200, "response_time_ms", 45)

	// Example 5: Context logger
	fmt.Println("\n5. Context Logger:")
	ctx5 := context.Background()
	ctx5 = applogger.WithRequestID(ctx5, "req-context")

	ctxLogger := applogger.NewContextLogger(ctx5)
	ctxLogger.Infow("Using context logger", "feature", "enhanced_logging")
	ctxLogger.Debugw("Debug information", "debug_level", 3)

	// Example 6: Error chain logging
	fmt.Println("\n6. Error Chain Logging:")
	ctx6 := context.Background()
	ctx6 = applogger.WithRequestID(ctx6, "req-error-chain")

	baseErr := fmt.Errorf("connection timeout")
	wrappedErr := fmt.Errorf("failed to fetch data: %w", baseErr)

	applogger.LogErrorChain(ctx6, "Request failed with error chain", wrappedErr, false)

	// Example 7: Convenience functions
	fmt.Println("\n7. Convenience Functions:")
	ctx7 := context.Background()
	ctx7 = applogger.WithRequestID(ctx7, "req-convenience")

	applogger.LogInfo(ctx7, "Information message", "info_type", "system")
	applogger.LogDebug(ctx7, "Debug message", "debug_flag", true)
	applogger.LogWarn(ctx7, "Warning message", "warning_level", "medium")

	fmt.Println("\n=== Demo Complete ===")
}
