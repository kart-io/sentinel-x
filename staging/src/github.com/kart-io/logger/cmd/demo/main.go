package main

import (
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== Unified Logger Demonstration ===")

	// Test 1: Slog Engine
	fmt.Println("1. Using Slog Engine:")
	slogOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP:        &option.OTLPOption{},
	}

	slogLogger, err := logger.New(slogOpt)
	if err != nil {
		panic(err)
	}

	slogLogger.Infow("User logged in", "user_id", "12345", "action", "login", "ip", "192.168.1.100")

	// Test 2: Zap Engine
	fmt.Println("\n2. Using Zap Engine:")
	zapOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP:        &option.OTLPOption{},
	}

	zapLogger, err := logger.New(zapOpt)
	if err != nil {
		panic(err)
	}

	zapLogger.Infow("User logged in", "user_id", "12345", "action", "login", "ip", "192.168.1.100")

	// Test 3: Global Logger (uses default Slog)
	fmt.Println("\n3. Using Global Logger:")
	logger.Infow("Global logger message", "service", "demo", "version", "1.0.0")

	// Test 4: Package-level convenience functions
	fmt.Println("\n4. Package-level convenience functions:")
	logger.Info("Simple info message")
	logger.Warnf("Warning with format: %s", "formatted text")

	// Test 5: Field Standardization - using various field name formats
	fmt.Println("\n5. Field Standardization Test:")
	childLogger := zapLogger.With("ts", "2023-01-01", "msg", "test-message", "trace.id", "abc123")
	childLogger.Info("Field standardization demo")

	fmt.Println("\n=== Demo Complete ===")
}
