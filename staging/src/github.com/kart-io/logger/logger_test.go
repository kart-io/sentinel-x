package logger

import (
	"testing"

	"github.com/kart-io/logger/option"
)

func TestNew(t *testing.T) {
	opt := option.DefaultLogOption()
	logger, err := New(opt)

	// Now that slog is implemented, this should work
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if logger == nil {
		t.Error("Expected logger to be created successfully")
	}
}

func TestNewWithDefaults(t *testing.T) {
	logger, err := NewWithDefaults()

	// Now that slog is implemented, this should work
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if logger == nil {
		t.Error("Expected logger to be created successfully")
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test that global logger management works

	// Initially global should be nil
	if global != nil {
		t.Error("Expected global logger to be nil initially")
	}

	// Now that slog works, Global() should successfully create a logger
	globalLogger := Global()

	if globalLogger == nil {
		t.Error("Expected Global() to return a logger")
	}

	// The global variable should now be set
	if global == nil {
		t.Error("Expected global variable to be set after calling Global()")
	}
}

func TestSetGlobal(t *testing.T) {
	// Test that we can set a global logger (using nil for now since we can't create real ones)
	SetGlobal(nil)

	if global != nil {
		t.Error("Expected global to be nil after SetGlobal(nil)")
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Test that package-level convenience functions exist and can be called
	// Reset global to nil first
	SetGlobal(nil)

	// These should work now that slog is implemented
	Debug("test debug")
	Info("test info")
	Warn("test warn")
	Error("test error")

	// Formatted functions
	Debugf("test %s", "debug")
	Infof("test %s", "info")
	Warnf("test %s", "warn")
	Errorf("test %s", "error")

	// Structured functions
	Debugw("test debug", "key", "value")
	Infow("test info", "key", "value")
	Warnw("test warn", "key", "value")
	Errorw("test error", "key", "value")

	// With function
	childLogger := With("service", "test")
	if childLogger == nil {
		t.Error("Expected With() to return a logger")
	}

	t.Log("All package-level functions executed successfully")
}

func TestPackageLevelFunctionsList(t *testing.T) {
	// Verify that all expected package-level functions exist by checking they don't cause compile errors
	// We can't actually call them without panicking, but we can verify they exist

	// Basic logging functions
	_ = Debug
	_ = Info
	_ = Warn
	_ = Error
	_ = Fatal

	// Formatted logging functions
	_ = Debugf
	_ = Infof
	_ = Warnf
	_ = Errorf
	_ = Fatalf

	// Structured logging functions
	_ = Debugw
	_ = Infow
	_ = Warnw
	_ = Errorw
	_ = Fatalw

	// Logger enhancement function
	_ = With

	// If we reach here, all functions exist
	t.Log("All expected package-level functions are defined")
}

func TestNewWithCustomOption(t *testing.T) {
	customOpt := &option.LogOption{
		Engine:      "zap", // This will fallback to slog
		Level:       "DEBUG",
		Format:      "console",
		OutputPaths: []string{"stdout"},
		Development: true,
		OTLP:        &option.OTLPOption{Protocol: "http"},
	}

	logger, err := New(customOpt)

	// Should work now with fallback to slog
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if logger == nil {
		t.Error("Expected logger to be created successfully")
	}
}
