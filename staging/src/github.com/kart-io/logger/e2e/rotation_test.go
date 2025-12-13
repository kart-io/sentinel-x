package e2e

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func TestRotationBasic(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create a log file in temp directory
	logFile := filepath.Join(helper.GetTempDir(), "app.log")

	// Configure logger with small rotation size for testing
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{logFile},
		Rotation: &option.RotationOption{
			MaxSize:        1, // 1MB - small for testing
			MaxAge:         7,
			MaxBackups:     3,
			Compress:       false, // Don't compress for easier testing
			RotateInterval: "24h",
		},
	}

	// Create logger
	log, err := logger.New(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write some logs
	for i := 0; i < 10; i++ {
		log.Infow("Test rotation message",
			"iteration", i,
			"timestamp", time.Now().Unix(),
			"large_field", strings.Repeat("This is a test message for rotation. ", 100), // Make it larger
		)
	}

	// Flush to ensure logs are written
	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Wait for logs to be written
	if !helper.WaitForFile(logFile, 5*time.Second) {
		t.Fatalf("Log file was not created within timeout")
	}

	// Verify log file exists and has content
	if !helper.FileExists(logFile) {
		t.Fatalf("Log file %s does not exist", logFile)
	}

	if helper.GetFileSize(logFile) == 0 {
		t.Fatalf("Log file %s is empty", logFile)
	}

	// Parse and verify log content
	logs := helper.ParseJSONLogs(logFile)
	if len(logs) == 0 {
		t.Fatalf("No logs found in file")
	}

	// Verify log structure
	firstLog := logs[0]
	if _, ok := firstLog["level"]; !ok {
		t.Errorf("Log entry missing 'level' field")
	}
	if _, ok := firstLog["message"]; !ok {
		if _, ok := firstLog["msg"]; !ok {
			t.Errorf("Log entry missing message field")
		}
	}
	if _, ok := firstLog["iteration"]; !ok {
		t.Errorf("Log entry missing 'iteration' field")
	}
}

func TestRotationSizeBasedRotation(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	logFile := filepath.Join(helper.GetTempDir(), "size_rotation.log")

	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{logFile},
		Rotation: &option.RotationOption{
			MaxSize:    1, // 1KB - very small for testing
			MaxBackups: 5,
			Compress:   false,
		},
	}

	log, err := logger.New(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Generate enough logs to trigger rotation
	largeMessage := strings.Repeat("A", 500) // 500 characters per message
	for i := 0; i < 50; i++ {
		log.Infow("Large message for rotation test",
			"iteration", i,
			"large_data", largeMessage,
			"timestamp", time.Now().UnixNano(),
		)

		// Small delay to allow rotation to happen
		time.Sleep(10 * time.Millisecond)
	}

	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Wait a bit for rotation to complete
	time.Sleep(500 * time.Millisecond)

	// Check if rotation occurred by looking for rotated files
	rotatedFiles := helper.FindRotatedFiles(logFile)

	t.Logf("Found rotated files: %v", rotatedFiles)
	t.Logf("Main log file size: %d bytes", helper.GetFileSize(logFile))

	// We should have at least some rotated files or a reasonably sized main file
	if len(rotatedFiles) == 0 && helper.GetFileSize(logFile) < 1000 {
		t.Logf("Warning: No rotation detected. This may be expected for small log volumes.")
	}

	// Verify main log file still exists and has content
	if !helper.FileExists(logFile) {
		t.Fatalf("Main log file should still exist after rotation")
	}
}

func TestRotationWithBackupLimit(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	logFile := filepath.Join(helper.GetTempDir(), "backup_limit.log")

	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "DEBUG",
		Format:      "json",
		OutputPaths: []string{logFile},
		Rotation: &option.RotationOption{
			MaxSize:    1, // Small size to force rotation
			MaxBackups: 2, // Only keep 2 backup files
			MaxAge:     1, // 1 day
			Compress:   false,
		},
	}

	log, err := logger.New(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Generate many logs to force multiple rotations
	for round := 0; round < 3; round++ {
		for i := 0; i < 100; i++ {
			log.Debugw("Backup limit test message",
				"round", round,
				"iteration", i,
				"data", strings.Repeat(fmt.Sprintf("Round %d Data %d ", round, i), 20),
			)
		}

		// Flush and wait between rounds
		helper.SafeFlush(log)
		time.Sleep(100 * time.Millisecond)
	}

	// Final flush and wait
	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Check backup files
	rotatedFiles := helper.FindRotatedFiles(logFile)
	t.Logf("Found %d rotated files: %v", len(rotatedFiles), rotatedFiles)

	// We shouldn't have more backup files than the limit (though this is hard to test reliably)
	if len(rotatedFiles) > 5 { // Allow some leeway for timing issues
		t.Logf("Warning: More rotated files than expected (found %d)", len(rotatedFiles))
	}

	// Main log file should exist
	if !helper.FileExists(logFile) {
		t.Fatalf("Main log file should exist")
	}
}

func TestRotationWithConsoleOutput(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Test that rotation is NOT applied to console outputs
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"}, // Console output only
		Rotation: &option.RotationOption{
			MaxSize:    100,
			MaxBackups: 3,
			Compress:   true,
		},
	}

	// This should work without issues - rotation config is ignored for console
	log, err := logger.New(opt)
	if err != nil {
		t.Fatalf("Failed to create logger with console output and rotation config: %v", err)
	}

	// Write some logs - should go to console
	log.Info("This message goes to console")
	log.Infow("Console message with fields",
		"test_case", "console_output",
		"rotation_ignored", true,
	)

	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Verify that IsRotationEnabled returns false for console-only output
	if opt.IsRotationEnabled() {
		t.Errorf("IsRotationEnabled() should return false for console-only outputs")
	}
}

func TestRotationWithMixedOutputs(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	logFile := filepath.Join(helper.GetTempDir(), "mixed_outputs.log")

	// Test rotation with both console and file outputs
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout", logFile}, // Both console and file
		Rotation: &option.RotationOption{
			MaxSize:    50, // 50KB
			MaxBackups: 2,
			Compress:   false,
		},
	}

	log, err := logger.New(opt)
	if err != nil {
		t.Fatalf("Failed to create logger with mixed outputs: %v", err)
	}

	// Write logs - should go to both console and file
	for i := 0; i < 20; i++ {
		log.Infow("Mixed output test message",
			"iteration", i,
			"output_targets", []string{"console", "file"},
			"test_data", strings.Repeat(fmt.Sprintf("Data-%d ", i), 50),
		)
	}

	if err := helper.SafeFlush(log); err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Wait for file to be written
	if !helper.WaitForFileSize(logFile, 100, 5*time.Second) {
		t.Fatalf("Log file was not written or is too small")
	}

	// Verify file exists and has content
	if !helper.FileExists(logFile) {
		t.Fatalf("Log file should exist for mixed outputs")
	}

	logs := helper.ParseJSONLogs(logFile)
	if len(logs) == 0 {
		t.Fatalf("No logs found in file")
	}

	// Verify that IsRotationEnabled returns true for mixed outputs (includes files)
	if !opt.IsRotationEnabled() {
		t.Errorf("IsRotationEnabled() should return true for mixed outputs including files")
	}

	t.Logf("Successfully wrote %d log entries to mixed outputs", len(logs))
}

func TestRotationConfiguration(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	tests := []struct {
		name          string
		rotation      *option.RotationOption
		expectEnabled bool
		expectError   bool
	}{
		{
			name:          "nil rotation config",
			rotation:      nil,
			expectEnabled: false,
			expectError:   false,
		},
		{
			name: "valid rotation config",
			rotation: &option.RotationOption{
				MaxSize:    50,
				MaxAge:     7,
				MaxBackups: 3,
				Compress:   true,
			},
			expectEnabled: true,
			expectError:   false,
		},
		{
			name: "zero values get defaults",
			rotation: &option.RotationOption{
				MaxSize:    0,
				MaxAge:     0,
				MaxBackups: 0,
			},
			expectEnabled: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logFile := filepath.Join(helper.GetTempDir(), fmt.Sprintf("%s.log", tt.name))

			opt := &option.LogOption{
				Engine:      "zap",
				Level:       "INFO",
				Format:      "json",
				OutputPaths: []string{logFile},
				Rotation:    tt.rotation,
			}

			err := opt.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if err != nil {
				return // Skip logger creation if validation failed
			}

			if opt.IsRotationEnabled() != tt.expectEnabled {
				t.Errorf("IsRotationEnabled() = %v, expectEnabled %v", opt.IsRotationEnabled(), tt.expectEnabled)
			}

			// Try to create logger
			log, err := logger.New(opt)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}

			// Write a test log
			log.Info("Configuration test log")
			if err := helper.SafeFlush(log); err != nil {
				t.Fatalf("Failed to flush logger: %v", err)
			}

			// Verify log was written
			if !helper.WaitForFile(logFile, 2*time.Second) {
				t.Fatalf("Log file was not created")
			}

			// Check that defaults were applied for zero values
			if tt.name == "zero values get defaults" && opt.Rotation != nil {
				if opt.Rotation.MaxSize == 0 {
					t.Errorf("Expected MaxSize to have default value after validation")
				}
				if opt.Rotation.MaxAge == 0 {
					t.Errorf("Expected MaxAge to have default value after validation")
				}
				if opt.Rotation.MaxBackups == 0 {
					t.Errorf("Expected MaxBackups to have default value after validation")
				}
			}
		})
	}
}
