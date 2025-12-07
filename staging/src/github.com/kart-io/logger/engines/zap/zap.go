package zap

import (
	"context"
	"fmt"
	goruntime "runtime"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/errors"
	"github.com/kart-io/logger/fields"
	"github.com/kart-io/logger/option"
	"github.com/kart-io/logger/otlp"
	"github.com/kart-io/logger/runtime"
)

// ZapLogger implements the core.Logger interface using Uber's Zap library.
type ZapLogger struct {
	logger           *zap.Logger
	sugar            *zap.SugaredLogger
	level            core.Level
	atomicLevel      *zap.AtomicLevel // For dynamic level changes
	mapper           *fields.FieldMapper
	callerSkip       int
	isGlobalCall     bool // Cache whether this is a global call to avoid runtime detection
	otlpProvider     *otlp.LoggerProvider
	initialFields    map[string]interface{} // Fields from InitialFields config
	persistentFields map[string]interface{} // Fields added via With()
}

// NewZapLogger creates a new Zap-based logger with the provided configuration.
func NewZapLogger(opt *option.LogOption) (core.Logger, error) {
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

	// Create Zap config with atomic level for dynamic changes
	config := createZapConfig(opt, level)

	// Store atomic level for dynamic changes
	atomicLevel := config.Level

	// Create Zap logger
	zapLogger, err := config.Build(
		zap.AddCallerSkip(1), // Base skip for our wrapper methods
	)
	if err != nil {
		return nil, err
	}

	// Use the zapLogger directly as field standardization is handled in EncoderConfig
	standardizedLogger := zapLogger

	// Add engine identifier as a persistent field
	standardizedLogger = standardizedLogger.With(zap.String("engine", "zap"))

	// Add initial fields from configuration with default values
	initialFields := make([]zap.Field, 0)

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
	for key, value := range serviceFields {
		initialFields = append(initialFields, zap.Any(key, value))
	}

	standardizedLogger = standardizedLogger.With(initialFields...)

	// Add basic OTEL fields if OTLP is enabled
	if otlpProvider != nil {
		// Get pod/container/hostname based on deployment environment
		podName := runtime.GetPodName("kart-io-service")

		fields := []zap.Field{
			zap.String("pod", podName),
			zap.String("job", "kart-io-logger"),
		}

		// Add namespace if running in Kubernetes
		if runtime.IsKubernetes() {
			if namespace := runtime.GetNamespace(); namespace != "" {
				fields = append(fields, zap.String("ns", namespace))
			}
		}

		standardizedLogger = standardizedLogger.With(fields...)
	}

	return &ZapLogger{
		logger:           standardizedLogger,
		sugar:            standardizedLogger.Sugar(),
		level:            level,
		atomicLevel:      &atomicLevel,
		mapper:           fields.NewFieldMapper(),
		callerSkip:       0,
		isGlobalCall:     false, // Default to direct call
		otlpProvider:     otlpProvider,
		initialFields:    serviceFields, // Store InitialFields for OTLP export
		persistentFields: make(map[string]interface{}),
	}, nil
}

// Debug logs a debug message.
func (l *ZapLogger) Debug(args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Debug(args...) })
}

// Info logs an info message.
func (l *ZapLogger) Info(args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Info(args...) })
}

// Warn logs a warning message.
func (l *ZapLogger) Warn(args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Warn(args...) })
}

// Error logs an error message.
func (l *ZapLogger) Error(args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Error(args...) })
}

// Fatal logs a fatal message and exits.
func (l *ZapLogger) Fatal(args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Fatal(args...) })
}

// Debugf logs a formatted debug message.
func (l *ZapLogger) Debugf(template string, args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Debugf(template, args...) })
}

// Infof logs a formatted info message.
func (l *ZapLogger) Infof(template string, args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Infof(template, args...) })
}

// Warnf logs a formatted warning message.
func (l *ZapLogger) Warnf(template string, args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Warnf(template, args...) })
}

// Errorf logs a formatted error message.
func (l *ZapLogger) Errorf(template string, args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Errorf(template, args...) })
}

// Fatalf logs a formatted fatal message and exits.
func (l *ZapLogger) Fatalf(template string, args ...interface{}) {
	l.logWithOptionalSkip(func(logger *ZapLogger) { logger.sugar.Fatalf(template, args...) })
}

// Debugw logs a debug message with structured fields.
func (l *ZapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	logger := l.withDynamicCallerSkip().(*ZapLogger)
	logger.sugar.Debugw(msg, logger.standardizeFields(keysAndValues...)...)
	l.sendToOTLP(core.DebugLevel, msg, keysAndValues...)
}

// Infow logs an info message with structured fields.
func (l *ZapLogger) Infow(msg string, keysAndValues ...interface{}) {
	logger := l.withDynamicCallerSkip().(*ZapLogger)
	logger.sugar.Infow(msg, logger.standardizeFields(keysAndValues...)...)
	l.sendToOTLP(core.InfoLevel, msg, keysAndValues...)
}

// Warnw logs a warning message with structured fields.
func (l *ZapLogger) Warnw(msg string, keysAndValues ...interface{}) {
	logger := l.withDynamicCallerSkip().(*ZapLogger)
	logger.sugar.Warnw(msg, logger.standardizeFields(keysAndValues...)...)
	l.sendToOTLP(core.WarnLevel, msg, keysAndValues...)
}

// Errorw logs an error message with structured fields.
func (l *ZapLogger) Errorw(msg string, keysAndValues ...interface{}) {
	logger := l.withDynamicCallerSkip().(*ZapLogger)
	logger.sugar.Errorw(msg, logger.standardizeFields(keysAndValues...)...)
	l.sendToOTLP(core.ErrorLevel, msg, keysAndValues...)
}

// Fatalw logs a fatal message with structured fields and exits.
func (l *ZapLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	logger := l.withDynamicCallerSkip().(*ZapLogger)
	logger.sugar.Fatalw(msg, logger.standardizeFields(keysAndValues...)...)
	l.sendToOTLP(core.FatalLevel, msg, keysAndValues...)
}

// With creates a child logger with the specified key-value pairs.
func (l *ZapLogger) With(keysAndValues ...interface{}) core.Logger {
	standardizedFields := l.standardizeFields(keysAndValues...)
	newSugar := l.sugar.With(standardizedFields...)

	// Merge persistent fields
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

	return &ZapLogger{
		logger:           newSugar.Desugar(),
		sugar:            newSugar,
		level:            l.level,
		atomicLevel:      l.atomicLevel, // Share atomic level with parent
		mapper:           l.mapper,
		callerSkip:       l.callerSkip,
		isGlobalCall:     l.isGlobalCall, // Inherit from parent
		otlpProvider:     l.otlpProvider,
		initialFields:    l.initialFields, // Copy initialFields to child logger
		persistentFields: newPersistentFields,
	}
}

// WithCtx creates a child logger with context and key-value pairs.
func (l *ZapLogger) WithCtx(ctx context.Context, keysAndValues ...interface{}) core.Logger {
	// Zap doesn't have direct context support, so we'll just add the fields
	return l.With(keysAndValues...)
}

// WithCallerSkip creates a child logger that skips additional stack frames.
func (l *ZapLogger) WithCallerSkip(skip int) core.Logger {
	newLogger := l.logger.WithOptions(zap.AddCallerSkip(skip))

	return &ZapLogger{
		logger:           newLogger,
		sugar:            newLogger.Sugar(),
		level:            l.level,
		atomicLevel:      l.atomicLevel, // Share atomic level with parent
		mapper:           l.mapper,
		callerSkip:       l.callerSkip + skip,
		isGlobalCall:     l.isGlobalCall, // Preserve global call flag
		otlpProvider:     l.otlpProvider,
		initialFields:    l.initialFields,    // Preserve initial fields
		persistentFields: l.persistentFields, // Preserve persistent fields
	}
}

// logWithOptionalSkip executes the log function with caller skip optimization
func (l *ZapLogger) logWithOptionalSkip(logFn func(*ZapLogger)) {
	var skipLevel int
	if l.isGlobalCall {
		// Global path: logWithOptionalSkip -> logFn -> zap sugar method -> getOptimizedGlobal -> logger.Info -> actual caller
		skipLevel = 3
	} else {
		// Check if call is coming from integration adapter
		if l.isFromIntegration() {
			// Integration path: logWithOptionalSkip -> logFn -> zap sugar method -> adapter.method -> integration.method -> actual caller
			skipLevel = 4
		} else {
			// Direct call path: logWithOptionalSkip -> logFn -> zap sugar method -> actual caller
			skipLevel = 2
		}
	}

	logger := l.WithCallerSkip(skipLevel).(*ZapLogger)
	logFn(logger)
}

// withDynamicCallerSkip creates a logger with caller skip based on call stack (original implementation)
func (l *ZapLogger) withDynamicCallerSkip() core.Logger {
	return l // For now, just return self without dynamic detection
}

// CreateGlobalCallLogger creates a logger optimized for global function calls
func (l *ZapLogger) CreateGlobalCallLogger() core.Logger {
	return &ZapLogger{
		logger:           l.logger,
		sugar:            l.sugar,
		level:            l.level,
		atomicLevel:      l.atomicLevel,
		mapper:           l.mapper,
		callerSkip:       l.callerSkip,
		isGlobalCall:     true, // Mark as global call
		otlpProvider:     l.otlpProvider,
		initialFields:    l.initialFields,
		persistentFields: l.persistentFields,
	}
}

// SetLevel sets the minimum logging level dynamically.
func (l *ZapLogger) SetLevel(level core.Level) {
	l.level = level
	if l.atomicLevel != nil {
		l.atomicLevel.SetLevel(mapToZapLevel(level))
	}
}

// Flush flushes any buffered log entries.
func (l *ZapLogger) Flush() error {
	err := l.logger.Sync()
	if err != nil {
		// In test environments, sync to stdout/stderr often fails with "bad file descriptor"
		// This is expected and should be ignored
		if strings.Contains(err.Error(), "sync /dev/stdout") ||
			strings.Contains(err.Error(), "sync /dev/stderr") ||
			strings.Contains(err.Error(), "bad file descriptor") {
			return nil
		}
	}
	return err
}

// isFromIntegration checks if the logging call originates from an integration adapter
func (l *ZapLogger) isFromIntegration() bool {
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

// Helper functions

func (l *ZapLogger) standardizeFields(keysAndValues ...interface{}) []interface{} {
	standardized := make([]interface{}, 0, len(keysAndValues))

	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 >= len(keysAndValues) {
			// Odd number of arguments, use empty value for last key
			key := l.getStandardFieldName(anyToString(keysAndValues[i]))
			standardized = append(standardized, key, nil)
			break
		}

		key := anyToString(keysAndValues[i])
		value := keysAndValues[i+1]

		// Apply field mapping for consistency
		standardKey := l.getStandardFieldName(key)
		standardized = append(standardized, standardKey, value)
	}

	return standardized
}

func (l *ZapLogger) getStandardFieldName(fieldName string) string {
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

func anyToString(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	if s, ok := v.(string); ok {
		return s
	}
	// Use fmt.Sprintf for simple string conversion
	return fmt.Sprintf("%v", v)
}

func createZapConfig(opt *option.LogOption, level core.Level) zap.Config {
	// Start with appropriate preset
	var config zap.Config
	if opt.Development {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Override with our specific settings
	config.Level = zap.NewAtomicLevelAt(mapToZapLevel(level))
	config.DisableCaller = opt.DisableCaller
	config.DisableStacktrace = opt.DisableStacktrace

	// Set encoding format
	switch strings.ToLower(opt.Format) {
	case "console", "text":
		config.Encoding = "console"
	case "json":
		config.Encoding = "json"
	default:
		config.Encoding = "json"
	}

	// Configure output paths
	if len(opt.OutputPaths) > 0 {
		config.OutputPaths = normalizeOutputPaths(opt.OutputPaths)
		config.ErrorOutputPaths = normalizeOutputPaths(opt.OutputPaths) // Use same for errors
	}

	// Configure encoder with standardized field names
	config.EncoderConfig = createStandardizedEncoderConfig()

	return config
}

func createStandardizedEncoderConfig() zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()

	// Use our standardized field names
	config.TimeKey = fields.TimestampField
	config.LevelKey = fields.LevelField
	config.MessageKey = fields.MessageField
	config.CallerKey = fields.CallerField
	config.StacktraceKey = fields.StacktraceField

	// Configure time format
	config.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	config.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncodeCaller = zapcore.ShortCallerEncoder

	return config
}

func mapToZapLevel(level core.Level) zapcore.Level {
	switch level {
	case core.DebugLevel:
		return zapcore.DebugLevel
	case core.InfoLevel:
		return zapcore.InfoLevel
	case core.WarnLevel:
		return zapcore.WarnLevel
	case core.ErrorLevel:
		return zapcore.ErrorLevel
	case core.FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func normalizeOutputPaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		switch strings.ToLower(path) {
		case "stdout", "":
			normalized = append(normalized, "stdout")
		case "stderr":
			normalized = append(normalized, "stderr")
		default:
			normalized = append(normalized, path)
		}
	}
	return normalized
}

// sendToOTLP sends log data to OTLP as a log record.
func (l *ZapLogger) sendToOTLP(level core.Level, msg string, keysAndValues ...interface{}) {
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
