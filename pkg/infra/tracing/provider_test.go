package tracing

import (
	"context"
	"testing"
	"time"
)

func TestNewOptions(t *testing.T) {
	opts := NewOptions()

	if opts.Enabled {
		t.Error("Expected tracing to be disabled by default")
	}

	if opts.ServiceName != "sentinel-x" {
		t.Errorf("Expected service name to be 'sentinel-x', got %s", opts.ServiceName)
	}

	if opts.ExporterType != ExporterOTLPGRPC {
		t.Errorf("Expected exporter type to be OTLP gRPC, got %s", opts.ExporterType)
	}

	if opts.SamplerType != SamplerParentBased {
		t.Errorf("Expected sampler type to be parent-based, got %s", opts.SamplerType)
	}

	if opts.SamplerRatio != 1.0 {
		t.Errorf("Expected sampler ratio to be 1.0, got %f", opts.SamplerRatio)
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name:    "disabled tracing is valid",
			opts:    &Options{Enabled: false},
			wantErr: false,
		},
		{
			name: "missing service name",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "",
				ExporterType:  ExporterOTLPGRPC,
				Endpoint:      "localhost:4317",
				SamplerType:   SamplerAlwaysOn,
				BatchTimeout:  5 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: true,
		},
		{
			name: "missing endpoint for OTLP exporter",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "test",
				ExporterType:  ExporterOTLPGRPC,
				Endpoint:      "",
				SamplerType:   SamplerAlwaysOn,
				BatchTimeout:  5 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: true,
		},
		{
			name: "invalid exporter type",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "test",
				ExporterType:  "invalid",
				Endpoint:      "localhost:4317",
				SamplerType:   SamplerAlwaysOn,
				BatchTimeout:  5 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: true,
		},
		{
			name: "invalid sampler type",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "test",
				ExporterType:  ExporterOTLPGRPC,
				Endpoint:      "localhost:4317",
				SamplerType:   "invalid",
				BatchTimeout:  5 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: true,
		},
		{
			name: "invalid sampler ratio",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "test",
				ExporterType:  ExporterOTLPGRPC,
				Endpoint:      "localhost:4317",
				SamplerType:   SamplerRatio,
				SamplerRatio:  1.5,
				BatchTimeout:  5 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: true,
		},
		{
			name: "negative batch timeout",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "test",
				ExporterType:  ExporterOTLPGRPC,
				Endpoint:      "localhost:4317",
				SamplerType:   SamplerAlwaysOn,
				BatchTimeout:  -1 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: true,
		},
		{
			name: "valid configuration",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "test",
				ExporterType:  ExporterOTLPGRPC,
				Endpoint:      "localhost:4317",
				SamplerType:   SamplerAlwaysOn,
				BatchTimeout:  5 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: false,
		},
		{
			name: "stdout exporter without endpoint",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "test",
				ExporterType:  ExporterStdout,
				SamplerType:   SamplerAlwaysOn,
				BatchTimeout:  5 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: false,
		},
		{
			name: "noop exporter without endpoint",
			opts: &Options{
				Enabled:       true,
				ServiceName:   "test",
				ExporterType:  ExporterNoop,
				SamplerType:   SamplerAlwaysOn,
				BatchTimeout:  5 * time.Second,
				BatchMaxSize:  512,
				ExportTimeout: 30 * time.Second,
				MaxQueueSize:  2048,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptionsComplete(t *testing.T) {
	opts := &Options{}
	if err := opts.Complete(); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if opts.Headers == nil {
		t.Error("Expected headers to be initialized")
	}

	if opts.ResourceAttributes == nil {
		t.Error("Expected resource attributes to be initialized")
	}
}

func TestNewProvider_Disabled(t *testing.T) {
	opts := &Options{
		Enabled: false,
	}

	provider, err := NewProvider(opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider to be non-nil")
	}

	// Should return a tracer even when disabled
	tracer := provider.Tracer("test")
	if tracer == nil {
		t.Error("Expected tracer to be non-nil")
	}

	// Shutdown should not error
	if err := provider.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestNewProvider_NoopExporter(t *testing.T) {
	opts := &Options{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterNoop,
		SamplerType:    SamplerAlwaysOn,
		BatchTimeout:   5 * time.Second,
		BatchMaxSize:   512,
		ExportTimeout:  30 * time.Second,
		MaxQueueSize:   2048,
	}

	provider, err := NewProvider(opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	defer func() { _ = provider.Shutdown(context.Background()) }()

	if provider == nil {
		t.Fatal("Expected provider to be non-nil")
	}

	// Create and use a tracer
	tracer := provider.Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	span.End()
}

func TestNewProvider_StdoutExporter(t *testing.T) {
	opts := &Options{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplerType:    SamplerAlwaysOn,
		BatchTimeout:   5 * time.Second,
		BatchMaxSize:   512,
		ExportTimeout:  30 * time.Second,
		MaxQueueSize:   2048,
	}

	provider, err := NewProvider(opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	defer func() { _ = provider.Shutdown(context.Background()) }()

	if provider == nil {
		t.Fatal("Expected provider to be non-nil")
	}

	// Create and use a tracer
	tracer := provider.Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	span.End()

	// Flush to ensure span is exported
	if err := provider.ForceFlush(context.Background()); err != nil {
		t.Errorf("ForceFlush() error = %v", err)
	}
}

func TestProvider_Tracer(t *testing.T) {
	opts := &Options{
		Enabled:       true,
		ServiceName:   "test-service",
		ExporterType:  ExporterNoop,
		SamplerType:   SamplerAlwaysOn,
		BatchTimeout:  5 * time.Second,
		BatchMaxSize:  512,
		ExportTimeout: 30 * time.Second,
		MaxQueueSize:  2048,
	}

	provider, err := NewProvider(opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	defer func() { _ = provider.Shutdown(context.Background()) }()

	tracer := provider.Tracer("test-tracer")
	if tracer == nil {
		t.Fatal("Expected tracer to be non-nil")
	}

	// Verify tracer can create spans
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	if span == nil {
		t.Fatal("Expected span to be non-nil")
	}
	span.End()
}

func TestProvider_Shutdown(t *testing.T) {
	opts := &Options{
		Enabled:       true,
		ServiceName:   "test-service",
		ExporterType:  ExporterNoop,
		SamplerType:   SamplerAlwaysOn,
		BatchTimeout:  5 * time.Second,
		BatchMaxSize:  512,
		ExportTimeout: 30 * time.Second,
		MaxQueueSize:  2048,
	}

	provider, err := NewProvider(opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Create a span
	tracer := provider.Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	span.End()

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// Shutdown again should not error
	if err := provider.Shutdown(context.Background()); err != nil {
		t.Errorf("Second Shutdown() error = %v", err)
	}
}

func TestNoopExporter(t *testing.T) {
	exporter := newNoopExporter()

	// Export should not error
	ctx := context.Background()
	if err := exporter.ExportSpans(ctx, nil); err != nil {
		t.Errorf("ExportSpans() error = %v", err)
	}

	// Shutdown should not error
	if err := exporter.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestMustNewProvider_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustNewProvider() panicked: %v", r)
		}
	}()

	opts := &Options{
		Enabled:       true,
		ServiceName:   "test-service",
		ExporterType:  ExporterNoop,
		SamplerType:   SamplerAlwaysOn,
		BatchTimeout:  5 * time.Second,
		BatchMaxSize:  512,
		ExportTimeout: 30 * time.Second,
		MaxQueueSize:  2048,
	}

	provider := MustNewProvider(opts)
	if provider == nil {
		t.Fatal("Expected provider to be non-nil")
	}
	defer provider.Shutdown(context.Background())
}

func TestMustNewProvider_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNewProvider() should have panicked")
		}
	}()

	// Invalid options should cause panic
	opts := &Options{
		Enabled:      true,
		ServiceName:  "", // Missing service name
		ExporterType: ExporterOTLPGRPC,
		SamplerType:  SamplerAlwaysOn,
	}

	MustNewProvider(opts)
}

func TestGetGlobalTracerProvider(t *testing.T) {
	provider := GetGlobalTracerProvider()
	if provider == nil {
		t.Error("Expected global tracer provider to be non-nil")
	}
}

func TestGetGlobalTextMapPropagator(t *testing.T) {
	propagator := GetGlobalTextMapPropagator()
	if propagator == nil {
		t.Error("Expected global text map propagator to be non-nil")
	}
}

// Benchmark tests
func BenchmarkNewProvider(b *testing.B) {
	opts := &Options{
		Enabled:       true,
		ServiceName:   "benchmark-service",
		ExporterType:  ExporterNoop,
		SamplerType:   SamplerAlwaysOn,
		BatchTimeout:  5 * time.Second,
		BatchMaxSize:  512,
		ExportTimeout: 30 * time.Second,
		MaxQueueSize:  2048,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider, err := NewProvider(opts)
		if err != nil {
			b.Fatal(err)
		}
		provider.Shutdown(context.Background())
	}
}

func BenchmarkTracer_StartSpan(b *testing.B) {
	opts := &Options{
		Enabled:       true,
		ServiceName:   "benchmark-service",
		ExporterType:  ExporterNoop,
		SamplerType:   SamplerAlwaysOn,
		BatchTimeout:  5 * time.Second,
		BatchMaxSize:  512,
		ExportTimeout: 30 * time.Second,
		MaxQueueSize:  2048,
	}

	provider, err := NewProvider(opts)
	if err != nil {
		b.Fatal(err)
	}
	defer provider.Shutdown(context.Background())

	tracer := provider.Tracer("benchmark")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "benchmark-span")
		span.End()
	}
}
