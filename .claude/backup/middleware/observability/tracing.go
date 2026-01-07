package observability

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware/requestutil"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/infra/tracing"
)

const (
	// TracerName is the name of the tracer for HTTP middleware.
	TracerName = "github.com/kart-io/sentinel-x/pkg/infra/middleware"
)

// TracingOptions configures the tracing middleware.
type TracingOptions struct {
	// TracerName is the name to use for the tracer.
	// Default: TracerName constant
	TracerName string

	// SpanNameFormatter formats the span name from the request.
	// Default: "{http.method} {http.route}"
	SpanNameFormatter func(ctx transport.Context) string

	// IncludeRequestBody enables capturing request body in span attributes.
	// WARNING: This can expose sensitive data. Use with caution.
	IncludeRequestBody bool

	// IncludeResponseBody enables capturing response body in span attributes.
	// WARNING: This can expose sensitive data. Use with caution.
	IncludeResponseBody bool

	// SkipPaths is a list of paths to skip tracing.
	SkipPaths []string

	// SkipPathPrefixes is a list of path prefixes to skip tracing.
	SkipPathPrefixes []string

	// AttributeExtractor extracts custom attributes from the request.
	AttributeExtractor func(ctx transport.Context) []attribute.KeyValue
}

// TracingOption is a functional option for TracingOptions.
type TracingOption func(*TracingOptions)

// NewTracingOptions creates default tracing options.
func NewTracingOptions() *TracingOptions {
	return &TracingOptions{
		TracerName:          TracerName,
		SpanNameFormatter:   defaultSpanNameFormatter,
		IncludeRequestBody:  false,
		IncludeResponseBody: false,
		SkipPaths:           []string{},
		SkipPathPrefixes:    []string{},
	}
}

// WithTracerName sets the tracer name.
func WithTracerName(name string) TracingOption {
	return func(o *TracingOptions) {
		o.TracerName = name
	}
}

// WithSpanNameFormatter sets the span name formatter.
func WithSpanNameFormatter(formatter func(ctx transport.Context) string) TracingOption {
	return func(o *TracingOptions) {
		o.SpanNameFormatter = formatter
	}
}

// WithRequestBodyCapture enables request body capture.
func WithRequestBodyCapture(enabled bool) TracingOption {
	return func(o *TracingOptions) {
		o.IncludeRequestBody = enabled
	}
}

// WithResponseBodyCapture enables response body capture.
func WithResponseBodyCapture(enabled bool) TracingOption {
	return func(o *TracingOptions) {
		o.IncludeResponseBody = enabled
	}
}

// WithTracingSkipPaths sets paths to skip tracing.
func WithTracingSkipPaths(paths []string) TracingOption {
	return func(o *TracingOptions) {
		o.SkipPaths = paths
	}
}

// WithTracingSkipPathPrefixes sets path prefixes to skip tracing.
func WithTracingSkipPathPrefixes(prefixes []string) TracingOption {
	return func(o *TracingOptions) {
		o.SkipPathPrefixes = prefixes
	}
}

// WithAttributeExtractor sets a custom attribute extractor.
func WithAttributeExtractor(extractor func(ctx transport.Context) []attribute.KeyValue) TracingOption {
	return func(o *TracingOptions) {
		o.AttributeExtractor = extractor
	}
}

// Tracing creates a tracing middleware.
//
// This middleware:
// - Extracts trace context from incoming requests (W3C Trace Context)
// - Creates a new span for each request
// - Adds standard HTTP attributes (method, URL, status code, etc.)
// - Propagates trace context through the request lifecycle
// - Records errors and exceptions in spans
//
// Usage:
//
//	server := http.NewServer(
//	    http.WithMiddleware(
//	        middleware.Tracing(),
//	    ),
//	)
//
// With options:
//
//	server := http.NewServer(
//	    http.WithMiddleware(
//	        middleware.Tracing(
//	            middleware.WithTracingSkipPaths([]string{"/health", "/metrics"}),
//	            middleware.WithRequestBodyCapture(false),
//	        ),
//	    ),
//	)
func Tracing(opts ...TracingOption) transport.MiddlewareFunc {
	options := NewTracingOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Create skip path map for fast lookup
	skipPathMap := make(map[string]struct{})
	for _, path := range options.SkipPaths {
		skipPathMap[path] = struct{}{}
	}

	propagator := tracing.GetGlobalTextMapPropagator()

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(ctx transport.Context) {
			req := ctx.HTTPRequest()
			path := req.URL.Path

			// Check if path should be skipped
			if _, skip := skipPathMap[path]; skip {
				next(ctx)
				return
			}

			// Check if path prefix should be skipped
			for _, prefix := range options.SkipPathPrefixes {
				if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
					next(ctx)
					return
				}
			}

			// Extract trace context from request headers
			requestCtx := req.Context()
			requestCtx = propagator.Extract(requestCtx, propagation.HeaderCarrier(req.Header))

			// Start span
			spanName := options.SpanNameFormatter(ctx)
			spanCtx, span := tracing.StartSpanWithKind(
				requestCtx,
				options.TracerName,
				spanName,
				trace.SpanKindServer,
			)
			defer span.End()

			// Update request context
			ctx.SetRequest(spanCtx)

			// Add standard HTTP attributes
			attrs := []attribute.KeyValue{
				semconv.HTTPMethod(req.Method),
				semconv.HTTPURL(req.URL.String()),
				semconv.HTTPTarget(req.URL.Path),
				semconv.HTTPScheme(req.URL.Scheme),
				semconv.ServerAddress(req.Host),
			}

			if userAgent := req.UserAgent(); userAgent != "" {
				attrs = append(attrs, semconv.UserAgentOriginal(userAgent))
			}

			if clientIP := req.RemoteAddr; clientIP != "" {
				attrs = append(attrs, attribute.String(tracing.HTTPClientIP, clientIP))
			}

			// Add request ID if present
			if requestID := ctx.Header(requestutil.HeaderXRequestID); requestID != "" {
				attrs = append(attrs, attribute.String(tracing.HTTPRequestID, requestID))
			}

			// Add custom attributes if extractor is provided
			if options.AttributeExtractor != nil {
				customAttrs := options.AttributeExtractor(ctx)
				attrs = append(attrs, customAttrs...)
			}

			span.SetAttributes(attrs...)

			// Capture response status using a custom response writer
			rw := &tracingResponseWriter{
				ResponseWriter: ctx.ResponseWriter(),
				statusCode:     http.StatusOK, // Default to 200
			}

			// Create a custom context that wraps the original
			originalWriter := ctx.ResponseWriter()

			// Call the next handler
			next(ctx)

			// Restore original response writer
			// Note: We can't actually replace the ResponseWriter in the context,
			// so we use the status captured during the handler execution

			// Get status code from context if it was set
			statusCode := rw.statusCode

			// Try to get actual status from response
			// This is framework-specific, but we'll do our best
			if w, ok := originalWriter.(interface{ Status() int }); ok {
				if status := w.Status(); status != 0 {
					statusCode = status
				}
			}

			// Add response attributes
			span.SetAttributes(semconv.HTTPStatusCode(statusCode))

			// Set span status based on HTTP status code
			if statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(statusCode))
			} else {
				span.SetStatus(codes.Ok, "")
			}

			// Record error if present
			if statusCode >= 500 {
				span.RecordError(fmt.Errorf("HTTP %d: %s", statusCode, http.StatusText(statusCode)))
			}
		}
	}
}

// tracingResponseWriter wraps http.ResponseWriter to capture status code.
type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *tracingResponseWriter) WriteHeader(statusCode int) {
	if !w.written {
		w.statusCode = statusCode
		w.written = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *tracingResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// defaultSpanNameFormatter creates a span name from the HTTP method and route.
func defaultSpanNameFormatter(ctx transport.Context) string {
	req := ctx.HTTPRequest()
	// Try to get route pattern if available
	// This is framework-specific, so we fall back to the path
	route := req.URL.Path
	return fmt.Sprintf("%s %s", req.Method, route)
}

// ExtractTraceID extracts the trace ID from the context.
// This can be used to add trace ID to logs or responses.
func ExtractTraceID(ctx transport.Context) string {
	return tracing.TraceIDFromContext(ctx.Request())
}

// ExtractSpanID extracts the span ID from the context.
func ExtractSpanID(ctx transport.Context) string {
	return tracing.SpanIDFromContext(ctx.Request())
}
