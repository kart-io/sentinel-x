// Package logger provides structured logging utilities with context propagation.
package logger

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
)

// contextKey is the type for context keys to avoid collisions.
type contextKey int

const (
	// loggerFieldsKey is the context key for logger fields.
	loggerFieldsKey contextKey = iota
	// contextLoggerKey is the context key for context-scoped logger.
	contextLoggerKey
)

// loggerFields holds structured logging fields extracted from context.
type loggerFields struct {
	fields map[string]interface{}
}

// newLoggerFields creates a new loggerFields instance.
func newLoggerFields() *loggerFields {
	return &loggerFields{
		fields: make(map[string]interface{}),
	}
}

// clone creates a deep copy of loggerFields.
func (lf *loggerFields) clone() *loggerFields {
	newFields := newLoggerFields()
	for k, v := range lf.fields {
		newFields.fields[k] = v
	}
	return newFields
}

// set adds or updates a field.
func (lf *loggerFields) set(key string, value interface{}) {
	lf.fields[key] = value
}

// toSlice converts fields map to a key-value slice for structured logging.
func (lf *loggerFields) toSlice() []interface{} {
	if len(lf.fields) == 0 {
		return nil
	}

	slice := make([]interface{}, 0, len(lf.fields)*2)
	for k, v := range lf.fields {
		slice = append(slice, k, v)
	}
	return slice
}

// getLoggerFields retrieves or creates loggerFields from context.
func getLoggerFields(ctx context.Context) *loggerFields {
	if lf, ok := ctx.Value(loggerFieldsKey).(*loggerFields); ok {
		return lf
	}
	return newLoggerFields()
}

// withField adds a single field to the context.
func withField(ctx context.Context, key string, value interface{}) context.Context {
	lf := getLoggerFields(ctx).clone()
	lf.set(key, value)
	return context.WithValue(ctx, loggerFieldsKey, lf)
}

// WithRequestID adds request_id to the context logger fields.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return withField(ctx, "request_id", requestID)
}

// WithTraceID adds trace_id to the context logger fields.
// This is useful for manual trace ID injection when not using OpenTelemetry.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return withField(ctx, "trace_id", traceID)
}

// WithSpanID adds span_id to the context logger fields.
// This is useful for manual span ID injection when not using OpenTelemetry.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	if spanID == "" {
		return ctx
	}
	return withField(ctx, "span_id", spanID)
}

// WithUserID adds user_id to the context logger fields.
// This is useful for multi-tenant logging and user activity tracking.
func WithUserID(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return withField(ctx, "user_id", userID)
}

// WithTenantID adds tenant_id to the context logger fields.
// This is essential for multi-tenant application logging.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	if tenantID == "" {
		return ctx
	}
	return withField(ctx, "tenant_id", tenantID)
}

// WithError adds structured error fields to the context.
// It extracts error_message and optionally error_type.
func WithError(ctx context.Context, err error) context.Context {
	if err == nil {
		return ctx
	}

	lf := getLoggerFields(ctx).clone()
	lf.set("error_message", err.Error())
	lf.set("error_type", fmt.Sprintf("%T", err))

	return context.WithValue(ctx, loggerFieldsKey, lf)
}

// WithErrorCode adds error_code to the context logger fields.
// Useful for categorizing errors by application-specific error codes.
func WithErrorCode(ctx context.Context, code string) context.Context {
	if code == "" {
		return ctx
	}
	return withField(ctx, "error_code", code)
}

// WithFields adds multiple custom fields to the context at once.
// The fields should be provided as key-value pairs.
func WithFields(ctx context.Context, keysAndValues ...interface{}) context.Context {
	if len(keysAndValues) == 0 {
		return ctx
	}

	if len(keysAndValues)%2 != 0 {
		// Invalid input: odd number of arguments, ignore the last one
		keysAndValues = keysAndValues[:len(keysAndValues)-1]
	}

	lf := getLoggerFields(ctx).clone()
	for i := 0; i < len(keysAndValues); i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			lf.set(key, keysAndValues[i+1])
		}
	}

	return context.WithValue(ctx, loggerFieldsKey, lf)
}

// ExtractOpenTelemetryFields extracts trace_id and span_id from OpenTelemetry span context.
// This function should be called automatically by middleware if OpenTelemetry is in use.
func ExtractOpenTelemetryFields(ctx context.Context) context.Context {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return ctx
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return ctx
	}

	lf := getLoggerFields(ctx).clone()

	if spanCtx.HasTraceID() {
		lf.set("trace_id", spanCtx.TraceID().String())
	}

	if spanCtx.HasSpanID() {
		lf.set("span_id", spanCtx.SpanID().String())
	}

	// Include trace flags if sampled
	if spanCtx.IsSampled() {
		lf.set("trace_sampled", true)
	}

	return context.WithValue(ctx, loggerFieldsKey, lf)
}

// GetContextFields retrieves all logger fields from context as a slice.
// Returns nil if no fields are present.
func GetContextFields(ctx context.Context) []interface{} {
	lf := getLoggerFields(ctx)
	return lf.toSlice()
}

// GetLogger retrieves or creates a context-aware logger.
// The returned logger includes all fields stored in the context.
// This is the primary way to get a logger that includes context fields.
func GetLogger(ctx context.Context) core.Logger {
	// Check if a context-scoped logger already exists
	if ctxLogger, ok := ctx.Value(contextLoggerKey).(core.Logger); ok {
		return ctxLogger
	}

	// Get the global logger
	baseLogger := logger.Global()

	// Get context fields
	fields := GetContextFields(ctx)
	if len(fields) == 0 {
		return baseLogger
	}

	// Create a new logger with context fields
	return baseLogger.With(fields...)
}

// WithLogger stores a pre-configured logger in the context.
// This is useful when you want to create a logger with specific fields
// and reuse it throughout a request lifecycle.
func WithLogger(ctx context.Context, log core.Logger) context.Context {
	return context.WithValue(ctx, contextLoggerKey, log)
}
