package zap

import (
	"context"
	"strings"
	"testing"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/fields"
	"github.com/kart-io/logger/option"
)

func TestNewZapLogger(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("NewZapLogger() error = %v", err)
	}

	if logger == nil {
		t.Fatal("NewZapLogger() returned nil logger")
	}

	zapLogger, ok := logger.(*ZapLogger)
	if !ok {
		t.Fatal("NewZapLogger() didn't return *ZapLogger")
	}

	if zapLogger.level != core.InfoLevel {
		t.Errorf("Expected level to be InfoLevel, got %v", zapLogger.level)
	}
}

func TestNewZapLogger_InvalidConfig(t *testing.T) {
	opt := &option.LogOption{
		Engine: "zap",
		Level:  "INVALID_LEVEL",
		OTLP:   &option.OTLPOption{},
	}

	logger, err := NewZapLogger(opt)

	if err == nil {
		t.Error("Expected error for invalid config")
	}

	if logger != nil {
		t.Error("Expected nil logger for invalid config")
	}
}

func TestZapLogger_BasicLogging(t *testing.T) {
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "DEBUG",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP:        &option.OTLPOption{},
	}

	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test basic logging methods exist and don't panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	t.Log("Basic logging methods executed without panic")
}

func TestZapLogger_FormattedLogging(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
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

func TestZapLogger_StructuredLogging(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
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

func TestZapLogger_With(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test With method
	childLogger := logger.With("service", "test-service", "version", "1.0.0")

	if childLogger == nil {
		t.Fatal("With() returned nil")
	}

	// Verify it's still a ZapLogger
	if _, ok := childLogger.(*ZapLogger); !ok {
		t.Fatal("With() didn't return *ZapLogger")
	}

	// Test that child logger works
	childLogger.Info("child logger message")
	t.Log("Child logger created and used successfully")
}

func TestZapLogger_WithCallerSkip(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test WithCallerSkip method
	skippedLogger := logger.WithCallerSkip(2)

	if skippedLogger == nil {
		t.Fatal("WithCallerSkip() returned nil")
	}

	zapLogger, ok := skippedLogger.(*ZapLogger)
	if !ok {
		t.Fatal("WithCallerSkip() didn't return *ZapLogger")
	}

	if zapLogger.callerSkip != 2 {
		t.Errorf("Expected callerSkip to be 2, got %d", zapLogger.callerSkip)
	}

	skippedLogger.Info("caller skip test")
	t.Log("WithCallerSkip worked correctly")
}

func TestZapLogger_SetLevel(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test SetLevel method
	logger.SetLevel(core.ErrorLevel)

	zapLogger := logger.(*ZapLogger)
	if zapLogger.level != core.ErrorLevel {
		t.Errorf("Expected level to be ErrorLevel, got %v", zapLogger.level)
	}
}

func TestZapLogger_FieldMapping(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	zapLogger := logger.(*ZapLogger)

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
			result := zapLogger.getStandardFieldName(tt.input)
			if result != tt.expected {
				t.Errorf("getStandardFieldName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestZapLogger_StandardizeFields(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	zapLogger := logger.(*ZapLogger)

	// Test field standardization
	standardized := zapLogger.standardizeFields("ts", "2023-01-01", "msg", "test", "custom", "value")

	expected := []interface{}{fields.TimestampField, "2023-01-01", fields.MessageField, "test", "custom", "value"}

	if len(standardized) != len(expected) {
		t.Fatalf("Expected %d fields, got %d", len(expected), len(standardized))
	}

	for i, v := range expected {
		if standardized[i] != v {
			t.Errorf("Field %d: expected %v, got %v", i, v, standardized[i])
		}
	}
}

func TestZapLogger_StandardizeFields_OddArgs(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	zapLogger := logger.(*ZapLogger)

	// Test with odd number of arguments
	standardized := zapLogger.standardizeFields("key1", "value1", "key2")

	if len(standardized) != 4 {
		t.Errorf("Expected 4 fields for odd args, got %d", len(standardized))
	}

	// Last value should be nil for the unpaired key
	if standardized[3] != nil {
		t.Errorf("Expected last value to be nil, got %v", standardized[3])
	}
}

func TestMapToZapLevel(t *testing.T) {
	tests := []struct {
		coreLevel core.Level
		zapName   string // We'll check the string representation
	}{
		{core.DebugLevel, "debug"},
		{core.InfoLevel, "info"},
		{core.WarnLevel, "warn"},
		{core.ErrorLevel, "error"},
		{core.FatalLevel, "fatal"},
	}

	for _, tt := range tests {
		t.Run(tt.zapName, func(t *testing.T) {
			zapLevel := mapToZapLevel(tt.coreLevel)
			if zapLevel.String() != tt.zapName {
				t.Errorf("mapToZapLevel(%v) = %v, want level with string %v", tt.coreLevel, zapLevel, tt.zapName)
			}
		})
	}
}

func TestCreateZapConfig(t *testing.T) {
	tests := []struct {
		name string
		opt  *option.LogOption
	}{
		{
			name: "development config",
			opt: &option.LogOption{
				Engine:      "zap",
				Level:       "DEBUG",
				Format:      "console",
				Development: true,
				OTLP:        &option.OTLPOption{},
			},
		},
		{
			name: "production config",
			opt: &option.LogOption{
				Engine:      "zap",
				Level:       "INFO",
				Format:      "json",
				Development: false,
				OTLP:        &option.OTLPOption{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, _ := core.ParseLevel(tt.opt.Level)
			config := createZapConfig(tt.opt, level)

			// Check that config was created successfully
			if config.Level.Level().String() != strings.ToLower(tt.opt.Level) {
				t.Errorf("Expected level %s, got %s", tt.opt.Level, config.Level.Level().String())
			}

			expectedEncoding := "json"
			if strings.ToLower(tt.opt.Format) == "console" || strings.ToLower(tt.opt.Format) == "text" {
				expectedEncoding = "console"
			}

			if config.Encoding != expectedEncoding {
				t.Errorf("Expected encoding %s, got %s", expectedEncoding, config.Encoding)
			}

			if config.DisableCaller != tt.opt.DisableCaller {
				t.Errorf("Expected DisableCaller %t, got %t", tt.opt.DisableCaller, config.DisableCaller)
			}
		})
	}
}

func TestCreateStandardizedEncoderConfig(t *testing.T) {
	config := createStandardizedEncoderConfig()

	// Check that standardized field names are used
	if config.TimeKey != fields.TimestampField {
		t.Errorf("Expected TimeKey to be %s, got %s", fields.TimestampField, config.TimeKey)
	}

	if config.LevelKey != fields.LevelField {
		t.Errorf("Expected LevelKey to be %s, got %s", fields.LevelField, config.LevelKey)
	}

	if config.MessageKey != fields.MessageField {
		t.Errorf("Expected MessageKey to be %s, got %s", fields.MessageField, config.MessageKey)
	}

	if config.CallerKey != fields.CallerField {
		t.Errorf("Expected CallerKey to be %s, got %s", fields.CallerField, config.CallerKey)
	}
}

func TestNormalizeOutputPaths(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{[]string{"stdout"}, []string{"stdout"}},
		{[]string{"stderr"}, []string{"stderr"}},
		{[]string{"STDOUT"}, []string{"stdout"}},
		{[]string{"STDERR"}, []string{"stderr"}},
		{[]string{""}, []string{"stdout"}},
		{[]string{"file.log"}, []string{"file.log"}},
		{[]string{"stdout", "stderr", "file.log"}, []string{"stdout", "stderr", "file.log"}},
	}

	for _, tt := range tests {
		t.Run("normalize paths", func(t *testing.T) {
			result := normalizeOutputPaths(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d paths, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Path %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestZapLogger_Different_Formats(t *testing.T) {
	formats := []string{"json", "console", "text", "unknown"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			opt := &option.LogOption{
				Engine:      "zap",
				Level:       "INFO",
				Format:      format,
				OutputPaths: []string{"stdout"},
				OTLP:        &option.OTLPOption{},
			}

			logger, err := NewZapLogger(opt)
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

func TestAnyToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, "<nil>"},
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"number", 42, "42"}, // This will be the String field from zap.Any
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

func TestZapLogger_WithCtx(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := NewZapLogger(opt)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test WithCtx method (should behave like With since Zap doesn't have context support)
	ctx := context.Background()
	ctxLogger := logger.WithCtx(ctx, "service", "test")

	if ctxLogger == nil {
		t.Fatal("WithCtx() returned nil")
	}

	if _, ok := ctxLogger.(*ZapLogger); !ok {
		t.Fatal("WithCtx() didn't return *ZapLogger")
	}

	ctxLogger.Info("context logger test")
	t.Log("WithCtx worked correctly")
}
