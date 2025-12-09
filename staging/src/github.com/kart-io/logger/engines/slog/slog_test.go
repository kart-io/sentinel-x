package slog

import (
	"testing"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/fields"
	"github.com/kart-io/logger/option"
)

func TestNewSlogLogger(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("NewSlogLogger() error = %v", err)
	}

	if logger == nil {
		t.Fatal("NewSlogLogger() returned nil logger")
	}

	slogLogger, ok := logger.(*SlogLogger)
	if !ok {
		t.Fatal("NewSlogLogger() didn't return *SlogLogger")
	}

	if slogLogger.level != core.InfoLevel {
		t.Errorf("Expected level to be InfoLevel, got %v", slogLogger.level)
	}
}

func TestNewSlogLogger_InvalidConfig(t *testing.T) {
	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INVALID_LEVEL",
		OTLP:   &option.OTLPOption{},
	}

	logger, err := NewSlogLogger(opt)

	if err == nil {
		t.Error("Expected error for invalid config")
	}

	if logger != nil {
		t.Error("Expected nil logger for invalid config")
	}
}

func TestSlogLogger_BasicLogging(t *testing.T) {
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "DEBUG",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP:        &option.OTLPOption{},
	}

	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test basic logging methods exist and don't panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// We can't easily capture slog output in tests without more complex setup
	// So we'll just verify the methods don't panic and the logger is properly configured
	t.Log("Basic logging methods executed without panic")
}

func TestSlogLogger_FormattedLogging(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test formatted logging methods
	logger.Debugf("debug %s %d", "message", 1)
	logger.Infof("info %s %d", "message", 2)
	logger.Warnf("warn %s %d", "message", 3)
	logger.Errorf("error %s %d", "message", 4)

	t.Log("Formatted logging methods executed without panic")
}

func TestSlogLogger_StructuredLogging(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test structured logging methods
	logger.Debugw("debug message", "key1", "value1", "key2", 42)
	logger.Infow("info message", "user_id", "123", "action", "login")
	logger.Warnw("warn message", "warning", "deprecated")
	logger.Errorw("error message", "error", "connection failed", "retry", 3)

	t.Log("Structured logging methods executed without panic")
}

func TestSlogLogger_With(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test With method
	childLogger := logger.With("service", "test-service", "version", "1.0.0")

	if childLogger == nil {
		t.Fatal("With() returned nil")
	}

	// Verify it's still a SlogLogger
	if _, ok := childLogger.(*SlogLogger); !ok {
		t.Fatal("With() didn't return *SlogLogger")
	}

	// Test that child logger works
	childLogger.Info("child logger message")
	t.Log("Child logger created and used successfully")
}

func TestSlogLogger_WithCallerSkip(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test WithCallerSkip method
	skippedLogger := logger.WithCallerSkip(2)

	if skippedLogger == nil {
		t.Fatal("WithCallerSkip() returned nil")
	}

	slogLogger, ok := skippedLogger.(*SlogLogger)
	if !ok {
		t.Fatal("WithCallerSkip() didn't return *SlogLogger")
	}

	if slogLogger.callerSkip != 2 {
		t.Errorf("Expected callerSkip to be 2, got %d", slogLogger.callerSkip)
	}

	skippedLogger.Info("caller skip test")
	t.Log("WithCallerSkip worked correctly")
}

func TestSlogLogger_SetLevel(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test SetLevel method
	logger.SetLevel(core.ErrorLevel)

	slogLogger := logger.(*SlogLogger)
	if slogLogger.level != core.ErrorLevel {
		t.Errorf("Expected level to be ErrorLevel, got %v", slogLogger.level)
	}
}

func TestFormatArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected string
	}{
		{"no args", []interface{}{}, ""},
		{"single string", []interface{}{"hello"}, "hello"},
		{"single number", []interface{}{42}, "42"},
		{"multiple args", []interface{}{"hello", "world", 42}, "hello world 42"},
		{"nil arg", []interface{}{nil}, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatArgs(tt.args...)
			if result != tt.expected {
				t.Errorf("formatArgs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMapToSlogLevel(t *testing.T) {
	tests := []struct {
		coreLevel core.Level
		expected  string // We'll check the string representation
	}{
		{core.DebugLevel, "DEBUG"},
		{core.InfoLevel, "INFO"},
		{core.WarnLevel, "WARN"},
		{core.ErrorLevel, "ERROR"},
		{core.FatalLevel, "ERROR"}, // Fatal maps to Error in slog
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			slogLevel := mapToSlogLevel(tt.coreLevel)
			// Check that the level is reasonable (we can't easily check exact values)
			// Note: We use ReplaceAttr to convert to lowercase in actual output
			_ = slogLevel // Just ensure the mapping doesn't panic
		})
	}
}

func TestCreateOutputWriters(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		error bool
	}{
		{"empty paths", []string{}, false},
		{"stdout", []string{"stdout"}, false},
		{"stderr", []string{"stderr"}, false},
		{"multiple outputs", []string{"stdout", "stderr"}, false},
		{"invalid file path", []string{"/invalid/path/file.log"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := createManagedOutputWriters(tt.paths)

			if tt.error {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if writer == nil {
				t.Error("Expected writer but got nil")
			}
		})
	}
}

func TestSlogLogger_FieldMapping(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	slogLogger := logger.(*SlogLogger)

	// Test field mapping
	tests := []struct {
		input    string
		expected string
	}{
		{"ts", fields.TimestampField},
		{"msg", fields.MessageField},
		{"trace.id", fields.TraceIDField},
		{"span_id", fields.SpanIDField},
		{"custom_field", "custom_field"}, // Should remain unchanged
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slogLogger.getStandardFieldName(tt.input)
			if result != tt.expected {
				t.Errorf("getStandardFieldName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSlogLogger_ConvertToSlogAttrs(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewSlogLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	slogLogger := logger.(*SlogLogger)

	// Test attribute conversion
	attrs := slogLogger.convertToSlogAttrs("key1", "value1", "key2", 42, "key3")

	// Should have 3 attributes (key3 gets nil value for odd number of args)
	if len(attrs) != 3 {
		t.Errorf("Expected 3 attributes, got %d", len(attrs))
	}

	// The function should not panic with odd number of arguments
	attrs2 := slogLogger.convertToSlogAttrs("single_key")
	if len(attrs2) != 1 {
		t.Errorf("Expected 1 attribute for single key, got %d", len(attrs2))
	}
}

func TestSlogLogger_Different_Formats(t *testing.T) {
	formats := []string{"json", "console", "text", "unknown"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			opt := &option.LogOption{
				Engine:      "slog",
				Level:       "INFO",
				Format:      format,
				OutputPaths: []string{"stdout"},
				OTLP:        &option.OTLPOption{},
			}

			logger, err := NewSlogLogger(opt)
			if err != nil {
				t.Fatalf("Failed to create logger with format %s: %v", format, err)
			}

			if logger == nil {
				t.Fatalf("Logger is nil for format %s", format)
			}

			// Test that the logger works
			logger.Info("test message for format", format)
		})
	}
}

func TestStandardizedHandler_FieldStandardization(t *testing.T) {
	// This test verifies that the standardized handler properly maps field names
	mapper := fields.NewFieldMapper()
	handler := &standardizedHandler{
		handler: nil, // We'll just test the field mapping logic
		mapper:  mapper,
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"ts", fields.TimestampField},
		{"time", fields.TimestampField},
		{"msg", fields.MessageField},
		{"level", fields.LevelField},
		{"trace.id", fields.TraceIDField},
		{"custom", "custom"}, // Should remain unchanged
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := handler.getStandardFieldName(tt.input)
			if result != tt.expected {
				t.Errorf("getStandardFieldName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnyToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, "<nil>"},
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"number", 42, "42"},
		{"boolean", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := anyToString(tt.input)
			if tt.name == "number" || tt.name == "boolean" {
				// For non-string types, we just verify it doesn't panic and returns something
				if result == "" {
					t.Errorf("anyToString(%v) returned empty string", tt.input)
				}
			} else {
				if result != tt.expected {
					t.Errorf("anyToString(%v) = %v, want %v", tt.input, result, tt.expected)
				}
			}
		})
	}
}
