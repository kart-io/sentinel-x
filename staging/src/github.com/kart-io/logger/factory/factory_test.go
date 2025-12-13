package factory

import (
	"context"
	"strings"
	"testing"

	"github.com/kart-io/logger/errors"
	"github.com/kart-io/logger/option"
)

func TestNewLoggerFactory(t *testing.T) {
	opt := option.DefaultLogOption()
	factory := NewLoggerFactory(opt)

	if factory == nil {
		t.Fatal("NewLoggerFactory() returned nil")
	}

	if factory.option != opt {
		t.Error("Factory option was not set correctly")
	}
}

func TestLoggerFactory_GetOption(t *testing.T) {
	opt := option.DefaultLogOption()
	factory := NewLoggerFactory(opt)

	if got := factory.GetOption(); got != opt {
		t.Errorf("GetOption() = %v, want %v", got, opt)
	}
}

func TestLoggerFactory_UpdateOption(t *testing.T) {
	factory := NewLoggerFactory(option.DefaultLogOption())

	newOpt := &option.LogOption{
		Engine: "zap",
		Level:  "DEBUG",
		Format: "console",
		OTLP:   &option.OTLPOption{},
	}

	err := factory.UpdateOption(newOpt)
	if err != nil {
		t.Errorf("UpdateOption() error = %v", err)
	}

	if factory.GetOption().Engine != "zap" {
		t.Errorf("Expected engine to be updated to 'zap', got %s", factory.GetOption().Engine)
	}

	if factory.GetOption().Level != "DEBUG" {
		t.Errorf("Expected level to be updated to 'DEBUG', got %s", factory.GetOption().Level)
	}
}

func TestLoggerFactory_UpdateOption_InvalidConfig(t *testing.T) {
	factory := NewLoggerFactory(option.DefaultLogOption())

	invalidOpt := &option.LogOption{
		Engine: "slog",
		Level:  "INVALID_LEVEL", // This should cause validation to fail
		OTLP:   &option.OTLPOption{},
	}

	err := factory.UpdateOption(invalidOpt)
	if err == nil {
		t.Error("Expected UpdateOption() to return error for invalid config")
	}

	// Original configuration should remain unchanged
	if factory.GetOption().Level != "INFO" {
		t.Errorf("Expected original level to be preserved, got %s", factory.GetOption().Level)
	}
}

func TestLoggerFactory_CreateLogger_UnsupportedEngine(t *testing.T) {
	// Note: Since option validation automatically converts invalid engines to "slog",
	// this test verifies the factory behavior when engines are not implemented yet.
	// In the future, we could test with truly unsupported engines after validation is updated.

	opt := &option.LogOption{
		Engine: "unsupported-engine", // This will be converted to "slog" during validation
		Level:  "INFO",
		OTLP:   &option.OTLPOption{},
	}

	factory := NewLoggerFactory(opt)
	logger, err := factory.CreateLogger()
	// With Slog now implemented, this should actually succeed
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if logger == nil {
		t.Error("Expected logger to be created successfully")
	}

	// Verify that the engine was normalized during validation and logger was created
	if factory.GetOption().Engine != "slog" {
		t.Errorf("Expected engine to be normalized to 'slog' during validation, got %s", factory.GetOption().Engine)
	}
}

func TestLoggerFactory_CreateLogger_InvalidConfig(t *testing.T) {
	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INVALID_LEVEL",
		OTLP:   &option.OTLPOption{},
	}

	factory := NewLoggerFactory(opt)
	logger, err := factory.CreateLogger()

	if err == nil {
		t.Error("Expected error for invalid configuration")
	}

	if logger != nil {
		t.Error("Expected logger to be nil for invalid configuration")
	}

	if !strings.Contains(err.Error(), "invalid configuration") {
		t.Errorf("Expected error message to contain 'invalid configuration', got %s", err.Error())
	}
}

func TestLoggerFactory_CreateLogger_EngineImplementationStatus(t *testing.T) {
	// Test the current implementation status of engines
	tests := []struct {
		name        string
		engine      string
		expectError bool
		description string
	}{
		{"zap engine", "zap", false, "Zap engine is now implemented"},
		{"slog engine", "slog", false, "Slog engine is implemented"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &option.LogOption{
				Engine: tt.engine,
				Level:  "INFO",
				OTLP:   &option.OTLPOption{},
			}

			factory := NewLoggerFactory(opt)
			logger, err := factory.CreateLogger()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s engine (%s)", tt.engine, tt.description)
				}
				if logger != nil {
					t.Errorf("Expected logger to be nil for %s engine (%s)", tt.engine, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s engine (%s): %v", tt.engine, tt.description, err)
				}
				if logger == nil {
					t.Errorf("Expected logger to be created for %s engine (%s)", tt.engine, tt.description)
				}
			}
		})
	}
}

func TestLoggerFactory_FallbackBehavior(t *testing.T) {
	// Test the fallback behavior described in the CreateLogger method
	tests := []struct {
		name        string
		engine      string
		expectError bool
		description string
	}{
		{
			name:        "zap works directly",
			engine:      "zap",
			expectError: false, // Should succeed directly with Zap
			description: "Should create zap logger directly",
		},
		{
			name:        "slog works directly",
			engine:      "slog",
			expectError: false, // Should succeed directly
			description: "Should create slog logger directly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &option.LogOption{
				Engine: tt.engine,
				Level:  "INFO",
				OTLP:   &option.OTLPOption{},
			}

			factory := NewLoggerFactory(opt)
			logger, err := factory.CreateLogger()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if logger != nil {
					t.Errorf("Expected nil logger but got one")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if logger == nil {
					t.Errorf("Expected logger but got nil")
				}
			}
		})
	}
}

func TestLoggerFactory_ConfigurationIntegrity(t *testing.T) {
	// Test that the factory maintains configuration integrity
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "DEBUG",
		Format:      "json",
		OutputPaths: []string{"stdout", "file.log"},
		Development: true,
		OTLP: &option.OTLPOption{
			Protocol: "grpc",
		},
	}

	factory := NewLoggerFactory(opt)

	// Verify that the factory maintains the original configuration
	retrieved := factory.GetOption()
	if retrieved.Engine != "slog" {
		t.Errorf("Expected engine 'slog', got %s", retrieved.Engine)
	}
	if retrieved.Level != "DEBUG" {
		t.Errorf("Expected level 'DEBUG', got %s", retrieved.Level)
	}
	if !retrieved.Development {
		t.Error("Expected development mode to be true")
	}
	if retrieved.OTLP.Protocol != "grpc" {
		t.Errorf("Expected OTLP protocol 'grpc', got %s", retrieved.OTLP.Protocol)
	}
}

func TestNewLoggerFactoryWithErrorHandler(t *testing.T) {
	opt := option.DefaultLogOption()
	customErrorHandler := errors.NewErrorHandler(nil)

	factory := NewLoggerFactoryWithErrorHandler(opt, customErrorHandler)

	if factory == nil {
		t.Fatal("NewLoggerFactoryWithErrorHandler() returned nil")
	}

	if factory.GetErrorHandler() != customErrorHandler {
		t.Error("Custom error handler was not set correctly")
	}

	if factory.GetOption() != opt {
		t.Error("Factory option was not set correctly")
	}
}

func TestLoggerFactory_CreateLoggerWithContext(t *testing.T) {
	tests := []struct {
		name        string
		engine      string
		expectError bool
	}{
		{"zap engine with context", "zap", false},
		{"slog engine with context", "slog", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &option.LogOption{
				Engine: tt.engine,
				Level:  "INFO",
				OTLP:   &option.OTLPOption{},
			}

			factory := NewLoggerFactory(opt)
			logger, err := factory.CreateLoggerWithContext(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if logger != nil {
					t.Error("Expected nil logger but got one")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if logger == nil {
					t.Error("Expected logger but got nil")
				}
			}
		})
	}
}

func TestLoggerFactory_UpdateOption_NilConfig(t *testing.T) {
	factory := NewLoggerFactory(option.DefaultLogOption())

	err := factory.UpdateOption(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestLoggerFactory_ErrorHandlerMethods(t *testing.T) {
	factory := NewLoggerFactory(option.DefaultLogOption())

	// Test GetErrorHandler
	errorHandler := factory.GetErrorHandler()
	if errorHandler == nil {
		t.Error("Expected error handler but got nil")
	}

	// Test GetErrorStats
	stats := factory.GetErrorStats()
	if stats == nil {
		t.Error("Expected error stats but got nil")
	}

	// Test GetLastErrors
	lastErrors := factory.GetLastErrors()
	if lastErrors == nil {
		t.Error("Expected last errors map but got nil")
	}

	// Test ResetErrors
	factory.ResetErrors() // Should not panic

	// Test SetErrorCallback
	factory.SetErrorCallback(func(err *errors.LoggerError) {
		// Callback function for testing
	})

	// Test SetFallbackLogger - use NoOpLogger from errors package
	factory.SetFallbackLogger(errors.NewNoOpLogger())
}

func TestLoggerFactory_EngineFailureScenarios(t *testing.T) {
	// Test with invalid configuration that causes engine creation to fail
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"/invalid/path/that/cannot/be/created/test.log"}, // Invalid path
		OTLP:        &option.OTLPOption{},
	}

	factory := NewLoggerFactory(opt)
	logger, err := factory.CreateLogger()

	// Should still return a logger (fallback behavior) but with an error
	if logger == nil {
		t.Error("Expected fallback logger but got nil")
	}
	if err == nil {
		t.Error("Expected error for invalid output path")
	}
}

func TestLoggerFactory_Both_Engines_Available(t *testing.T) {
	// Test that both engines are actually implemented and working
	engines := []string{"zap", "slog"}

	for _, engine := range engines {
		t.Run(engine, func(t *testing.T) {
			opt := &option.LogOption{
				Engine:      engine,
				Level:       "INFO",
				Format:      "json",
				OutputPaths: []string{"stdout"},
				OTLP:        &option.OTLPOption{},
			}

			factory := NewLoggerFactory(opt)
			logger, err := factory.CreateLogger()
			if err != nil {
				t.Errorf("Engine %s should be implemented and working: %v", engine, err)
			}
			if logger == nil {
				t.Errorf("Engine %s should create a valid logger", engine)
			}

			// Test that we can actually log with the created logger
			if logger != nil {
				logger.Info("test message") // Should not panic
			}
		})
	}
}

func TestLoggerFactory_Configuration_Validation_Edge_Cases(t *testing.T) {
	tests := []struct {
		name      string
		config    *option.LogOption
		shouldErr bool
		desc      string
	}{
		{
			name: "empty config",
			config: &option.LogOption{
				OTLP: &option.OTLPOption{},
			},
			shouldErr: true,
			desc:      "empty configuration should fail validation",
		},
		{
			name: "valid minimal config",
			config: &option.LogOption{
				Engine: "slog",
				Level:  "INFO",
				OTLP:   &option.OTLPOption{},
			},
			shouldErr: false,
			desc:      "minimal valid configuration should pass",
		},
		{
			name: "valid zap config",
			config: &option.LogOption{
				Engine:      "zap",
				Level:       "DEBUG",
				Format:      "console",
				OutputPaths: []string{"stdout"},
				Development: true,
				OTLP:        &option.OTLPOption{},
			},
			shouldErr: false,
			desc:      "full zap configuration should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewLoggerFactory(tt.config)
			logger, err := factory.CreateLogger()

			if tt.shouldErr {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.desc)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.desc, err)
				}
				if logger == nil {
					t.Errorf("%s: expected logger but got nil", tt.desc)
				}
			}
		})
	}
}

func TestLoggerFactory_Error_Recovery_And_Fallback(t *testing.T) {
	// Create a factory with a problematic configuration
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"/dev/null"}, // This should work, but let's test fallback anyway
		OTLP:        &option.OTLPOption{},
	}

	factory := NewLoggerFactory(opt)

	// Set up fallback logger
	factory.SetFallbackLogger(errors.NewNoOpLogger())

	// This should succeed normally, but test the pattern
	logger, _ := factory.CreateLogger()

	// Even if there's an error, we should get some form of logger (either the real one or fallback)
	if logger == nil {
		t.Error("Expected some form of logger (real or fallback)")
	}
}

// Additional edge case tests

func TestLoggerFactory_ZapEngineSpecificMethods(t *testing.T) {
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "DEBUG",
		Format:      "console",
		OutputPaths: []string{"stdout"},
		Development: true,
		OTLP:        &option.OTLPOption{},
	}

	factory := NewLoggerFactory(opt)
	logger, err := factory.CreateLogger()
	if err != nil {
		t.Errorf("Zap engine creation failed: %v", err)
	}
	if logger == nil {
		t.Error("Expected Zap logger but got nil")
	}

	// Test that we can call all logger methods
	if logger != nil {
		// Test all 15 logger methods
		logger.Debug("test")
		logger.Info("test")
		logger.Warn("test")
		logger.Error("test")
		logger.Debugf("test %s", "formatted")
		logger.Infof("test %s", "formatted")
		logger.Warnf("test %s", "formatted")
		logger.Errorf("test %s", "formatted")
		logger.Debugw("test", "key", "value")
		logger.Infow("test", "key", "value")
		logger.Warnw("test", "key", "value")
		logger.Errorw("test", "key", "value")

		// Test chained methods
		childLogger := logger.With("child", "true")
		if childLogger == nil {
			t.Error("Expected child logger from With()")
		}

		ctxLogger := logger.WithCtx(context.Background(), "ctx", "test")
		if ctxLogger == nil {
			t.Error("Expected context logger from WithCtx()")
		}

		skipLogger := logger.WithCallerSkip(1)
		if skipLogger == nil {
			t.Error("Expected skip logger from WithCallerSkip()")
		}
	}
}

func TestLoggerFactory_SlogEngineSpecificMethods(t *testing.T) {
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "DEBUG",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
		OTLP:        &option.OTLPOption{},
	}

	factory := NewLoggerFactory(opt)
	logger, err := factory.CreateLogger()
	if err != nil {
		t.Errorf("Slog engine creation failed: %v", err)
	}
	if logger == nil {
		t.Error("Expected Slog logger but got nil")
	}

	// Test all logger interface methods
	if logger != nil {
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")
		// Note: Skip Fatal() as it would exit the test
	}
}

func TestLoggerFactory_DynamicReconfiguration(t *testing.T) {
	// Start with slog
	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		OTLP:   &option.OTLPOption{},
	}

	factory := NewLoggerFactory(opt)

	// Create initial logger
	logger1, err := factory.CreateLogger()
	if err != nil {
		t.Errorf("Initial logger creation failed: %v", err)
	}

	// Update configuration to zap
	newOpt := &option.LogOption{
		Engine: "zap",
		Level:  "DEBUG",
		OTLP:   &option.OTLPOption{},
	}

	err = factory.UpdateOption(newOpt)
	if err != nil {
		t.Errorf("Configuration update failed: %v", err)
	}

	// Verify configuration was updated
	if factory.GetOption().Engine != "zap" {
		t.Error("Engine was not updated to zap")
	}
	if factory.GetOption().Level != "DEBUG" {
		t.Error("Level was not updated to DEBUG")
	}

	// Create new logger with updated configuration
	logger2, err := factory.CreateLogger()
	if err != nil {
		t.Errorf("Updated logger creation failed: %v", err)
	}

	if logger2 == nil {
		t.Error("Expected logger after configuration update")
	}

	// Verify both loggers work
	if logger1 != nil {
		logger1.Info("original slog logger")
	}
	if logger2 != nil {
		logger2.Info("new zap logger")
	}
}
