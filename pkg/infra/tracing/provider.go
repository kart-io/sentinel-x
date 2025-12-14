package tracing

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	options "github.com/kart-io/sentinel-x/pkg/options/tracing"
)

// Options is re-exported from pkg/options/tracing for convenience.
type Options = options.Options

// SamplerType is re-exported from pkg/options/tracing for convenience.
type SamplerType = options.SamplerType

// ExporterType is re-exported from pkg/options/tracing for convenience.
type ExporterType = options.ExporterType

// NewOptions is re-exported from pkg/options/tracing for convenience.
var NewOptions = options.NewOptions

// Re-export sampler type constants.
const (
	SamplerAlwaysOn    = options.SamplerAlwaysOn
	SamplerAlwaysOff   = options.SamplerAlwaysOff
	SamplerRatio       = options.SamplerRatio
	SamplerParentBased = options.SamplerParentBased
)

// Re-export exporter type constants.
const (
	ExporterOTLPGRPC = options.ExporterOTLPGRPC
	ExporterOTLPHTTP = options.ExporterOTLPHTTP
	ExporterStdout   = options.ExporterStdout
	ExporterNoop     = options.ExporterNoop
)

// Provider manages the OpenTelemetry tracer provider lifecycle.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	opts           *Options
}

// NewProvider creates and initializes a new tracer provider.
func NewProvider(opts *Options) (*Provider, error) {
	if opts == nil {
		opts = NewOptions()
	}

	if err := opts.Complete(); err != nil {
		return nil, fmt.Errorf("failed to complete options: %w", err)
	}

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate options: %w", err)
	}

	if !opts.Enabled {
		// Return a provider with no-op tracer
		return &Provider{
			tracerProvider: sdktrace.NewTracerProvider(),
			opts:           opts,
		}, nil
	}

	// Create resource
	res, err := newResource(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter
	exporter, err := newExporter(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create sampler
	sampler := newSampler(opts)

	// Create batch span processor
	bsp := sdktrace.NewBatchSpanProcessor(
		exporter,
		sdktrace.WithBatchTimeout(opts.BatchTimeout),
		sdktrace.WithMaxExportBatchSize(opts.BatchMaxSize),
		sdktrace.WithExportTimeout(opts.ExportTimeout),
		sdktrace.WithMaxQueueSize(opts.MaxQueueSize),
	)

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithSpanProcessor(bsp),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global text map propagator to W3C Trace Context
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		tracerProvider: tp,
		opts:           opts,
	}, nil
}

// Tracer returns a tracer with the given name.
func (p *Provider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	if p.tracerProvider == nil {
		return otel.Tracer(name, opts...)
	}
	return p.tracerProvider.Tracer(name, opts...)
}

// Shutdown shuts down the tracer provider gracefully.
// It flushes any pending spans and releases resources.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tracerProvider == nil {
		return nil
	}
	return p.tracerProvider.Shutdown(ctx)
}

// ForceFlush flushes any pending spans.
func (p *Provider) ForceFlush(ctx context.Context) error {
	if p.tracerProvider == nil {
		return nil
	}
	return p.tracerProvider.ForceFlush(ctx)
}

// newResource creates a resource with service information.
func newResource(opts *Options) (*resource.Resource, error) {
	attributes := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceName(opts.ServiceName),
			semconv.ServiceVersion(opts.ServiceVersion),
		),
	}

	if opts.ServiceNamespace != "" {
		attributes = append(attributes, resource.WithAttributes(
			semconv.ServiceNamespace(opts.ServiceNamespace),
		))
	}

	if opts.Environment != "" {
		attributes = append(attributes, resource.WithAttributes(
			semconv.DeploymentEnvironment(opts.Environment),
		))
	}

	// Add custom resource attributes
	for k, v := range opts.ResourceAttributes {
		attributes = append(attributes, resource.WithAttributes(
			attribute.String(k, v),
		))
	}

	// Merge with default resource (includes host, process info)
	attributes = append(attributes, resource.WithFromEnv())
	attributes = append(attributes, resource.WithTelemetrySDK())
	attributes = append(attributes, resource.WithHost())
	attributes = append(attributes, resource.WithProcess())

	return resource.New(context.Background(), attributes...)
}

// newExporter creates a trace exporter based on the configuration.
func newExporter(ctx context.Context, opts *Options) (sdktrace.SpanExporter, error) {
	switch opts.ExporterType {
	case ExporterOTLPGRPC:
		return newOTLPGRPCExporter(ctx, opts)
	case ExporterOTLPHTTP:
		return newOTLPHTTPExporter(ctx, opts)
	case ExporterStdout:
		return newStdoutExporter()
	case ExporterNoop:
		return newNoopExporter(), nil
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", opts.ExporterType)
	}
}

// newOTLPGRPCExporter creates an OTLP gRPC exporter.
func newOTLPGRPCExporter(ctx context.Context, opts *Options) (sdktrace.SpanExporter, error) {
	grpcOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(opts.Endpoint),
	}

	if opts.Insecure {
		grpcOpts = append(grpcOpts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
	}

	if len(opts.Headers) > 0 {
		grpcOpts = append(grpcOpts, otlptracegrpc.WithHeaders(opts.Headers))
	}

	// Create OTLP trace client
	client := otlptracegrpc.NewClient(grpcOpts...)

	// Create exporter
	return otlptrace.New(ctx, client)
}

// newOTLPHTTPExporter creates an OTLP HTTP exporter.
func newOTLPHTTPExporter(ctx context.Context, opts *Options) (sdktrace.SpanExporter, error) {
	httpOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(opts.Endpoint),
	}

	if opts.Insecure {
		httpOpts = append(httpOpts, otlptracehttp.WithInsecure())
	}

	if len(opts.Headers) > 0 {
		httpOpts = append(httpOpts, otlptracehttp.WithHeaders(opts.Headers))
	}

	// Create OTLP trace client
	client := otlptracehttp.NewClient(httpOpts...)

	// Create exporter
	return otlptrace.New(ctx, client)
}

// newStdoutExporter creates a stdout exporter for development/debugging.
func newStdoutExporter() (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(os.Stdout),
	)
}

// newNoopExporter creates a no-op exporter.
func newNoopExporter() sdktrace.SpanExporter {
	return &noopExporter{}
}

// noopExporter is a no-op span exporter.
type noopExporter struct{}

func (e *noopExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *noopExporter) Shutdown(_ context.Context) error {
	return nil
}

// newSampler creates a sampler based on the configuration.
func newSampler(opts *Options) sdktrace.Sampler {
	switch opts.SamplerType {
	case SamplerAlwaysOn:
		return sdktrace.AlwaysSample()
	case SamplerAlwaysOff:
		return sdktrace.NeverSample()
	case SamplerRatio:
		return sdktrace.TraceIDRatioBased(opts.SamplerRatio)
	case SamplerParentBased:
		// Parent-based sampling with trace ID ratio as the root sampler
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(opts.SamplerRatio))
	default:
		// Default to parent-based sampling
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

// GetGlobalTracerProvider returns the global tracer provider.
func GetGlobalTracerProvider() trace.TracerProvider {
	return otel.GetTracerProvider()
}

// GetGlobalTextMapPropagator returns the global text map propagator.
func GetGlobalTextMapPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

// MustNewProvider creates a new provider and panics if initialization fails.
// This is useful for application startup where tracing configuration errors
// should prevent the application from starting.
func MustNewProvider(opts *Options) *Provider {
	provider, err := NewProvider(opts)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize tracing provider: %v", err))
	}
	return provider
}

var _ grpc.ClientConnInterface = (*grpc.ClientConn)(nil)
