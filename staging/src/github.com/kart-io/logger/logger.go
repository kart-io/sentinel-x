package logger

import (
	"sync"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/factory"
	"github.com/kart-io/logger/option"
)

// Global logger instance with thread-safe initialization
var (
	global          core.Logger
	globalOptimized core.Logger // Cached optimized global logger for better performance
	globalMu        sync.RWMutex
)

// New creates a new logger with the provided configuration.
func New(opt *option.LogOption) (core.Logger, error) {
	f := factory.NewLoggerFactory(opt)
	return f.CreateLogger()
}

// NewWithDefaults creates a new logger with default configuration.
func NewWithDefaults() (core.Logger, error) {
	return New(option.DefaultLogOption())
}

// SetGlobal sets the global logger instance.
func SetGlobal(logger core.Logger) {
	globalMu.Lock()
	global = logger
	globalOptimized = nil // Clear cached optimized logger
	globalMu.Unlock()
}

// Global returns the global logger instance.
// If no global logger is set, it returns a logger with default configuration.
// This function is thread-safe and ensures only one default logger is created.
func Global() core.Logger {
	globalMu.RLock()
	if global != nil {
		defer globalMu.RUnlock()
		return global
	}
	globalMu.RUnlock()

	// Need to create default logger
	globalMu.Lock()
	defer globalMu.Unlock()

	// Double-check after acquiring write lock
	if global != nil {
		return global
	}

	logger, err := NewWithDefaults()
	if err != nil {
		// Use no-op logger as fallback to prevent application crash
		// This ensures the application can continue running even if default logger creation fails
		global = core.NewNoOpLogger(err)
		return global
	}
	global = logger
	return global
}

// getOptimizedGlobal returns an optimized global logger for package-level functions
func getOptimizedGlobal() core.Logger {
	globalMu.RLock()
	if globalOptimized != nil {
		defer globalMu.RUnlock()
		return globalOptimized
	}
	globalMu.RUnlock()

	globalMu.Lock()
	defer globalMu.Unlock()

	// Double-check after acquiring write lock
	if globalOptimized != nil {
		return globalOptimized
	}

	// Ensure we have a global logger first
	if global == nil {
		logger, err := NewWithDefaults()
		if err != nil {
			global = core.NewNoOpLogger(err)
		} else {
			global = logger
		}
	}

	// Try to create optimized logger
	if optimizer, ok := global.(core.GlobalCallOptimizer); ok {
		globalOptimized = optimizer.CreateGlobalCallLogger()
	} else {
		// Fallback to regular global logger if optimization not supported
		globalOptimized = global
	}

	return globalOptimized
}

// Package-level convenience functions using the global logger

// Debug logs a debug message using the global logger.
func Debug(args ...interface{}) {
	getOptimizedGlobal().Debug(args...)
}

// Info logs an info message using the global logger.
func Info(args ...interface{}) {
	getOptimizedGlobal().Info(args...)
}

// Warn logs a warning message using the global logger.
func Warn(args ...interface{}) {
	getOptimizedGlobal().Warn(args...)
}

// Error logs an error message using the global logger.
func Error(args ...interface{}) {
	getOptimizedGlobal().Error(args...)
}

// Fatal logs a fatal message using the global logger.
func Fatal(args ...interface{}) {
	getOptimizedGlobal().Fatal(args...)
}

// Debugf logs a debug message with formatting using the global logger.
func Debugf(template string, args ...interface{}) {
	getOptimizedGlobal().Debugf(template, args...)
}

// Infof logs an info message with formatting using the global logger.
func Infof(template string, args ...interface{}) {
	getOptimizedGlobal().Infof(template, args...)
}

// Warnf logs a warning message with formatting using the global logger.
func Warnf(template string, args ...interface{}) {
	getOptimizedGlobal().Warnf(template, args...)
}

// Errorf logs an error message with formatting using the global logger.
func Errorf(template string, args ...interface{}) {
	getOptimizedGlobal().Errorf(template, args...)
}

// Fatalf logs a fatal message with formatting using the global logger.
func Fatalf(template string, args ...interface{}) {
	getOptimizedGlobal().Fatalf(template, args...)
}

// Debugw logs a debug message with structured fields using the global logger.
func Debugw(msg string, keysAndValues ...interface{}) {
	getOptimizedGlobal().Debugw(msg, keysAndValues...)
}

// Infow logs an info message with structured fields using the global logger.
func Infow(msg string, keysAndValues ...interface{}) {
	getOptimizedGlobal().Infow(msg, keysAndValues...)
}

// Warnw logs a warning message with structured fields using the global logger.
func Warnw(msg string, keysAndValues ...interface{}) {
	getOptimizedGlobal().Warnw(msg, keysAndValues...)
}

// Errorw logs an error message with structured fields using the global logger.
func Errorw(msg string, keysAndValues ...interface{}) {
	getOptimizedGlobal().Errorw(msg, keysAndValues...)
}

// Fatalw logs a fatal message with structured fields using the global logger.
func Fatalw(msg string, keysAndValues ...interface{}) {
	getOptimizedGlobal().Fatalw(msg, keysAndValues...)
}

// With creates a child logger with the specified key-value pairs using the global logger.
func With(keysAndValues ...interface{}) core.Logger {
	return Global().With(keysAndValues...)
}

// Flush flushes any buffered log entries using the global logger.
func Flush() error {
	return Global().Flush()
}
