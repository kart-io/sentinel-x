package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func TestYAMLConfigBasicRotation(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Read the basic rotation config
	configFile := "./testdata/basic_rotation.yaml"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Parse YAML config - start with empty config to avoid default value override
	var config struct {
		Log *option.LogOption `yaml:"log"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse YAML config: %v", err)
	}

	// Ensure we have a valid config object
	if config.Log == nil {
		t.Fatalf("Failed to parse log configuration from YAML")
	}

	// Update output paths to use temp directory
	tempLogFile := filepath.Join(helper.GetTempDir(), "app.log")
	config.Log.OutputPaths = []string{"stdout", tempLogFile}

	// Validate the configuration
	if err := config.Log.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Verify rotation configuration was parsed correctly
	if config.Log.Rotation == nil {
		t.Fatalf("Rotation configuration should not be nil")
	}

	rotation := config.Log.Rotation
	if rotation.MaxSize != 10 {
		t.Errorf("Expected MaxSize 10, got %d", rotation.MaxSize)
	}
	if rotation.MaxAge != 7 {
		t.Errorf("Expected MaxAge 7, got %d", rotation.MaxAge)
	}
	if rotation.MaxBackups != 5 {
		t.Errorf("Expected MaxBackups 5, got %d", rotation.MaxBackups)
	}
	if !rotation.Compress {
		t.Errorf("Expected Compress to be true")
	}
	if rotation.RotateInterval != "24h" {
		t.Errorf("Expected RotateInterval '24h', got %s", rotation.RotateInterval)
	}

	// Verify other configuration
	if config.Log.Engine != "zap" {
		t.Errorf("Expected engine 'zap', got %s", config.Log.Engine)
	}
	if config.Log.Level != "info" {
		t.Errorf("Expected level 'info', got %s", config.Log.Level)
	}

	// Create logger from config
	log, err := logger.New(config.Log)
	if err != nil {
		t.Fatalf("Failed to create logger from config: %v", err)
	}

	// Test logging
	log.Infow("YAML config test message",
		"config_file", "basic_rotation.yaml",
		"test_case", "yaml_parsing",
	)

	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Verify log file was created
	if !helper.WaitForFile(tempLogFile, 5*time.Second) {
		t.Fatalf("Log file was not created")
	}

	// Verify log content
	logs := helper.ParseJSONLogs(tempLogFile)
	if len(logs) == 0 {
		t.Fatalf("No logs found")
	}

	// Check that log contains expected fields
	helper.AssertLogContains(tempLogFile, []LogAssertion{
		{
			Level:   "info",
			Message: "YAML config test message",
			Fields: map[string]interface{}{
				"config_file": "basic_rotation.yaml",
				"test_case":   "yaml_parsing",
			},
		},
	})
}

func TestYAMLConfigRotationWithOTLP(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	configFile := "./testdata/rotation_with_otlp.yaml"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var config struct {
		Log *option.LogOption `yaml:"log"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse YAML config: %v", err)
	}

	// Ensure we have a valid config object
	if config.Log == nil {
		t.Fatalf("Failed to parse log configuration from YAML")
	}

	// Update paths for temp directory
	tempLogFile := filepath.Join(helper.GetTempDir(), "debug.log")
	config.Log.OutputPaths = []string{"stdout", tempLogFile}

	if err := config.Log.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Verify both OTLP and rotation are configured
	if !config.Log.IsOTLPEnabled() {
		t.Errorf("OTLP should be enabled")
	}

	if !config.Log.IsRotationEnabled() {
		t.Errorf("Rotation should be enabled")
	}

	// Verify rotation config
	rotation := config.Log.Rotation
	if rotation.MaxSize != 50 {
		t.Errorf("Expected MaxSize 50, got %d", rotation.MaxSize)
	}
	if rotation.RotateInterval != "12h" {
		t.Errorf("Expected RotateInterval '12h', got %s", rotation.RotateInterval)
	}

	// Verify OTLP config
	if config.Log.OTLPEndpoint != "http://localhost:4317" {
		t.Errorf("Expected OTLP endpoint 'http://localhost:4317', got %s", config.Log.OTLPEndpoint)
	}

	// Create logger (OTLP may fail but that's OK for this test)
	log, err := logger.New(config.Log)
	if err != nil {
		t.Logf("Logger creation failed (expected if OTLP is not available): %v", err)

		// Try without OTLP
		config.Log.OTLPEndpoint = ""
		config.Log.OTLP = nil
		log, err = logger.New(config.Log)
		if err != nil {
			t.Fatalf("Failed to create logger without OTLP: %v", err)
		}
	}

	// Test logging
	log.Debugw("YAML config with OTLP test",
		"config_file", "rotation_with_otlp.yaml",
		"otlp_configured", config.Log.OTLPEndpoint != "",
		"rotation_configured", config.Log.Rotation != nil,
	)

	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Verify log file
	if !helper.WaitForFile(tempLogFile, 5*time.Second) {
		t.Fatalf("Log file was not created")
	}

	logs := helper.ParseJSONLogs(tempLogFile)
	if len(logs) == 0 {
		t.Fatalf("No logs found")
	}
}

func TestYAMLConfigConsoleOnly(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	configFile := "./testdata/console_only.yaml"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var config struct {
		Log *option.LogOption `yaml:"log"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse YAML config: %v", err)
	}

	// Ensure we have a valid config object
	if config.Log == nil {
		t.Fatalf("Failed to parse log configuration from YAML")
	}

	if err := config.Log.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Verify rotation is configured but not enabled (console only)
	if config.Log.Rotation == nil {
		t.Fatalf("Rotation config should be parsed even for console-only")
	}

	if config.Log.IsRotationEnabled() {
		t.Errorf("Rotation should be disabled for console-only outputs")
	}

	// Verify console format
	if config.Log.Format != "console" {
		t.Errorf("Expected format 'console', got %s", config.Log.Format)
	}

	// Create logger - should work fine
	log, err := logger.New(config.Log)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test logging to console
	log.Warn("Console-only test message")
	log.Warnw("Console test with fields",
		"output_type", "console_only",
		"rotation_ignored", true,
	)

	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}
}

func TestYAMLConfigInvalidRotation(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	configFile := "./testdata/invalid_rotation.yaml"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var config struct {
		Log *option.LogOption `yaml:"log"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse YAML config: %v", err)
	}

	// Ensure we have a valid config object
	if config.Log == nil {
		t.Fatalf("Failed to parse log configuration from YAML")
	}

	// Update path for temp directory
	tempLogFile := filepath.Join(helper.GetTempDir(), "invalid.log")
	config.Log.OutputPaths = []string{tempLogFile}

	// Configuration validation should fail due to invalid rotation values
	err = config.Log.Validate()
	if err == nil {
		t.Fatalf("Expected validation to fail for invalid rotation config")
	}

	t.Logf("Validation failed as expected: %v", err)

	// Verify the error message contains information about the invalid fields
	errStr := err.Error()
	expectedErrors := []string{"max_size", "max_age", "max_backups", "rotate_interval"}

	foundError := false
	for _, expected := range expectedErrors {
		if contains(errStr, expected) {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Errorf("Expected error message to contain one of %v, got: %s", expectedErrors, errStr)
	}
}

func TestYAMLConfigDefaultValues(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	configFile := "./testdata/default_values.yaml"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var config struct {
		Log *option.LogOption `yaml:"log"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse YAML config: %v", err)
	}

	// Ensure we have a valid config object
	if config.Log == nil {
		t.Fatalf("Failed to parse log configuration from YAML")
	}

	// Update path for temp directory
	tempLogFile := filepath.Join(helper.GetTempDir(), "defaults.log")
	config.Log.OutputPaths = []string{tempLogFile}

	// Before validation, check that values are zero
	rotation := config.Log.Rotation
	if rotation.MaxSize != 0 {
		t.Errorf("Expected MaxSize to be 0 before validation, got %d", rotation.MaxSize)
	}

	// Validation should succeed and apply defaults
	if err := config.Log.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// After validation, check that defaults were applied
	if rotation.MaxSize == 0 {
		t.Errorf("Expected MaxSize to have default value after validation, still 0")
	}
	if rotation.MaxAge == 0 {
		t.Errorf("Expected MaxAge to have default value after validation, still 0")
	}
	if rotation.MaxBackups == 0 {
		t.Errorf("Expected MaxBackups to have default value after validation, still 0")
	}
	if rotation.RotateInterval == "" {
		t.Errorf("Expected RotateInterval to have default value after validation, still empty")
	}

	t.Logf("Default values applied: MaxSize=%d, MaxAge=%d, MaxBackups=%d, RotateInterval=%s",
		rotation.MaxSize, rotation.MaxAge, rotation.MaxBackups, rotation.RotateInterval)

	// Create logger
	log, err := logger.New(config.Log)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test logging
	log.Infow("Default values test",
		"max_size_applied", rotation.MaxSize,
		"max_age_applied", rotation.MaxAge,
		"max_backups_applied", rotation.MaxBackups,
	)

	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Verify log file
	if !helper.WaitForFile(tempLogFile, 5*time.Second) {
		t.Fatalf("Log file was not created")
	}

	logs := helper.ParseJSONLogs(tempLogFile)
	if len(logs) == 0 {
		t.Fatalf("No logs found")
	}
}

func TestYAMLConfigCompleteScenario(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create a comprehensive config in memory
	config := &option.LogOption{
		Engine:            "zap",
		Level:             "debug",
		Format:            "json",
		OutputPaths:       []string{"stdout", filepath.Join(helper.GetTempDir(), "complete.log")},
		Development:       true,
		DisableCaller:     false,
		DisableStacktrace: false,
		Rotation: &option.RotationOption{
			MaxSize:        25,
			MaxAge:         10,
			MaxBackups:     8,
			Compress:       true,
			RotateInterval: "6h",
		},
	}

	// Validate config
	if err := config.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Verify all settings
	if !config.IsRotationEnabled() {
		t.Errorf("Rotation should be enabled")
	}

	if config.Engine != "zap" {
		t.Errorf("Expected engine 'zap', got %s", config.Engine)
	}

	if config.Development != true {
		t.Errorf("Expected development mode to be true")
	}

	// Create logger
	log, err := logger.New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test comprehensive logging
	log.Debugw("Complete scenario debug",
		"scenario", "comprehensive_test",
		"rotation_enabled", config.IsRotationEnabled(),
	)

	log.Infow("Complete scenario info",
		"engine", config.Engine,
		"development", config.Development,
		"format", config.Format,
	)

	log.Warnw("Complete scenario warning",
		"rotation_config", map[string]interface{}{
			"max_size":    config.Rotation.MaxSize,
			"max_age":     config.Rotation.MaxAge,
			"max_backups": config.Rotation.MaxBackups,
			"compress":    config.Rotation.Compress,
		},
	)

	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Verify log file
	logFile := filepath.Join(helper.GetTempDir(), "complete.log")
	if !helper.WaitForFile(logFile, 5*time.Second) {
		t.Fatalf("Log file was not created")
	}

	logs := helper.ParseJSONLogs(logFile)
	if len(logs) < 3 {
		t.Fatalf("Expected at least 3 log entries, got %d", len(logs))
	}

	// Verify log content
	helper.AssertLogContains(logFile, []LogAssertion{
		{Level: "debug", Message: "Complete scenario debug"},
		{Level: "info", Message: "Complete scenario info"},
		{Level: "warn", Message: "Complete scenario warning"},
	})

	t.Logf("Complete scenario test passed with %d log entries", len(logs))
}

// Helper function for string contains check (simplified version)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(substr) > 0 && strings.Contains(s, substr)))
}
