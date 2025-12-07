package errors

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// SafeErrorLogger provides thread-safe error logging to prevent concurrent write issues.
type SafeErrorLogger struct {
	mu sync.Mutex
	w  io.Writer
}

// defaultErrorLogger is a shared instance for OTLP and other internal error logging.
var defaultErrorLogger = &SafeErrorLogger{
	w: os.Stderr,
}

// GetDefaultErrorLogger returns the default thread-safe error logger.
func GetDefaultErrorLogger() *SafeErrorLogger {
	return defaultErrorLogger
}

// LogOTLPError logs OTLP-related errors in a thread-safe manner.
func (s *SafeErrorLogger) LogOTLPError(operation string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintf(s.w, "[OTLP-ERROR] %s: %v\n", operation, err)
}

// LogEngineError logs engine-related errors in a thread-safe manner.
func (s *SafeErrorLogger) LogEngineError(engine string, operation string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintf(s.w, "[%s-ERROR] %s: %v\n", engine, operation, err)
}

// LogGeneralError logs general errors in a thread-safe manner.
func (s *SafeErrorLogger) LogGeneralError(component string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintf(s.w, "[%s-ERROR] %v\n", component, err)
}

// SetWriter changes the output writer (useful for testing).
func (s *SafeErrorLogger) SetWriter(w io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.w = w
}
