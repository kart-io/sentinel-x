package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanFromContext returns the current span from the context.
// If no span is found, it returns a non-recording span.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// StartSpan starts a new span with the given name and options.
// It returns the new span and a context containing the span.
func StartSpan(ctx context.Context, tracerName, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName, opts...)
}

// StartSpanWithKind starts a new span with the given name and kind.
func StartSpanWithKind(ctx context.Context, tracerName, spanName string, kind trace.SpanKind, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	opts = append(opts, trace.WithSpanKind(kind))
	return StartSpan(ctx, tracerName, spanName, opts...)
}

// AddSpanAttributes adds attributes to the span in the context.
// If no span is found, this is a no-op.
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// AddSpanEvent adds an event to the span in the context.
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordError records an error on the span in the context.
// It marks the span as failed and adds the error as an event.
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, err.Error())
}

// RecordErrorWithStatus records an error on the span with a custom status message.
func RecordErrorWithStatus(ctx context.Context, err error, statusMsg string, opts ...trace.EventOption) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, statusMsg)
}

// SetSpanStatus sets the status of the span in the context.
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// SetSpanOK marks the span as successful.
func SetSpanOK(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(codes.Ok, "")
}

// SetSpanError marks the span as failed with the given description.
func SetSpanError(ctx context.Context, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(codes.Error, description)
}

// WithSpanContext returns a copy of the parent context with the span context.
// This is useful for propagating trace context across goroutines.
func WithSpanContext(parent, spanCtx context.Context) context.Context {
	return trace.ContextWithSpan(parent, trace.SpanFromContext(spanCtx))
}

// TraceIDFromContext extracts the trace ID from the context.
// Returns an empty string if no trace is active.
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// SpanIDFromContext extracts the span ID from the context.
// Returns an empty string if no span is active.
func SpanIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// IsRecording returns true if the span in the context is recording.
func IsRecording(ctx context.Context) bool {
	span := trace.SpanFromContext(ctx)
	return span.IsRecording()
}

// EndSpan ends the span in the context.
// This is a convenience function that's more explicit than defer span.End().
func EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

// String creates a string attribute.
func String(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// Int creates an int attribute.
func Int(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

// Int64 creates an int64 attribute.
func Int64(key string, value int64) attribute.KeyValue {
	return attribute.Int64(key, value)
}

// Float64 creates a float64 attribute.
func Float64(key string, value float64) attribute.KeyValue {
	return attribute.Float64(key, value)
}

// Bool creates a bool attribute.
func Bool(key string, value bool) attribute.KeyValue {
	return attribute.Bool(key, value)
}

// StringSlice creates a string slice attribute.
func StringSlice(key string, value []string) attribute.KeyValue {
	return attribute.StringSlice(key, value)
}

// IntSlice creates an int slice attribute.
func IntSlice(key string, value []int) attribute.KeyValue {
	return attribute.IntSlice(key, value)
}

// Any creates an attribute from any value.
func Any(key string, value interface{}) attribute.KeyValue {
	return attribute.String(key, fmt.Sprint(value))
}

// Common attribute keys for HTTP and gRPC tracing.
const (
	// HTTP attributes
	HTTPMethod       = "http.method"
	HTTPURL          = "http.url"
	HTTPTarget       = "http.target"
	HTTPHost         = "http.host"
	HTTPScheme       = "http.scheme"
	HTTPStatusCode   = "http.status_code"
	HTTPUserAgent    = "http.user_agent"
	HTTPRequestSize  = "http.request.size"
	HTTPResponseSize = "http.response.size"
	HTTPRoute        = "http.route"
	HTTPClientIP     = "http.client_ip"
	HTTPRequestID    = "http.request_id"
	HTTPRequestBody  = "http.request.body"
	HTTPResponseBody = "http.response.body"

	// gRPC attributes
	RPCSystem           = "rpc.system"
	RPCService          = "rpc.service"
	RPCMethod           = "rpc.method"
	RPCGRPCStatusCode   = "rpc.grpc.status_code"
	RPCGRPCRequestSize  = "rpc.grpc.request.size"
	RPCGRPCResponseSize = "rpc.grpc.response.size"

	// Database attributes
	DBSystem    = "db.system"
	DBName      = "db.name"
	DBStatement = "db.statement"
	DBOperation = "db.operation"
	DBUser      = "db.user"

	// Error attributes
	ErrorType    = "error.type"
	ErrorMessage = "error.message"
	ErrorStack   = "error.stack"

	// Custom application attributes
	UserID        = "user.id"
	UserEmail     = "user.email"
	TenantID      = "tenant.id"
	RequestID     = "request.id"
	CorrelationID = "correlation.id"
)
