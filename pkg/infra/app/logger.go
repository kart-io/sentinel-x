// Package app provides logger integration for applications.
package app

import (
	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
)

// Logger returns the global logger instance.
func Logger() core.Logger {
	return logger.Global()
}

// Debug logs a debug message using the global logger.
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Info logs an info message using the global logger.
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn logs a warning message using the global logger.
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Error logs an error message using the global logger.
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Fatal logs a fatal message using the global logger.
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Debugf logs a debug message with formatting using the global logger.
func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

// Infof logs an info message with formatting using the global logger.
func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

// Warnf logs a warning message with formatting using the global logger.
func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

// Errorf logs an error message with formatting using the global logger.
func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

// Fatalf logs a fatal message with formatting using the global logger.
func Fatalf(template string, args ...interface{}) {
	logger.Fatalf(template, args...)
}

// Debugw logs a debug message with structured fields using the global logger.
func Debugw(msg string, keysAndValues ...interface{}) {
	logger.Debugw(msg, keysAndValues...)
}

// Infow logs an info message with structured fields using the global logger.
func Infow(msg string, keysAndValues ...interface{}) {
	logger.Infow(msg, keysAndValues...)
}

// Warnw logs a warning message with structured fields using the global logger.
func Warnw(msg string, keysAndValues ...interface{}) {
	logger.Warnw(msg, keysAndValues...)
}

// Errorw logs an error message with structured fields using the global logger.
func Errorw(msg string, keysAndValues ...interface{}) {
	logger.Errorw(msg, keysAndValues...)
}

// Fatalw logs a fatal message with structured fields using the global logger.
func Fatalw(msg string, keysAndValues ...interface{}) {
	logger.Fatalw(msg, keysAndValues...)
}

// With creates a child logger with the specified key-value pairs using the global logger.
func With(keysAndValues ...interface{}) core.Logger {
	return logger.With(keysAndValues...)
}

// Flush flushes any buffered log entries using the global logger.
func Flush() error {
	return logger.Flush()
}
