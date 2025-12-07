package core

import "context"

// NoOpLogger implements Logger interface but performs no actual logging.
// Used as a fallback when logger creation fails to prevent application crashes.
type NoOpLogger struct {
	lastError error
}

// NewNoOpLogger creates a new no-operation logger.
func NewNoOpLogger(err error) Logger {
	return &NoOpLogger{lastError: err}
}

// GetLastError returns the last error that caused this no-op logger to be created.
func (n *NoOpLogger) GetLastError() error {
	return n.lastError
}

// Basic logging methods - all no-op
func (n *NoOpLogger) Debug(args ...interface{}) {}
func (n *NoOpLogger) Info(args ...interface{})  {}
func (n *NoOpLogger) Warn(args ...interface{})  {}
func (n *NoOpLogger) Error(args ...interface{}) {}
func (n *NoOpLogger) Fatal(args ...interface{}) {}

// Printf-style methods - all no-op
func (n *NoOpLogger) Debugf(template string, args ...interface{}) {}
func (n *NoOpLogger) Infof(template string, args ...interface{})  {}
func (n *NoOpLogger) Warnf(template string, args ...interface{})  {}
func (n *NoOpLogger) Errorf(template string, args ...interface{}) {}
func (n *NoOpLogger) Fatalf(template string, args ...interface{}) {}

// Structured logging methods - all no-op
func (n *NoOpLogger) Debugw(msg string, keysAndValues ...interface{}) {}
func (n *NoOpLogger) Infow(msg string, keysAndValues ...interface{})  {}
func (n *NoOpLogger) Warnw(msg string, keysAndValues ...interface{})  {}
func (n *NoOpLogger) Errorw(msg string, keysAndValues ...interface{}) {}
func (n *NoOpLogger) Fatalw(msg string, keysAndValues ...interface{}) {}

// Enhancement methods - return self to maintain chain
func (n *NoOpLogger) With(keyValues ...interface{}) Logger {
	return n
}

func (n *NoOpLogger) WithCtx(ctx context.Context, keyValues ...interface{}) Logger {
	return n
}

func (n *NoOpLogger) WithCallerSkip(skip int) Logger {
	return n
}

// Configuration methods - no-op
func (n *NoOpLogger) SetLevel(level Level) {}

// Buffer management - always returns nil
func (n *NoOpLogger) Flush() error {
	return nil
}
