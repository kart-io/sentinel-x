package kratos

import (
	"fmt"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/integrations"
)

// Level represents Kratos log levels without importing Kratos
type Level int32

const (
	LevelDebug Level = iota - 1
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// Logger mimics Kratos' log.Logger interface to avoid direct dependency
type Logger interface {
	Log(level Level, keyvals ...interface{}) error
	With(keyvals ...interface{}) Logger
}

// Helper mimics Kratos' log.Helper
type Helper struct {
	logger Logger
}

// NewHelper creates a new Helper
func NewHelper(logger Logger) *Helper {
	return &Helper{logger: logger}
}

// Debug logs debug level message
func (h *Helper) Debug(a ...interface{}) {
	h.logger.Log(LevelDebug, "msg", fmt.Sprint(a...))
}

// Info logs info level message
func (h *Helper) Info(a ...interface{}) {
	h.logger.Log(LevelInfo, "msg", fmt.Sprint(a...))
}

// Warn logs warn level message
func (h *Helper) Warn(a ...interface{}) {
	h.logger.Log(LevelWarn, "msg", fmt.Sprint(a...))
}

// Error logs error level message
func (h *Helper) Error(a ...interface{}) {
	h.logger.Log(LevelError, "msg", fmt.Sprint(a...))
}

// FilterFunc defines a function to filter log entries
type FilterFunc func(level Level, keyvals ...interface{}) bool

// Filter wraps a logger with filtering capability
type Filter struct {
	logger Logger
	filter FilterFunc
}

// NewFilter creates a new filtered logger
func NewFilter(logger Logger, filter FilterFunc) Logger {
	return &Filter{logger: logger, filter: filter}
}

// Log applies filter and logs if passed
func (f *Filter) Log(level Level, keyvals ...interface{}) error {
	if f.filter != nil && !f.filter(level, keyvals...) {
		return nil
	}
	return f.logger.Log(level, keyvals...)
}

// With returns a new filtered logger with additional keyvals
func (f *Filter) With(keyvals ...interface{}) Logger {
	return &Filter{logger: f.logger.With(keyvals...), filter: f.filter}
}

// StdLogger provides standard library logger interface
type StdLogger struct {
	logger Logger
}

// NewStdLogger creates a new standard logger wrapper
func NewStdLogger(logger Logger) *StdLogger {
	return &StdLogger{logger: logger}
}

// Print implements standard library logger interface
func (s *StdLogger) Print(v ...interface{}) {
	s.logger.Log(LevelInfo, "msg", fmt.Sprint(v...))
}

// Printf implements standard library logger interface
func (s *StdLogger) Printf(format string, v ...interface{}) {
	s.logger.Log(LevelInfo, "msg", fmt.Sprintf(format, v...))
}

// Println implements standard library logger interface
func (s *StdLogger) Println(v ...interface{}) {
	s.logger.Log(LevelInfo, "msg", fmt.Sprintln(v...))
}

// KratosAdapter implements Kratos' log.Logger interface using our unified logger
type KratosAdapter struct {
	*integrations.BaseAdapter
	keyvals []interface{}
}

// NewKratosAdapter creates a new Kratos adapter
func NewKratosAdapter(coreLogger core.Logger) *KratosAdapter {
	baseAdapter := integrations.NewBaseAdapter(coreLogger, "Kratos", "v2.x")
	return &KratosAdapter{
		BaseAdapter: baseAdapter,
		keyvals:     make([]interface{}, 0),
	}
}

// Log implements Kratos' log.Logger interface
func (k *KratosAdapter) Log(level Level, keyvals ...interface{}) error {
	// Combine adapter keyvals with provided keyvals
	allKeyvals := make([]interface{}, 0, len(k.keyvals)+len(keyvals))
	allKeyvals = append(allKeyvals, k.keyvals...)
	allKeyvals = append(allKeyvals, keyvals...)

	// Extract message if present
	msg := k.extractMessage(allKeyvals)
	if msg == "" {
		msg = "Kratos log message"
	}

	// Add kratos-specific fields
	fieldsWithMeta := append([]interface{}{
		"component", "kratos",
		"level", level.String(),
	}, allKeyvals...)

	// Map Kratos levels to our core levels and log
	switch level {
	case LevelDebug:
		k.GetLogger().Debugw(msg, fieldsWithMeta...)
	case LevelInfo:
		k.GetLogger().Infow(msg, fieldsWithMeta...)
	case LevelWarn:
		k.GetLogger().Warnw(msg, fieldsWithMeta...)
	case LevelError:
		k.GetLogger().Errorw(msg, fieldsWithMeta...)
	case LevelFatal:
		k.GetLogger().Fatalw(msg, fieldsWithMeta...)
	default:
		k.GetLogger().Infow(msg, fieldsWithMeta...)
	}

	return nil
}

// With implements Kratos' log.Logger interface
func (k *KratosAdapter) With(keyvals ...interface{}) Logger {
	newAdapter := &KratosAdapter{
		BaseAdapter: k.BaseAdapter,
		keyvals:     make([]interface{}, len(k.keyvals)+len(keyvals)),
	}

	copy(newAdapter.keyvals, k.keyvals)
	copy(newAdapter.keyvals[len(k.keyvals):], keyvals)

	return newAdapter
}

// LogRequest logs an HTTP request (implements HTTPAdapter interface)
func (k *KratosAdapter) LogRequest(method, path string, statusCode int, duration int64, userID string) {
	fields := []interface{}{
		"component", "kratos",
		"operation", "http_request",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration_ms", float64(duration) / 1e6,
	}

	if userID != "" {
		fields = append(fields, "user_id", userID)
	}

	level := k.getLogLevelForStatusCode(statusCode)
	msg := fmt.Sprintf("HTTP %s %s", method, path)

	switch level {
	case core.InfoLevel:
		k.GetLogger().Infow(msg, fields...)
	case core.WarnLevel:
		k.GetLogger().Warnw(msg, fields...)
	case core.ErrorLevel:
		k.GetLogger().Errorw(msg, fields...)
	default:
		k.GetLogger().Infow(msg, fields...)
	}
}

// LogMiddleware logs middleware execution (implements HTTPAdapter interface)
func (k *KratosAdapter) LogMiddleware(middlewareName string, duration int64) {
	fields := []interface{}{
		"component", "kratos",
		"operation", "middleware",
		"middleware_name", middlewareName,
		"duration_ms", float64(duration) / 1e6,
	}

	k.GetLogger().Debugw("Middleware executed", fields...)
}

// LogError logs HTTP-related errors (implements HTTPAdapter interface)
func (k *KratosAdapter) LogError(err error, method, path string, statusCode int) {
	fields := []interface{}{
		"component", "kratos",
		"operation", "http_error",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"error", err.Error(),
	}

	k.GetLogger().Errorw("HTTP request failed", fields...)
}

// Helper methods

// extractMessage extracts a message from keyvals, looking for common message keys
func (k *KratosAdapter) extractMessage(keyvals []interface{}) string {
	messageKeys := []string{"msg", "message", "event", "description"}

	for i := 0; i < len(keyvals); i += 2 {
		if i+1 >= len(keyvals) {
			break
		}

		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}

		for _, msgKey := range messageKeys {
			if key == msgKey {
				if msgValue, ok := keyvals[i+1].(string); ok {
					return msgValue
				}
				return fmt.Sprintf("%v", keyvals[i+1])
			}
		}
	}

	return ""
}

// getLogLevelForStatusCode determines the appropriate log level based on HTTP status code
func (k *KratosAdapter) getLogLevelForStatusCode(statusCode int) core.Level {
	switch {
	case statusCode >= 200 && statusCode < 400:
		return core.InfoLevel
	case statusCode >= 400 && statusCode < 500:
		return core.WarnLevel
	case statusCode >= 500:
		return core.ErrorLevel
	default:
		return core.InfoLevel
	}
}

// Helper functions for creating Kratos loggers

// NewKratosHelper creates a Kratos helper with our adapter
func NewKratosHelper(coreLogger core.Logger) *Helper {
	adapter := NewKratosAdapter(coreLogger)
	return NewHelper(adapter)
}

// NewKratosFilter creates a Kratos filter logger
func NewKratosFilter(coreLogger core.Logger, filter FilterFunc) Logger {
	adapter := NewKratosAdapter(coreLogger)
	return NewFilter(adapter, filter)
}

// NewKratosStdLogger creates a standard library logger compatible with Kratos
func NewKratosStdLogger(coreLogger core.Logger) *StdLogger {
	adapter := NewKratosAdapter(coreLogger)
	return NewStdLogger(adapter)
}

// Verify that KratosAdapter implements both Kratos' logger interface and our HTTPAdapter interface
var (
	_ Logger                   = (*KratosAdapter)(nil)
	_ integrations.HTTPAdapter = (*KratosAdapter)(nil)
)
