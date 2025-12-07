package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/factory"
	"github.com/kart-io/logger/option"
	"github.com/kart-io/logger/reload"
)

func main() {
	fmt.Println("=== Dynamic Configuration Reload Example ===\n")

	// Create temporary directory for config file
	tmpDir, err := os.MkdirTemp("", "logger-reload-example")
	if err != nil {
		panic(fmt.Sprintf("Failed to create temp dir: %v", err))
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "logger-config.json")
	fmt.Printf("Config file location: %s\n", configFile)

	// 1. Create initial configuration
	fmt.Println("\n1. Setting up initial configuration...")
	initialConfig := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
	}

	// Write initial config to file
	if err := writeConfigFile(configFile, initialConfig); err != nil {
		panic(fmt.Sprintf("Failed to write initial config: %v", err))
	}

	// Create logger with initial config
	factory := factory.NewLoggerFactory(initialConfig)
	coreLogger, err := factory.CreateLogger()
	if err != nil {
		panic(fmt.Sprintf("Failed to create initial logger: %v", err))
	}

	// 2. Setup configuration reloader
	fmt.Println("2. Setting up configuration reloader...")
	reloadConfig := &reload.ReloadConfig{
		ConfigFile:           configFile,
		Triggers:             reload.TriggerAll, // Enable all triggers
		ValidateBeforeReload: true,
		ReloadTimeout:        10 * time.Second,
		BackupOnReload:       true,
		BackupRetention:      3,
		Callback: func(oldConfig, newConfig *option.LogOption) error {
			fmt.Printf("📄 Config reloaded: %s->%s, %s->%s, %s->%s\n",
				oldConfig.Engine, newConfig.Engine,
				oldConfig.Level, newConfig.Level,
				oldConfig.Format, newConfig.Format,
			)
			return nil
		},
		Logger: coreLogger,
	}

	reloader, err := reload.NewConfigReloader(reloadConfig, initialConfig, factory)
	if err != nil {
		panic(fmt.Sprintf("Failed to create reloader: %v", err))
	}

	// Start the reloader
	if err := reloader.Start(); err != nil {
		panic(fmt.Sprintf("Failed to start reloader: %v", err))
	}
	defer reloader.Stop()

	// 3. Demonstrate initial logging
	fmt.Println("\n3. Initial logging with slog engine...")
	logSamples(coreLogger, "Initial")

	// 4. Demonstrate file-based reload
	fmt.Println("\n4. Demonstrating file-based configuration reload...")
	time.Sleep(500 * time.Millisecond) // Allow file watcher to initialize

	// Update config file
	newConfig1 := &option.LogOption{
		Engine:      "zap",
		Level:       "DEBUG",
		Format:      "console",
		OutputPaths: []string{"stdout"},
		Development: true,
	}

	fmt.Println("Updating configuration file...")
	if err := writeConfigFile(configFile, newConfig1); err != nil {
		panic(fmt.Sprintf("Failed to write updated config: %v", err))
	}

	// Wait for file change detection
	time.Sleep(1 * time.Second)

	// Test logging with new config
	fmt.Println("Logging with updated configuration (should be zap engine, DEBUG level, text format):")
	logSamples(coreLogger, "After File Reload")

	// 5. Demonstrate API-triggered reload
	fmt.Println("\n5. Demonstrating API-triggered configuration reload...")
	newConfig2 := &option.LogOption{
		Engine:            "slog",
		Level:             "WARN",
		Format:            "json",
		OutputPaths:       []string{"stdout"},
		Development:       false,
		DisableCaller:     true,
		DisableStacktrace: true,
	}

	fmt.Println("Triggering reload via API...")
	if err := reloader.TriggerReload(newConfig2); err != nil {
		fmt.Printf("Failed to trigger reload: %v\n", err)
	} else {
		time.Sleep(500 * time.Millisecond)
		fmt.Println("Logging with API-reloaded configuration (should be slog engine, WARN level, no debug/info):")
		logSamples(coreLogger, "After API Reload")
	}

	// 6. Demonstrate signal-based reload
	fmt.Println("\n6. Demonstrating signal-based configuration reload...")

	// Update config file first
	newConfig3 := &option.LogOption{
		Engine:      "zap",
		Level:       "ERROR",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
	}

	if err := writeConfigFile(configFile, newConfig3); err != nil {
		panic(fmt.Sprintf("Failed to write config for signal reload: %v", err))
	}

	fmt.Println("Sending SIGUSR1 signal for configuration reload...")
	if err := syscall.Kill(os.Getpid(), syscall.SIGUSR1); err != nil {
		fmt.Printf("Failed to send signal: %v\n", err)
	} else {
		time.Sleep(1 * time.Second)
		fmt.Println("Logging with signal-reloaded configuration (should be zap engine, ERROR level only):")
		logSamples(coreLogger, "After Signal Reload")
	}

	// 7. Demonstrate backup and rollback
	fmt.Println("\n7. Demonstrating backup and rollback functionality...")
	backups := reloader.GetBackupConfigs()
	fmt.Printf("Current backup count: %d\n", len(backups))

	fmt.Println("Rolling back to previous configuration...")
	if err := reloader.RollbackToPrevious(); err != nil {
		fmt.Printf("Failed to rollback: %v\n", err)
	} else {
		time.Sleep(500 * time.Millisecond)
		fmt.Println("Logging with rolled-back configuration:")
		logSamples(coreLogger, "After Rollback")
	}

	// 8. Show current configuration
	fmt.Println("\n8. Current configuration summary...")
	currentConfig := reloader.GetCurrentConfig()
	fmt.Printf("Engine: %s, Level: %s, Format: %s, Development: %v\n",
		currentConfig.Engine, currentConfig.Level, currentConfig.Format, currentConfig.Development)

	// 9. Demonstrate validation failure
	fmt.Println("\n9. Demonstrating configuration validation...")

	// Create invalid config (empty engine)
	invalidConfig := &option.LogOption{
		Engine: "", // Invalid empty engine
		Level:  "INFO",
		Format: "json",
	}

	fmt.Println("Attempting to reload with invalid configuration (empty engine)...")
	if err := reloader.TriggerReload(invalidConfig); err != nil {
		fmt.Printf("Reload failed as expected: %v\n", err)
	} else {
		time.Sleep(500 * time.Millisecond)
		fmt.Println("Invalid config should have been rejected, current config unchanged:")
		currentConfig = reloader.GetCurrentConfig()
		fmt.Printf("Engine: %s (should not be empty)\n", currentConfig.Engine)
	}

	fmt.Println("\n=== Dynamic Configuration Reload Example Complete ===")
}

func logSamples(logger core.Logger, prefix string) {
	logger.Debug("Debug message", "prefix", prefix, "level", "debug")
	logger.Info("Info message", "prefix", prefix, "level", "info")
	logger.Warn("Warning message", "prefix", prefix, "level", "warn")
	logger.Error("Error message", "prefix", prefix, "level", "error")

	logger.Debugf("[%s] Debug formatted message: %d", prefix, 123)
	logger.Infof("[%s] Info formatted message: %s", prefix, "test")
	logger.Warnf("[%s] Warning formatted message: %v", prefix, true)
	logger.Errorf("[%s] Error formatted message: %f", prefix, 3.14)

	logger.Debugw("Debug structured message", "prefix", prefix, "type", "structured", "counter", 1)
	logger.Infow("Info structured message", "prefix", prefix, "type", "structured", "counter", 2)
	logger.Warnw("Warning structured message", "prefix", prefix, "type", "structured", "counter", 3)
	logger.Errorw("Error structured message", "prefix", prefix, "type", "structured", "counter", 4)
}

func writeConfigFile(filename string, config *option.LogOption) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}
