package grpc

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kart-io/sentinel-x/pkg/infra/tracing"
)

const (
	// TracerName is the name of the tracer for gRPC interceptors.
	TracerName = "github.com/kart-io/sentinel-x/pkg/infra/middleware/grpc"
)

// TracingInterceptorOptions configures the tracing interceptor.
type TracingInterceptorOptions struct {
	// TracerName is the name to use for the tracer.
	TracerName string

	// SpanNameFormatter formats the span name from the method.
	SpanNameFormatter func(fullMethod string) string

	// AttributeExtractor extracts custom attributes from the request.
	AttributeExtractor func(ctx context.Context, fullMethod string, req interface{}) []attribute.KeyValue

	// SkipMethods is a list of full method names to skip tracing.
	SkipMethods []string
}

// TracingInterceptorOption is a functional option for TracingInterceptorOptions.
type TracingInterceptorOption func(*TracingInterceptorOptions)

// NewTracingInterceptorOptions creates default tracing interceptor options.
func NewTracingInterceptorOptions() *TracingInterceptorOptions {
	return &TracingInterceptorOptions{
		TracerName:        TracerName,
		SpanNameFormatter: defaultGRPCSpanNameFormatter,
		SkipMethods:       []string{},
	}
}

// WithGRPCTracerName sets the tracer name.
func WithGRPCTracerName(name string) TracingInterceptorOption {
	return func(o *TracingInterceptorOptions) {
		o.TracerName = name
	}
}

// WithGRPCSpanNameFormatter sets the span name formatter.
func WithGRPCSpanNameFormatter(formatter func(fullMethod string) string) TracingInterceptorOption {
	return func(o *TracingInterceptorOptions) {
		o.SpanNameFormatter = formatter
	}
}

// WithGRPCAttributeExtractor sets a custom attribute extractor.
func WithGRPCAttributeExtractor(extractor func(ctx context.Context, fullMethod string, req interface{}) []attribute.KeyValue) TracingInterceptorOption {
	return func(o *TracingInterceptorOptions) {
		o.AttributeExtractor = extractor
	}
}

// WithGRPCSkipMethods sets methods to skip tracing.
func WithGRPCSkipMethods(methods ...string) TracingInterceptorOption {
	return func(o *TracingInterceptorOptions) {
		o.SkipMethods = methods
	}
}

// UnaryTracingInterceptor creates a unary server interceptor for distributed tracing.
//
// This interceptor:
// - Extracts trace context from gRPC metadata
// - Creates a new span for each RPC call
// - Adds standard gRPC attributes (service, method, status code)
// - Propagates trace context through the RPC lifecycle
// - Records errors and exceptions in spans
//
// Usage:
//
//	server := grpc.NewServer(
//	    grpc.UnaryInterceptor(grpcmw.UnaryTracingInterceptor()),
//	)
func UnaryTracingInterceptor(opts ...TracingInterceptorOption) grpc.UnaryServerInterceptor {
	options := NewTracingInterceptorOptions()
	for _, opt := range opts {
		opt(options)
	}

	skipMap := make(map[string]struct{})
	for _, method := range options.SkipMethods {
		skipMap[method] = struct{}{}
	}

	propagator := tracing.GetGlobalTextMapPropagator()

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if method should be skipped
		if _, skip := skipMap[info.FullMethod]; skip {
			return handler(ctx, req)
		}

		// Extract trace context from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			ctx = propagator.Extract(ctx, &metadataCarrier{md: md})
		}

		// Start span
		spanName := options.SpanNameFormatter(info.FullMethod)
		ctx, span := tracing.StartSpanWithKind(
			ctx,
			options.TracerName,
			spanName,
			trace.SpanKindServer,
		)
		defer span.End()

		// Add standard gRPC attributes
		attrs := []attribute.KeyValue{
			semconv.RPCSystemGRPC,
			semconv.RPCService(extractServiceName(info.FullMethod)),
			semconv.RPCMethod(extractMethodName(info.FullMethod)),
		}

		// Add custom attributes if extractor is provided
		if options.AttributeExtractor != nil {
			customAttrs := options.AttributeExtractor(ctx, info.FullMethod, req)
			attrs = append(attrs, customAttrs...)
		}

		span.SetAttributes(attrs...)

		// Call handler
		resp, err := handler(ctx, req)

		// Set status based on error
		if err != nil {
			// Get gRPC status
			st, _ := status.FromError(err)

			// Add status code attribute
			span.SetAttributes(attribute.Int(string(semconv.RPCGRPCStatusCodeKey), int(st.Code())))

			// Record error
			span.RecordError(err)
			span.SetStatus(codes.Error, st.Message())
		} else {
			// Success
			span.SetAttributes(semconv.RPCGRPCStatusCodeOk) // OK
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}

// StreamTracingInterceptor creates a stream server interceptor for distributed tracing.
//
// This interceptor:
// - Extracts trace context from gRPC metadata
// - Creates a new span for the stream
// - Adds standard gRPC attributes
// - Propagates trace context through the stream lifecycle
//
// Usage:
//
//	server := grpc.NewServer(
//	    grpc.StreamInterceptor(grpcmw.StreamTracingInterceptor()),
//	)
func StreamTracingInterceptor(opts ...TracingInterceptorOption) grpc.StreamServerInterceptor {
	options := NewTracingInterceptorOptions()
	for _, opt := range opts {
		opt(options)
	}

	skipMap := make(map[string]struct{})
	for _, method := range options.SkipMethods {
		skipMap[method] = struct{}{}
	}

	propagator := tracing.GetGlobalTextMapPropagator()

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method should be skipped
		if _, skip := skipMap[info.FullMethod]; skip {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		// Extract trace context from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			ctx = propagator.Extract(ctx, &metadataCarrier{md: md})
		}

		// Start span
		spanName := options.SpanNameFormatter(info.FullMethod)
		ctx, span := tracing.StartSpanWithKind(
			ctx,
			options.TracerName,
			spanName,
			trace.SpanKindServer,
		)
		defer span.End()

		// Add standard gRPC attributes
		attrs := []attribute.KeyValue{
			semconv.RPCSystemGRPC,
			semconv.RPCService(extractServiceName(info.FullMethod)),
			semconv.RPCMethod(extractMethodName(info.FullMethod)),
			attribute.Bool("rpc.grpc.stream", true),
		}

		// Add stream type attributes
		if info.IsClientStream && info.IsServerStream {
			attrs = append(attrs, attribute.String("rpc.grpc.stream_type", "bidi"))
		} else if info.IsClientStream {
			attrs = append(attrs, attribute.String("rpc.grpc.stream_type", "client"))
		} else if info.IsServerStream {
			attrs = append(attrs, attribute.String("rpc.grpc.stream_type", "server"))
		}

		span.SetAttributes(attrs...)

		// Wrap stream with new context
		wrapped := &tracingServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Call handler
		err := handler(srv, wrapped)

		// Set status based on error
		if err != nil {
			// Get gRPC status
			st, _ := status.FromError(err)

			// Add status code attribute
			span.SetAttributes(attribute.Int(string(semconv.RPCGRPCStatusCodeKey), int(st.Code())))

			// Record error
			span.RecordError(err)
			span.SetStatus(codes.Error, st.Message())
		} else {
			// Success
			span.SetAttributes(semconv.RPCGRPCStatusCodeOk) // OK
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// tracingServerStream wraps a grpc.ServerStream with a custom context.
type tracingServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context with tracing information.
func (s *tracingServerStream) Context() context.Context {
	return s.ctx
}

// metadataCarrier implements propagation.TextMapCarrier for gRPC metadata.
type metadataCarrier struct {
	md metadata.MD
}

// Get retrieves a value from metadata.
func (c *metadataCarrier) Get(key string) string {
	values := c.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Set sets a value in metadata.
func (c *metadataCarrier) Set(key, value string) {
	c.md.Set(key, value)
}

// Keys returns all keys in metadata.
func (c *metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for key := range c.md {
		keys = append(keys, key)
	}
	return keys
}

// extractServiceName extracts the service name from the full method.
// Full method format: "/package.Service/Method"
func extractServiceName(fullMethod string) string {
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return fullMethod
}

// extractMethodName extracts the method name from the full method.
// Full method format: "/package.Service/Method"
func extractMethodName(fullMethod string) string {
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return fullMethod
}

// defaultGRPCSpanNameFormatter creates a span name from the gRPC method.
func defaultGRPCSpanNameFormatter(fullMethod string) string {
	// Format: "Service/Method"
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 3 {
		return fmt.Sprintf("%s/%s", extractServiceName(fullMethod), parts[2])
	}
	return fullMethod
}

// UnaryClientTracingInterceptor creates a unary client interceptor for distributed tracing.
//
// This interceptor:
// - Injects trace context into outgoing gRPC metadata
// - Creates a new span for each outgoing RPC call
// - Adds standard gRPC attributes
//
// Usage:
//
//	conn, err := grpc.Dial(
//	    target,
//	    grpc.WithUnaryInterceptor(grpcmw.UnaryClientTracingInterceptor()),
//	)
func UnaryClientTracingInterceptor(opts ...TracingInterceptorOption) grpc.UnaryClientInterceptor {
	options := NewTracingInterceptorOptions()
	for _, opt := range opts {
		opt(options)
	}

	propagator := tracing.GetGlobalTextMapPropagator()

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Start span
		spanName := options.SpanNameFormatter(method)
		ctx, span := tracing.StartSpanWithKind(
			ctx,
			options.TracerName,
			spanName,
			trace.SpanKindClient,
		)
		defer span.End()

		// Add standard gRPC attributes
		span.SetAttributes(
			semconv.RPCSystemGRPC,
			semconv.RPCService(extractServiceName(method)),
			semconv.RPCMethod(extractMethodName(method)),
		)

		// Inject trace context into metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		carrier := &metadataCarrier{md: md}
		propagator.Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Call invoker
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Set status based on error
		if err != nil {
			st, _ := status.FromError(err)
			span.SetAttributes(attribute.Int(string(semconv.RPCGRPCStatusCodeKey), int(st.Code())))
			span.RecordError(err)
			span.SetStatus(codes.Error, st.Message())
		} else {
			span.SetAttributes(semconv.RPCGRPCStatusCodeOk)
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// StreamClientTracingInterceptor creates a stream client interceptor for distributed tracing.
//
// Usage:
//
//	conn, err := grpc.Dial(
//	    target,
//	    grpc.WithStreamInterceptor(grpcmw.StreamClientTracingInterceptor()),
//	)
func StreamClientTracingInterceptor(opts ...TracingInterceptorOption) grpc.StreamClientInterceptor {
	options := NewTracingInterceptorOptions()
	for _, opt := range opts {
		opt(options)
	}

	propagator := tracing.GetGlobalTextMapPropagator()

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Start span
		spanName := options.SpanNameFormatter(method)
		ctx, span := tracing.StartSpanWithKind(
			ctx,
			options.TracerName,
			spanName,
			trace.SpanKindClient,
		)

		// Add standard gRPC attributes
		attrs := []attribute.KeyValue{
			semconv.RPCSystemGRPC,
			semconv.RPCService(extractServiceName(method)),
			semconv.RPCMethod(extractMethodName(method)),
			attribute.Bool("rpc.grpc.stream", true),
		}

		// Add stream type attributes
		if desc.ClientStreams && desc.ServerStreams {
			attrs = append(attrs, attribute.String("rpc.grpc.stream_type", "bidi"))
		} else if desc.ClientStreams {
			attrs = append(attrs, attribute.String("rpc.grpc.stream_type", "client"))
		} else if desc.ServerStreams {
			attrs = append(attrs, attribute.String("rpc.grpc.stream_type", "server"))
		}

		span.SetAttributes(attrs...)

		// Inject trace context into metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		carrier := &metadataCarrier{md: md}
		propagator.Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Create stream
		cs, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return nil, err
		}

		return &tracingClientStream{
			ClientStream: cs,
			span:         span,
		}, nil
	}
}

// tracingClientStream wraps a grpc.ClientStream to end the span when the stream is done.
type tracingClientStream struct {
	grpc.ClientStream
	span trace.Span
}

func (s *tracingClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		if err != context.Canceled {
			st, _ := status.FromError(err)
			s.span.SetAttributes(attribute.Int(string(semconv.RPCGRPCStatusCodeKey), int(st.Code())))
			s.span.RecordError(err)
			s.span.SetStatus(codes.Error, st.Message())
		}
		s.span.End()
	}
	return err
}

func (s *tracingClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	if err != nil {
		s.span.RecordError(err)
	}
	return err
}
