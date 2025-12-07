package main

import (
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/option"
	"github.com/spf13/pflag"
)

func main() {
	fmt.Println("=== Configuration Examples ===\n")

	// Example 1: Basic Configuration
	demonstrateBasicConfiguration()

	// Example 2: Command Line Flags Integration
	demonstrateCommandLineFlags()

	// Example 3: Environment-specific Configuration
	demonstrateEnvironmentConfigs()

	// Example 4: Custom Output Paths
	demonstrateOutputConfiguration()

	// Example 5: Level Configuration
	demonstrateLevelConfiguration()

	fmt.Println("\n=== Configuration Examples Complete ===")
}

// demonstrateBasicConfiguration shows basic configuration options
func demonstrateBasicConfiguration() {
	fmt.Println("1. Basic Configuration Options")
	fmt.Println("==============================")

	// Minimal configuration - uses defaults
	minimalOpt := option.DefaultLogOption()

	minimalLogger, err := logger.New(minimalOpt)
	if err != nil {
		panic(err)
	}

	fmt.Println("1.1 Minimal configuration (uses defaults):")
	minimalLogger.Info("Minimal configuration example")

	// Full configuration - all options specified
	fullOpt := &option.LogOption{
		Engine:            "zap",
		Level:             "DEBUG",
		Format:            "json",
		OutputPaths:       []string{"stdout"},
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),          // Disabled by default - enable if you have an OTLP collector
			Endpoint: "http://127.0.0.1:4317", // gRPC endpoint for configuration demo
			Protocol: "grpc",
			Timeout:  10 * time.Second,
			Headers: map[string]string{
				"service.name":    "configuration-demo",
				"service.version": "1.0.0",
				"environment":     "development",
			},
		},
	}

	fullLogger, err := logger.New(fullOpt)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n1.2 Full configuration (all options specified):")
	fullLogger.Infow("Full configuration example",
		"engine", "zap",
		"format", "json",
		"development", false,
	)

	fmt.Println()
}

// demonstrateCommandLineFlags shows pflag integration
func demonstrateCommandLineFlags() {
	fmt.Println("2. Command Line Flags Integration")
	fmt.Println("==================================")

	// Create flag set
	fs := pflag.NewFlagSet("logger-demo", pflag.ContinueOnError)

	// Create LogOption and add flags
	opt := option.DefaultLogOption()
	opt.AddFlags(fs)

	// Parse simulated command line arguments
	args := []string{
		"--engine", "zap",
		"--level", "WARN",
		"--format", "console",
		"--development", "true",
	}

	fmt.Printf("Simulating command line: %v\n", args)
	err := fs.Parse(args)
	if err != nil {
		panic(err)
	}

	// Create logger with parsed flags
	flagLogger, err := logger.New(opt)
	if err != nil {
		panic(err)
	}

	fmt.Println("\nLogger created with command line flags:")
	flagLogger.Info("This won't show - level is WARN")
	flagLogger.Warn("This will show - matches WARN level")
	flagLogger.Errorw("Error with context",
		"parsed_engine", opt.Engine,
		"parsed_level", opt.Level,
		"parsed_format", opt.Format,
	)

	fmt.Println()
}

// demonstrateEnvironmentConfigs shows different environment configurations
func demonstrateEnvironmentConfigs() {
	fmt.Println("3. Environment-specific Configuration")
	fmt.Println("=====================================")

	// Development environment configuration
	devOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "DEBUG",
		Format:      "console",
		OutputPaths: []string{"stdout"},
		Development: true,
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),          // Usually disabled in development
			Endpoint: "http://localhost:4317", // Local development collector
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "config-demo-dev",
				"environment":  "development",
			},
		},
	}

	devLogger, err := logger.New(devOpt)
	if err != nil {
		panic(err)
	}

	fmt.Println("3.1 Development environment:")
	devLogger.Debug("Debug message visible in development")
	devLogger.Infow("Development info",
		"environment", "development",
		"debug_enabled", true,
		"format", "console",
	)

	// Staging environment configuration
	stagingOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),                          // Can be enabled in staging for testing
			Endpoint: "https://otlp-staging.company.com:4317", // Staging OTLP endpoint
			Protocol: "grpc",
			Timeout:  5 * time.Second,
			Headers: map[string]string{
				"service.name":    "config-demo-staging",
				"service.version": "1.0.0",
				"environment":     "staging",
			},
		},
	}

	stagingLogger, err := logger.New(stagingOpt)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n3.2 Staging environment:")
	stagingLogger.Debug("Debug message hidden in staging")
	stagingLogger.Infow("Staging info",
		"environment", "staging",
		"debug_enabled", false,
		"format", "json",
	)

	// Production environment configuration
	prodOpt := &option.LogOption{
		Engine:            "zap",
		Level:             "WARN",
		Format:            "json",
		OutputPaths:       []string{"stdout"},
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),                       // Enable in production for observability
			Endpoint: "https://otlp-prod.company.com:4317", // Production OTLP endpoint
			Protocol: "grpc",
			Timeout:  10 * time.Second,
			Headers: map[string]string{
				"service.name":    "config-demo-production",
				"service.version": "2.1.0",
				"environment":     "production",
				"team":            "platform",
				"region":          "us-west-2",
			},
		},
	}

	prodLogger, err := logger.New(prodOpt)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n3.3 Production environment:")
	prodLogger.Debug("Debug message hidden in production")
	prodLogger.Info("Info message hidden in production")
	prodLogger.Warnw("Production warning",
		"environment", "production",
		"level", "WARN",
		"stacktrace_enabled", true,
	)

	fmt.Println()
}

// demonstrateOutputConfiguration shows different output configurations
func demonstrateOutputConfiguration() {
	fmt.Println("4. Output Path Configuration")
	fmt.Println("============================")

	// Standard output
	stdoutOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "config-demo-stdout",
			},
		},
	}

	stdoutLogger, err := logger.New(stdoutOpt)
	if err != nil {
		panic(err)
	}

	fmt.Println("4.1 Standard output:")
	stdoutLogger.Info("Message to stdout")

	// Standard error
	stderrOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stderr"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "config-demo-stderr",
			},
		},
	}

	stderrLogger, err := logger.New(stderrOpt)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n4.2 Standard error:")
	stderrLogger.Info("Message to stderr")

	// Multiple outputs (stdout + file)
	// Note: We'll use /tmp for the example, but in production use appropriate paths
	multiOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout", "/tmp/app.log"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "config-demo-multi-output",
			},
		},
	}

	multiLogger, err := logger.New(multiOpt)
	if err != nil {
		// If file creation fails, fallback to stdout only
		fmt.Printf("Warning: Could not create file logger: %v\n", err)
		multiOpt.OutputPaths = []string{"stdout"}
		multiLogger, err = logger.New(multiOpt)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("\n4.3 Multiple outputs (stdout + file):")
	multiLogger.Infow("Message to multiple outputs",
		"destinations", multiOpt.OutputPaths,
		"note", "This message appears in both stdout and file",
	)

	fmt.Println()
}

// demonstrateLevelConfiguration shows dynamic level configuration
func demonstrateLevelConfiguration() {
	fmt.Println("5. Level Configuration")
	fmt.Println("======================")

	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}

	for _, level := range levels {
		fmt.Printf("\n5.%d Testing level: %s\n", getlevelIndex(level)+1, level)

		opt := &option.LogOption{
			Engine:      "slog",
			Level:       level,
			Format:      "json",
			OutputPaths: []string{"stdout"},
			OTLP: &option.OTLPOption{
				Enabled:  boolPtr(false),
				Endpoint: "http://127.0.0.1:4317",
				Protocol: "grpc",
				Headers: map[string]string{
					"service.name": "config-demo-level-test",
					"test_level":   level,
				},
			},
		}

		levelLogger, err := logger.New(opt)
		if err != nil {
			panic(err)
		}

		// Test all levels to see which ones are visible
		levelLogger.Debug("DEBUG level message")
		levelLogger.Info("INFO level message")
		levelLogger.Warn("WARN level message")
		levelLogger.Error("ERROR level message")
	}

	// Dynamic level change example
	fmt.Println("\n5.5 Dynamic level changes:")
	dynamicOpt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),
			Endpoint: "http://127.0.0.1:4317",
			Protocol: "grpc",
			Headers: map[string]string{
				"service.name": "config-demo-dynamic-level",
			},
		},
	}

	dynamicLogger, err := logger.New(dynamicOpt)
	if err != nil {
		panic(err)
	}

	fmt.Println("Initial level: INFO")
	dynamicLogger.Debug("This debug won't show")
	dynamicLogger.Info("This info will show")

	// Note: SetLevel changes the internal level but may not affect the underlying
	// engine's level in all cases. This is a limitation of the current implementation.
	fmt.Println("\nChanging level to DEBUG (note: may not affect all engines):")
	dynamicLogger.SetLevel(core.DebugLevel)
	dynamicLogger.Debug("This debug might show now")
	dynamicLogger.Info("This info will still show")

	fmt.Println()
}

// Helper function to get level index
func getlevelIndex(level string) int {
	levels := map[string]int{
		"DEBUG": 0,
		"INFO":  1,
		"WARN":  2,
		"ERROR": 3,
	}
	return levels[level]
}

// Helper function to create boolean pointers
func boolPtr(b bool) *bool {
	return &b
}
