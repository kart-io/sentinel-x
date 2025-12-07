package errors

import (
	"context"

	"github.com/kart-io/logger/core"
)

// NoOpLogger is a logger that does nothing - used as a fallback when all else fails
type NoOpLogger struct{}

// NewNoOpLogger creates a new NoOp logger
func NewNoOpLogger() core.Logger {
	return &NoOpLogger{}
}

// Debug does nothing
func (n *NoOpLogger) Debug(args ...interface{}) {}

// Info does nothing
func (n *NoOpLogger) Info(args ...interface{}) {}

// Warn does nothing
func (n *NoOpLogger) Warn(args ...interface{}) {}

// Error does nothing
func (n *NoOpLogger) Error(args ...interface{}) {}

// Fatal does nothing (note: this breaks the typical Fatal behavior of exiting)
func (n *NoOpLogger) Fatal(args ...interface{}) {}

// Debugf does nothing
func (n *NoOpLogger) Debugf(template string, args ...interface{}) {}

// Infof does nothing
func (n *NoOpLogger) Infof(template string, args ...interface{}) {}

// Warnf does nothing
func (n *NoOpLogger) Warnf(template string, args ...interface{}) {}

// Errorf does nothing
func (n *NoOpLogger) Errorf(template string, args ...interface{}) {}

// Fatalf does nothing (note: this breaks the typical Fatal behavior of exiting)
func (n *NoOpLogger) Fatalf(template string, args ...interface{}) {}

// Debugw does nothing
func (n *NoOpLogger) Debugw(msg string, keysAndValues ...interface{}) {}

// Infow does nothing
func (n *NoOpLogger) Infow(msg string, keysAndValues ...interface{}) {}

// Warnw does nothing
func (n *NoOpLogger) Warnw(msg string, keysAndValues ...interface{}) {}

// Errorw does nothing
func (n *NoOpLogger) Errorw(msg string, keysAndValues ...interface{}) {}

// Fatalw does nothing (note: this breaks the typical Fatal behavior of exiting)
func (n *NoOpLogger) Fatalw(msg string, keysAndValues ...interface{}) {}

// With returns the same NoOp logger
func (n *NoOpLogger) With(keysAndValues ...interface{}) core.Logger {
	return n
}

// WithCtx returns the same NoOp logger
func (n *NoOpLogger) WithCtx(ctx context.Context, keysAndValues ...interface{}) core.Logger {
	return n
}

// WithCallerSkip returns the same NoOp logger
func (n *NoOpLogger) WithCallerSkip(skip int) core.Logger {
	return n
}

// SetLevel does nothing
func (n *NoOpLogger) SetLevel(level core.Level) {}

// Flush does nothing
func (n *NoOpLogger) Flush() error { return nil }
