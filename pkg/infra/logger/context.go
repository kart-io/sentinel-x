// Package logger provides context-aware structured logging capabilities.
package logger

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/kart-io/logger/core"
)

// ContextLogger wraps a core.Logger and automatically includes context fields.
// It is thread-safe and can be used concurrently.
type ContextLogger struct {
	ctx    context.Context
	logger core.Logger
}

// NewContextLogger creates a new ContextLogger from the given context.
// It extracts all logger fields from the context and creates a logger with those fields.
func NewContextLogger(ctx context.Context) *ContextLogger {
	return &ContextLogger{
		ctx:    ctx,
		logger: GetLogger(ctx),
	}
}

// WithContext creates a new ContextLogger with updated context.
// This is useful when the context changes (e.g., new fields are added).
func (cl *ContextLogger) WithContext(ctx context.Context) *ContextLogger {
	return &ContextLogger{
		ctx:    ctx,
		logger: GetLogger(ctx),
	}
}

// WithFields creates a new ContextLogger with additional fields.
// The new fields are appended to existing context fields.
func (cl *ContextLogger) WithFields(fields ...interface{}) *ContextLogger {
	return &ContextLogger{
		ctx:    cl.ctx,
		logger: cl.logger.With(fields...),
	}
}

// Debug logs a debug message with context fields.
func (cl *ContextLogger) Debug(msg string) {
	cl.logger.Debug(msg)
}

// Debugf logs a formatted debug message with context fields.
func (cl *ContextLogger) Debugf(format string, args ...interface{}) {
	cl.logger.Debugf(format, args...)
}

// Debugw logs a debug message with additional fields and context fields.
func (cl *ContextLogger) Debugw(msg string, keysAndValues ...interface{}) {
	cl.logger.Debugw(msg, keysAndValues...)
}

// Info logs an info message with context fields.
func (cl *ContextLogger) Info(msg string) {
	cl.logger.Info(msg)
}

// Infof logs a formatted info message with context fields.
func (cl *ContextLogger) Infof(format string, args ...interface{}) {
	cl.logger.Infof(format, args...)
}

// Infow logs an info message with additional fields and context fields.
func (cl *ContextLogger) Infow(msg string, keysAndValues ...interface{}) {
	cl.logger.Infow(msg, keysAndValues...)
}

// Warn logs a warning message with context fields.
func (cl *ContextLogger) Warn(msg string) {
	cl.logger.Warn(msg)
}

// Warnf logs a formatted warning message with context fields.
func (cl *ContextLogger) Warnf(format string, args ...interface{}) {
	cl.logger.Warnf(format, args...)
}

// Warnw logs a warning message with additional fields and context fields.
func (cl *ContextLogger) Warnw(msg string, keysAndValues ...interface{}) {
	cl.logger.Warnw(msg, keysAndValues...)
}

// Error logs an error message with context fields.
func (cl *ContextLogger) Error(msg string) {
	cl.logger.Error(msg)
}

// Errorf logs a formatted error message with context fields.
func (cl *ContextLogger) Errorf(format string, args ...interface{}) {
	cl.logger.Errorf(format, args...)
}

// Errorw logs an error message with additional fields and context fields.
func (cl *ContextLogger) Errorw(msg string, keysAndValues ...interface{}) {
	cl.logger.Errorw(msg, keysAndValues...)
}

// ErrorWithError logs an error with structured error fields and optional stack trace.
func (cl *ContextLogger) ErrorWithError(msg string, err error, captureStack bool) {
	fields := []interface{}{
		"error_message", err.Error(),
		"error_type", fmt.Sprintf("%T", err),
	}

	if captureStack {
		stack := captureStackTrace(3) // Skip 3 frames: captureStackTrace, ErrorWithError, caller
		fields = append(fields, "stack_trace", stack)
	}

	cl.logger.Errorw(msg, fields...)
}

// Fatal logs a fatal message with context fields and exits.
func (cl *ContextLogger) Fatal(msg string) {
	cl.logger.Fatal(msg)
}

// Fatalf logs a formatted fatal message with context fields and exits.
func (cl *ContextLogger) Fatalf(format string, args ...interface{}) {
	cl.logger.Fatalf(format, args...)
}

// Fatalw logs a fatal message with additional fields and context fields, then exits.
func (cl *ContextLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	cl.logger.Fatalw(msg, keysAndValues...)
}

// Context returns the underlying context.
func (cl *ContextLogger) Context() context.Context {
	return cl.ctx
}

// Logger returns the underlying core.Logger.
func (cl *ContextLogger) Logger() core.Logger {
	return cl.logger
}

// captureStackTrace captures the current stack trace, skipping the specified number of frames.
func captureStackTrace(skip int) string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip, pcs[:])

	if n == 0 {
		return ""
	}

	frames := runtime.CallersFrames(pcs[:n])
	var builder strings.Builder

	for {
		frame, more := frames.Next()
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))

		if !more {
			break
		}
	}

	return builder.String()
}

// LogError is a convenience function that logs an error with structured fields.
// It automatically categorizes the error and optionally captures a stack trace.
func LogError(ctx context.Context, msg string, err error, captureStack bool) {
	logger := GetLogger(ctx)
	fields := []interface{}{
		"error_message", err.Error(),
		"error_type", fmt.Sprintf("%T", err),
	}

	if captureStack {
		stack := captureStackTrace(2) // Skip 2 frames: captureStackTrace, LogError
		fields = append(fields, "stack_trace", stack)
	}

	logger.Errorw(msg, fields...)
}

// LogInfo is a convenience function that logs an info message with context fields.
func LogInfo(ctx context.Context, msg string, keysAndValues ...interface{}) {
	logger := GetLogger(ctx)
	logger.Infow(msg, keysAndValues...)
}

// LogDebug is a convenience function that logs a debug message with context fields.
func LogDebug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	logger := GetLogger(ctx)
	logger.Debugw(msg, keysAndValues...)
}

// LogWarn is a convenience function that logs a warning message with context fields.
func LogWarn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	logger := GetLogger(ctx)
	logger.Warnw(msg, keysAndValues...)
}

// UnwrapError recursively unwraps an error chain and returns all error messages.
// This is useful for logging the complete error chain in structured logs.
func UnwrapError(err error) []string {
	if err == nil {
		return nil
	}

	var messages []string
	for err != nil {
		messages = append(messages, err.Error())

		// Try to unwrap using Unwrap() method
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = unwrapper.Unwrap()
	}

	return messages
}

// LogErrorChain logs an error with its complete error chain.
// This is useful for debugging wrapped errors.
func LogErrorChain(ctx context.Context, msg string, err error, captureStack bool) {
	logger := GetLogger(ctx)

	errorChain := UnwrapError(err)
	fields := []interface{}{
		"error_message", err.Error(),
		"error_type", fmt.Sprintf("%T", err),
		"error_chain", errorChain,
	}

	if captureStack {
		stack := captureStackTrace(2)
		fields = append(fields, "stack_trace", stack)
	}

	logger.Errorw(msg, fields...)
}

// ContextualLoggerFunc is a helper type for functions that need a context-aware logger.
// This promotes a consistent pattern for passing loggers in function signatures.
type ContextualLoggerFunc func(ctx context.Context) core.Logger

// DefaultContextualLogger returns the default contextual logger function.
// It uses GetLogger to extract context fields.
var DefaultContextualLogger ContextualLoggerFunc = GetLogger

// SetGlobalContextualLogger sets a custom contextual logger function.
// This is useful for testing or custom logging strategies.
func SetGlobalContextualLogger(fn ContextualLoggerFunc) {
	if fn != nil {
		DefaultContextualLogger = fn
	}
}

// Must is a helper that wraps a call to a function returning (core.Logger, error)
// and panics if the error is non-nil. This is useful for logger initialization.
func Must(log core.Logger, err error) core.Logger {
	if err != nil {
		panic(err)
	}
	return log
}

// MustInit initializes the global logger with options and panics on error.
// This is a convenience function for application initialization.
func MustInit(opts *Options) {
	if err := opts.Init(); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
}

// SyncGlobal flushes any buffered log entries in the global logger.
// It's safe to call this multiple times. This should be called before application shutdown.
func SyncGlobal() error {
	// The kart-io/logger package doesn't expose a Sync function directly
	// This is a placeholder for future implementation if the logger package adds it
	return nil
}
