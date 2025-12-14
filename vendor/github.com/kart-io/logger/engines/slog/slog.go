package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	goruntime "runtime"
	"strings"
	"sync/atomic"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/errors"
	"github.com/kart-io/logger/fields"
	"github.com/kart-io/logger/option"
	"github.com/kart-io/logger/otlp"
	"github.com/kart-io/logger/runtime"
)

// managedWriter wraps io.Writer with cleanup functionality
type managedWriter struct {
	writers []io.Writer
	closers []io.Closer
}

func (mw *managedWriter) Write(p []byte) (n int, err error) {
	if len(mw.writers) == 1 {
		return mw.writers[0].Write(p)
	}

	// Write to all writers (similar to io.MultiWriter)
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

func (mw *managedWriter) Close() error {
	var firstErr error
	for _, closer := range mw.closers {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// dynamicLevelHandler wraps slog.Handler to support dynamic level changes
type dynamicLevelHandler struct {
	handler     slog.Handler
	levelGetter func() slog.Level
}

func (h *dynamicLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.levelGetter()
}

func (h *dynamicLevelHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.handler.Handle(ctx, record)
}

func (h *dynamicLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &dynamicLevelHandler{
		handler:     h.handler.WithAttrs(attrs),
		levelGetter: h.levelGetter,
	}
}

func (h *dynamicLevelHandler) WithGroup(name string) slog.Handler {
	return &dynamicLevelHandler{
		handler:     h.handler.WithGroup(name),
		levelGetter: h.levelGetter,
	}
}

// SlogLogger implements the core.Logger interface using Go's standard slog library.
type SlogLogger struct {
	logger            *slog.Logger
	level             core.Level
	atomicLevel       *int64 // For atomic level changes (stores slog.Level as int64)
	mapper            *fields.FieldMapper
	callerSkip        int
	isGlobalCall      bool // Cache whether this is a global call to avoid runtime detection
	disableStacktrace bool
	otlpProvider      *otlp.LoggerProvider
	initialFields     map[string]interface{} // Fields from InitialFields config
	persistentFields  map[string]interface{} // Fields added via With()
	managedWriter     *managedWriter         // For resource cleanup
}

// NewSlogLogger creates a new Slog-based logger with the provided configuration.
func NewSlogLogger(opt *option.LogOption) (core.Logger, error) {
	if err := opt.Validate(); err != nil {
		return nil, err
	}

	// Parse the log level
	level, err := core.ParseLevel(opt.Level)
	if err != nil {
		return nil, err
	}

	// Initialize OTLP provider if enabled
	var otlpProvider *otlp.LoggerProvider
	if opt.IsOTLPEnabled() {
		provider, err := otlp.NewLoggerProvider(context.Background(), opt.OTLP)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP provider: %w", err)
		}
		otlpProvider = provider
	}

	// Create managed output writers with proper resource management
	managedWriter, err := createManagedOutputWriters(opt.OutputPaths)
	if err != nil {
		return nil, err
	}

	// Create atomic level for dynamic changes
	atomicLevel := new(int64)
	atomic.StoreInt64(atomicLevel, int64(mapToSlogLevel(level)))

	// Create level getter function for dynamic handler
	levelGetter := func() slog.Level {
		return slog.Level(atomic.LoadInt64(atomicLevel))
	}

	// Create handler options with standardized field names
	handlerOpts := &slog.HandlerOptions{
		Level:     mapToSlogLevel(level),
		AddSource: false, // We'll add standardized caller field ourselves
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			// Standardize field names to match Zap engine output
			switch attr.Key {
			case slog.TimeKey:
				return slog.Attr{Key: fields.TimestampField, Value: attr.Value}
			case slog.LevelKey:
				// Convert level to lowercase for consistent formatting
				if level, ok := attr.Value.Any().(slog.Level); ok {
					return slog.Attr{
						Key:   fields.LevelField,
						Value: slog.StringValue(strings.ToLower(level.String())),
					}
				}
				return slog.Attr{Key: fields.LevelField, Value: attr.Value}
			case slog.MessageKey:
				return slog.Attr{Key: fields.MessageField, Value: attr.Value}
			default:
				return attr
			}
		},
	}

	// Create handler based on format
	var handler slog.Handler
	switch strings.ToLower(opt.Format) {
	case "json":
		handler = slog.NewJSONHandler(managedWriter, handlerOpts)
	case "console", "text":
		handler = slog.NewTextHandler(managedWriter, handlerOpts)
	default:
		handler = slog.NewJSONHandler(managedWriter, handlerOpts)
	}

	// Wrap with dynamic level handler
	dynamicHandler := &dynamicLevelHandler{
		handler:     handler,
		levelGetter: levelGetter,
	}

	// Create standardized handler wrapper for field consistency
	standardHandler := &standardizedHandler{
		handler:           dynamicHandler,
		mapper:            fields.NewFieldMapper(),
		disableCaller:     opt.DisableCaller,
		disableStacktrace: opt.DisableStacktrace,
		otlpProvider:      otlpProvider,
	}

	logger := slog.New(standardHandler)

	// Initialize service fields with defaults and user-provided initial fields
	serviceFields := map[string]interface{}{
		"service.name":    "unknown",
		"service.version": "unknown",
	}

	// Merge user-provided InitialFields, overriding defaults
	if opt.InitialFields != nil {
		for key, value := range opt.InitialFields {
			serviceFields[key] = value
		}
	}

	// Add all fields (defaults + user-provided)
	var initialArgs []interface{}
	for key, value := range serviceFields {
		initialArgs = append(initialArgs, slog.Any(key, value))
	}

	logger = logger.With(initialArgs...)

	// Add basic OTEL fields if OTLP is enabled
	if otlpProvider != nil {
		// Get pod/container/hostname based on deployment environment
		podName := runtime.GetPodName("kart-io-service")

		args := []interface{}{
			slog.String("pod", podName),
			slog.String("job", "kart-io-logger"),
		}

		// Add namespace if running in Kubernetes
		if runtime.IsKubernetes() {
			if namespace := runtime.GetNamespace(); namespace != "" {
				args = append(args, slog.String("ns", namespace))
			}
		}

		logger = logger.With(args...)
	}

	return &SlogLogger{
		logger:            logger,
		level:             level,
		atomicLevel:       atomicLevel, // Store for dynamic level changes
		mapper:            fields.NewFieldMapper(),
		callerSkip:        0,
		isGlobalCall:      false, // Default to direct call
		disableStacktrace: opt.DisableStacktrace,
		otlpProvider:      otlpProvider,
		initialFields:     serviceFields, // Store InitialFields for OTLP export
		persistentFields:  make(map[string]interface{}),
		managedWriter:     managedWriter, // Store for cleanup
	}, nil
}

// Debug logs a debug message.
func (l *SlogLogger) Debug(args ...interface{}) {
	l.logWithOptionalCaller(core.DebugLevel, formatArgs(args...), nil)
}

// Info logs an info message.
func (l *SlogLogger) Info(args ...interface{}) {
	l.logWithOptionalCaller(core.InfoLevel, formatArgs(args...), nil)
}

// Warn logs a warning message.
func (l *SlogLogger) Warn(args ...interface{}) {
	l.logWithOptionalCaller(core.WarnLevel, formatArgs(args...), nil)
}

// Error logs an error message.
func (l *SlogLogger) Error(args ...interface{}) {
	l.logWithOptionalCaller(core.ErrorLevel, formatArgs(args...), nil)
}

// Fatal logs a fatal message and exits.
func (l *SlogLogger) Fatal(args ...interface{}) {
	l.logWithOptionalCaller(core.FatalLevel, formatArgs(args...), nil)
	os.Exit(1)
}

// Debugf logs a formatted debug message.
func (l *SlogLogger) Debugf(template string, args ...interface{}) {
	l.logWithOptionalCaller(core.DebugLevel, fmt.Sprintf(template, args...), nil)
}

// Infof logs a formatted info message.
func (l *SlogLogger) Infof(template string, args ...interface{}) {
	l.logWithOptionalCaller(core.InfoLevel, fmt.Sprintf(template, args...), nil)
}

// Warnf logs a formatted warning message.
func (l *SlogLogger) Warnf(template string, args ...interface{}) {
	l.logWithOptionalCaller(core.WarnLevel, fmt.Sprintf(template, args...), nil)
}

// Errorf logs a formatted error message.
func (l *SlogLogger) Errorf(template string, args ...interface{}) {
	l.logWithOptionalCaller(core.ErrorLevel, fmt.Sprintf(template, args...), nil)
}

// Fatalf logs a formatted fatal message and exits.
func (l *SlogLogger) Fatalf(template string, args ...interface{}) {
	l.logWithOptionalCaller(core.FatalLevel, fmt.Sprintf(template, args...), nil)
	os.Exit(1)
}

// Debugw logs a debug message with structured fields.
func (l *SlogLogger) Debugw(msg string, keysAndValues ...interface{}) {
	attrs := l.convertToSlogAttrs(keysAndValues...)
	l.logWithOptionalCaller(core.DebugLevel, msg, attrs)
	l.sendToOTLP(core.DebugLevel, msg, keysAndValues...)
}

// Infow logs an info message with structured fields.
func (l *SlogLogger) Infow(msg string, keysAndValues ...interface{}) {
	attrs := l.convertToSlogAttrs(keysAndValues...)
	l.logWithOptionalCaller(core.InfoLevel, msg, attrs)
	l.sendToOTLP(core.InfoLevel, msg, keysAndValues...)
}

// Warnw logs a warning message with structured fields.
func (l *SlogLogger) Warnw(msg string, keysAndValues ...interface{}) {
	attrs := l.convertToSlogAttrs(keysAndValues...)
	l.logWithOptionalCaller(core.WarnLevel, msg, attrs)
	l.sendToOTLP(core.WarnLevel, msg, keysAndValues...)
}

// Errorw logs an error message with structured fields.
func (l *SlogLogger) Errorw(msg string, keysAndValues ...interface{}) {
	attrs := l.convertToSlogAttrs(keysAndValues...)
	l.logWithOptionalCaller(core.ErrorLevel, msg, attrs)
	l.sendToOTLP(core.ErrorLevel, msg, keysAndValues...)
}

// Fatalw logs a fatal message with structured fields and exits.
func (l *SlogLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	attrs := l.convertToSlogAttrs(keysAndValues...)
	l.logWithOptionalCaller(core.FatalLevel, msg, attrs)
	l.sendToOTLP(core.FatalLevel, msg, keysAndValues...)
	os.Exit(1)
}

// With creates a child logger with the specified key-value pairs.
func (l *SlogLogger) With(keysAndValues ...interface{}) core.Logger {
	newLogger := l.logger.With(l.convertToSlogAttrs(keysAndValues...)...)
	return &SlogLogger{
		logger:            newLogger,
		level:             l.level,
		atomicLevel:       l.atomicLevel, // Share atomic level with parent
		mapper:            l.mapper,
		callerSkip:        l.callerSkip,
		isGlobalCall:      l.isGlobalCall, // Inherit from parent
		disableStacktrace: l.disableStacktrace,
		otlpProvider:      l.otlpProvider,
		initialFields:     l.initialFields, // Copy initialFields to child logger
		persistentFields:  l.mergeWithPersistentFields(keysAndValues...),
		managedWriter:     nil, // Child loggers don't own the managed writer
	}
}

// WithCtx creates a child logger with context and key-value pairs.
func (l *SlogLogger) WithCtx(ctx context.Context, keysAndValues ...interface{}) core.Logger {
	// Slog doesn't have a direct equivalent, so we'll create a logger with the fields
	return l.With(keysAndValues...)
}

// WithCallerSkip creates a child logger that skips additional stack frames.
func (l *SlogLogger) WithCallerSkip(skip int) core.Logger {
	return &SlogLogger{
		logger:            l.logger,
		level:             l.level,
		atomicLevel:       l.atomicLevel, // Share atomic level with parent
		mapper:            l.mapper,
		callerSkip:        l.callerSkip + skip,
		isGlobalCall:      l.isGlobalCall, // Preserve global call flag
		disableStacktrace: l.disableStacktrace,
		otlpProvider:      l.otlpProvider,
		initialFields:     l.initialFields,    // Preserve initial fields
		persistentFields:  l.persistentFields, // Preserve persistent fields
		managedWriter:     nil,                // Child loggers don't own the managed writer
	}
}

// SetLevel sets the minimum logging level dynamically.
func (l *SlogLogger) SetLevel(level core.Level) {
	l.level = level
	if l.atomicLevel != nil {
		atomic.StoreInt64(l.atomicLevel, int64(mapToSlogLevel(level)))
	}
}

// Flush flushes any buffered log entries and closes file resources if this is the root logger.
func (l *SlogLogger) Flush() error {
	// slog automatically flushes to the underlying writer
	// Close managed files if available (only on root logger, not child loggers)
	if l.managedWriter != nil {
		err := l.managedWriter.Close()
		if err != nil {
			// In test environments, file closing often fails with "file already closed"
			// or "bad file descriptor" - this is expected and should be ignored
			if strings.Contains(err.Error(), "file already closed") ||
				strings.Contains(err.Error(), "bad file descriptor") {
				return nil
			}
		}
		return err
	}
	return nil
}

// Helper functions

func formatArgs(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		return anyToString(args[0])
	}

	var parts []string
	for _, arg := range args {
		parts = append(parts, anyToString(arg))
	}
	return strings.Join(parts, " ")
}

func anyToString(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	if s, ok := v.(string); ok {
		return s
	}
	return slog.AnyValue(v).String()
}

func (l *SlogLogger) convertToSlogAttrs(keysAndValues ...interface{}) []interface{} {
	attrs := make([]interface{}, 0, len(keysAndValues))

	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 >= len(keysAndValues) {
			// Odd number of arguments, use empty value for last key
			attrs = append(attrs, slog.Any(anyToString(keysAndValues[i]), nil))
			break
		}

		key := anyToString(keysAndValues[i])
		value := keysAndValues[i+1]

		// Apply field mapping for consistency
		if mappedKey := l.getStandardFieldName(key); mappedKey != "" {
			key = mappedKey
		}

		attrs = append(attrs, slog.Any(key, value))
	}

	return attrs
}

func (l *SlogLogger) getStandardFieldName(fieldName string) string {
	coreMapping := l.mapper.MapCoreFields()
	if mapped, exists := coreMapping[fieldName]; exists {
		return mapped
	}

	tracingMapping := l.mapper.MapTracingFields()
	if mapped, exists := tracingMapping[fieldName]; exists {
		return mapped
	}

	return fieldName // Return original if no mapping found
}

func mapToSlogLevel(level core.Level) slog.Level {
	switch level {
	case core.DebugLevel:
		return slog.LevelDebug
	case core.InfoLevel:
		return slog.LevelInfo
	case core.WarnLevel:
		return slog.LevelWarn
	case core.ErrorLevel:
		return slog.LevelError
	case core.FatalLevel:
		return slog.LevelError // slog doesn't have Fatal level
	default:
		return slog.LevelInfo
	}
}

func mapSlogLevelToCoreLevel(level slog.Level) core.Level {
	switch level {
	case slog.LevelDebug:
		return core.DebugLevel
	case slog.LevelInfo:
		return core.InfoLevel
	case slog.LevelWarn:
		return core.WarnLevel
	case slog.LevelError:
		return core.ErrorLevel
	default:
		return core.InfoLevel
	}
}

func createManagedOutputWriters(paths []string) (*managedWriter, error) {
	if len(paths) == 0 {
		return &managedWriter{
			writers: []io.Writer{os.Stdout},
			closers: []io.Closer{},
		}, nil
	}

	var writers []io.Writer
	var closers []io.Closer

	for _, path := range paths {
		switch strings.ToLower(path) {
		case "stdout", "":
			writers = append(writers, os.Stdout)
		case "stderr":
			writers = append(writers, os.Stderr)
		default:
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				// Close any already opened files on error
				for _, closer := range closers {
					closer.Close()
				}
				return nil, err
			}
			writers = append(writers, file)
			closers = append(closers, file)
		}
	}

	return &managedWriter{
		writers: writers,
		closers: closers,
	}, nil
}

// standardizedHandler wraps slog.Handler to ensure field standardization
type standardizedHandler struct {
	handler           slog.Handler
	mapper            *fields.FieldMapper
	disableCaller     bool
	disableStacktrace bool
	otlpProvider      *otlp.LoggerProvider
}

func (h *standardizedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *standardizedHandler) Handle(ctx context.Context, record slog.Record) error {
	// Create a new record with standardized field names
	newRecord := slog.Record{
		Time:    record.Time,
		Level:   record.Level,
		Message: record.Message,
		PC:      record.PC,
	}

	// Add standardized engine identifier
	newRecord.AddAttrs(slog.Attr{
		Key:   "engine",
		Value: slog.StringValue("slog"),
	})

	// Collect all attributes for OTLP export (including With() fields)
	attributes := make(map[string]interface{})

	// Map user-defined fields using our field standardization system
	record.Attrs(func(attr slog.Attr) bool {
		standardKey := h.getStandardFieldName(attr.Key)
		newRecord.AddAttrs(slog.Attr{
			Key:   standardKey,
			Value: attr.Value,
		})

		// Collect for OTLP export
		attributes[standardKey] = attr.Value.Any()
		return true
	})

	// Add engine info to OTLP attributes
	attributes["engine"] = "slog"

	// Send to OTLP if provider is available
	if h.otlpProvider != nil {
		level := mapSlogLevelToCoreLevel(record.Level)
		if err := h.otlpProvider.SendLogRecord(level, record.Message, attributes); err != nil {
			// Use thread-safe error logger to prevent concurrent write issues
			errors.GetDefaultErrorLogger().LogOTLPError("Failed to export log", err)
		}
	}

	return h.handler.Handle(ctx, newRecord)
}

func (h *standardizedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	standardizedAttrs := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		standardizedAttrs[i] = slog.Attr{
			Key:   h.getStandardFieldName(attr.Key),
			Value: attr.Value,
		}
	}
	return &standardizedHandler{
		handler:           h.handler.WithAttrs(standardizedAttrs),
		mapper:            h.mapper,
		disableCaller:     h.disableCaller,
		disableStacktrace: h.disableStacktrace,
		otlpProvider:      h.otlpProvider,
	}
}

func (h *standardizedHandler) WithGroup(name string) slog.Handler {
	return &standardizedHandler{
		handler:           h.handler.WithGroup(name),
		mapper:            h.mapper,
		disableCaller:     h.disableCaller,
		disableStacktrace: h.disableStacktrace,
		otlpProvider:      h.otlpProvider,
	}
}

func (h *standardizedHandler) getStandardFieldName(fieldName string) string {
	coreMapping := h.mapper.MapCoreFields()
	if mapped, exists := coreMapping[fieldName]; exists {
		return mapped
	}

	tracingMapping := h.mapper.MapTracingFields()
	if mapped, exists := tracingMapping[fieldName]; exists {
		return mapped
	}

	return fieldName // Return original if no mapping found
}

// getCaller returns the caller information for the SlogLogger
func (l *SlogLogger) getCaller() string {
	if l == nil {
		return ""
	}

	// Check if this is a call through global logger function
	// by looking at the call stack
	var pcs [10]uintptr
	n := goruntime.Callers(1, pcs[:])
	if n == 0 {
		return ""
	}

	fs := goruntime.CallersFrames(pcs[:n])
	hasGlobalCall := false

	// Check if there's a global logger function in the call stack
	for i := 0; i < n; i++ {
		if f, more := fs.Next(); more || i == n-1 {
			if strings.Contains(f.File, "github.com/kart-io/logger/logger.go") {
				hasGlobalCall = true
				break
			}
		}
	}

	// Determine skip based on call type
	var skip int
	if hasGlobalCall {
		skip = 4 + l.callerSkip // getCaller -> SlogLogger method -> global function -> actual caller
	} else {
		skip = 3 + l.callerSkip // getCaller -> SlogLogger method -> actual caller
	}

	var pcs2 [1]uintptr
	if goruntime.Callers(skip, pcs2[:]) > 0 {
		fs2 := goruntime.CallersFrames(pcs2[:1])
		if f, _ := fs2.Next(); f.File != "" {
			// Extract just the filename from the full path
			file := f.File
			if idx := strings.LastIndex(file, "/"); idx >= 0 {
				if idx2 := strings.LastIndex(file[:idx], "/"); idx2 >= 0 {
					file = file[idx2+1:] // Keep last two path segments
				}
			}

			return fmt.Sprintf("%s:%d", file, f.Line)
		}
	}

	return ""
}

// getStacktrace returns the stack trace for error/fatal level logs
func (l *SlogLogger) getStacktrace() string {
	if l == nil || l.disableStacktrace {
		return ""
	}

	// Skip frames: getStacktrace -> SlogLogger method -> actual caller
	const baseSkip = 3
	skip := baseSkip + l.callerSkip

	var pcs [10]uintptr
	n := goruntime.Callers(skip, pcs[:])
	if n == 0 {
		return ""
	}

	fs := goruntime.CallersFrames(pcs[:n])
	var stackTrace strings.Builder

	for {
		f, more := fs.Next()

		// Extract function name and location
		funcName := f.Function
		if idx := strings.LastIndex(funcName, "/"); idx >= 0 {
			funcName = funcName[idx+1:]
		}

		file := f.File
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			if idx2 := strings.LastIndex(file[:idx], "/"); idx2 >= 0 {
				file = file[idx2+1:] // Keep last two path segments
			}
		}

		if stackTrace.Len() > 0 {
			stackTrace.WriteString("\\n")
		}
		stackTrace.WriteString(fmt.Sprintf("%s\\n\\t%s:%d", funcName, file, f.Line))

		if !more {
			break
		}
	}

	return stackTrace.String()
}

// sendToOTLP sends log data to OTLP as a log record.
func (l *SlogLogger) sendToOTLP(level core.Level, msg string, keysAndValues ...interface{}) {
	if l.otlpProvider == nil {
		return
	}

	// Convert keysAndValues to map
	attributes := make(map[string]interface{})

	// First, add initial fields from logger configuration
	for k, v := range l.initialFields {
		attributes[k] = v
	}

	// Then, add persistent fields from With() (these can override initial fields)
	for k, v := range l.persistentFields {
		attributes[k] = v
	}

	// Then add current call fields (these override persistent fields if same key)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 >= len(keysAndValues) {
			break
		}

		key := anyToString(keysAndValues[i])
		value := keysAndValues[i+1]

		// Apply field mapping
		standardKey := l.getStandardFieldName(key)
		attributes[standardKey] = value
	}

	// Send log record to OTLP
	if err := l.otlpProvider.SendLogRecord(level, msg, attributes); err != nil {
		// Use thread-safe error logger to prevent concurrent write issues
		errors.GetDefaultErrorLogger().LogOTLPError("Failed to export log", err)
	}
}

// mergeWithPersistentFields merges persistent fields with new fields for With() method
func (l *SlogLogger) mergeWithPersistentFields(keysAndValues ...interface{}) map[string]interface{} {
	newPersistentFields := make(map[string]interface{})
	// Copy existing persistent fields
	for k, v := range l.persistentFields {
		newPersistentFields[k] = v
	}
	// Add new fields
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := anyToString(keysAndValues[i])
			standardKey := l.getStandardFieldName(key)
			newPersistentFields[standardKey] = keysAndValues[i+1]
		}
	}
	return newPersistentFields
}

// logWithOptionalCaller executes optimized logging with caller detection
func (l *SlogLogger) logWithOptionalCaller(level core.Level, msg string, attrs []any) {
	if attrs == nil {
		attrs = []any{}
	}

	// Determine caller skip based on call path
	var skip int
	if l.isGlobalCall {
		skip = 5 + l.callerSkip // getCallerWithSkip -> logWithOptionalCaller -> Info -> getOptimizedGlobal -> logger.Info -> actual caller
	} else {
		// Check if call is coming from integration adapter by examining stack
		if l.isFromIntegration() {
			skip = 6 + l.callerSkip // getCallerWithSkip -> logWithOptionalCaller -> Info -> adapter.method -> integration.method -> actual caller
		} else {
			skip = 4 + l.callerSkip // getCallerWithSkip -> logWithOptionalCaller -> Info -> actual caller
		}
	}

	if caller := l.getCallerWithSkip(skip); caller != "" {
		attrs = append(attrs, slog.String(fields.CallerField, caller))
	}

	// Add stacktrace for error and fatal levels
	if (level == core.ErrorLevel || level == core.FatalLevel) && !l.disableStacktrace {
		if stacktrace := l.getStacktrace(); stacktrace != "" {
			attrs = append(attrs, slog.String(fields.StacktraceField, stacktrace))
		}
	}

	// Log at appropriate level
	switch level {
	case core.DebugLevel:
		l.logger.DebugContext(context.Background(), msg, attrs...)
	case core.InfoLevel:
		l.logger.InfoContext(context.Background(), msg, attrs...)
	case core.WarnLevel:
		l.logger.WarnContext(context.Background(), msg, attrs...)
	case core.ErrorLevel:
		l.logger.ErrorContext(context.Background(), msg, attrs...)
	case core.FatalLevel:
		l.logger.ErrorContext(context.Background(), msg, attrs...)
	}
}

// getCallerWithSkip returns caller info with specified skip levels
func (l *SlogLogger) getCallerWithSkip(skip int) string {
	var pcs [1]uintptr
	if goruntime.Callers(skip, pcs[:]) > 0 {
		fs := goruntime.CallersFrames(pcs[:1])
		if f, _ := fs.Next(); f.File != "" {
			// Extract just the filename from the full path
			file := f.File
			if idx := strings.LastIndex(file, "/"); idx >= 0 {
				if idx2 := strings.LastIndex(file[:idx], "/"); idx2 >= 0 {
					file = file[idx2+1:] // Keep last two path segments
				}
			}
			return fmt.Sprintf("%s:%d", file, f.Line)
		}
	}
	return ""
}

// isFromIntegration checks if the logging call originates from an integration adapter
func (l *SlogLogger) isFromIntegration() bool {
	// Check call stack for integration package paths
	var pcs [10]uintptr
	n := goruntime.Callers(1, pcs[:])
	if n == 0 {
		return false
	}

	frames := goruntime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.File, "/integrations/") {
			return true
		}
		if !more {
			break
		}
	}
	return false
}

// CreateGlobalCallLogger creates a logger optimized for global function calls
func (l *SlogLogger) CreateGlobalCallLogger() core.Logger {
	return &SlogLogger{
		logger:            l.logger,
		level:             l.level,
		atomicLevel:       l.atomicLevel,
		mapper:            l.mapper,
		callerSkip:        l.callerSkip,
		isGlobalCall:      true, // Mark as global call
		disableStacktrace: l.disableStacktrace,
		otlpProvider:      l.otlpProvider,
		initialFields:     l.initialFields,
		persistentFields:  l.persistentFields,
		managedWriter:     nil, // Child loggers don't own the managed writer
	}
}
