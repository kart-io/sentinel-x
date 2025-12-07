package main

import (
	"fmt"
	"log"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== InitialFields Usage Example ===")

	// Create a logger configuration with initial fields
	opt := option.DefaultLogOption()

	// Method 1: Add fields using WithInitialFields (map)
	opt.WithInitialFields(map[string]interface{}{
		"service.name":    "example-service",
		"service.version": "v1.0.0",
	})

	// Method 2: Add individual fields using AddInitialField (fluent interface)
	opt.AddInitialField("environment", "development").
		AddInitialField("datacenter", "local").
		AddInitialField("instance_id", "example-001")

	// Method 3: You can also chain WithInitialFields and AddInitialField
	opt.WithInitialFields(map[string]interface{}{
		"team":    "platform",
		"project": "logging-demo",
	}).AddInitialField("build_id", "12345")

	// Show what fields we have configured
	fmt.Println("\nConfigured InitialFields:")
	fields := opt.GetInitialFields()
	for key, value := range fields {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Create logger with these initial fields
	serviceLogger, err := logger.New(opt)
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}

	fmt.Println("\nLogging examples (all will include InitialFields automatically):")
	fmt.Println()

	// All these log entries will automatically include the InitialFields
	serviceLogger.Info("Application started")

	serviceLogger.Infow("User action performed",
		"user_id", "user123",
		"action", "login",
		"success", true,
	)

	serviceLogger.Warnw("Performance warning",
		"operation", "database_query",
		"duration_ms", 1500,
		"threshold_ms", 1000,
	)

	serviceLogger.Errorw("Connection failed",
		"target", "redis://localhost:6379",
		"error", "connection refused",
		"retry_count", 3,
	)

	fmt.Println("\n=== Demonstrating field precedence ===")

	// InitialFields < With() < current log call fields
	childLogger := serviceLogger.With("service.name", "overridden-by-with")
	childLogger.Infow("Field precedence test",
		"service.version", "overridden-by-current-call", // This will override InitialField
		"additional_field", "only-in-this-log",
	)

	fmt.Println("\n=== Creating a new logger with different InitialFields ===")

	// Create another logger with different initial fields
	opt2 := option.DefaultLogOption().
		AddInitialField("service.name", "another-service").
		AddInitialField("service.version", "v2.0.0").
		AddInitialField("component", "worker")

	logger2, err := logger.New(opt2)
	if err != nil {
		log.Fatal("Failed to create second logger:", err)
	}

	logger2.Info("This is from a different service")
	logger2.Infow("Background job completed",
		"job_id", "job456",
		"duration", "2.3s",
	)

	fmt.Println("\n=== Example Complete ===")
}
