// Package tracing provides OpenTelemetry distributed tracing configuration and initialization.
package tracing

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

// SamplerType defines the type of sampler to use.
type SamplerType string

const (
	// SamplerAlwaysOn samples all traces.
	SamplerAlwaysOn SamplerType = "always_on"
	// SamplerAlwaysOff never samples traces.
	SamplerAlwaysOff SamplerType = "always_off"
	// SamplerRatio samples traces based on a ratio.
	SamplerRatio SamplerType = "ratio"
	// SamplerParentBased uses the parent span's sampling decision.
	SamplerParentBased SamplerType = "parent_based"
)

// ExporterType defines the type of exporter to use.
type ExporterType string

const (
	// ExporterOTLPGRPC exports spans via OTLP over gRPC.
	ExporterOTLPGRPC ExporterType = "otlp_grpc"
	// ExporterOTLPHTTP exports spans via OTLP over HTTP.
	ExporterOTLPHTTP ExporterType = "otlp_http"
	// ExporterStdout exports spans to stdout (for development).
	ExporterStdout ExporterType = "stdout"
	// ExporterNoop does not export spans.
	ExporterNoop ExporterType = "noop"
)

// Options defines configuration for OpenTelemetry tracing.
type Options struct {
	// Enabled enables or disables tracing.
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// ServiceName is the name of the service.
	ServiceName string `json:"service-name" mapstructure:"service-name"`

	// ServiceVersion is the version of the service.
	ServiceVersion string `json:"service-version" mapstructure:"service-version"`

	// ServiceNamespace is the namespace of the service.
	ServiceNamespace string `json:"service-namespace" mapstructure:"service-namespace"`

	// Environment is the deployment environment (e.g., production, staging, development).
	Environment string `json:"environment" mapstructure:"environment"`

	// ExporterType specifies which exporter to use.
	ExporterType ExporterType `json:"exporter-type" mapstructure:"exporter-type"`

	// Endpoint is the OTLP exporter endpoint.
	// For gRPC: "localhost:4317"
	// For HTTP: "http://localhost:4318/v1/traces"
	Endpoint string `json:"endpoint" mapstructure:"endpoint"`

	// Insecure disables TLS for the OTLP connection.
	Insecure bool `json:"insecure" mapstructure:"insecure"`

	// Headers are additional headers to send with OTLP requests.
	Headers map[string]string `json:"headers" mapstructure:"headers"`

	// SamplerType specifies the sampling strategy.
	SamplerType SamplerType `json:"sampler-type" mapstructure:"sampler-type"`

	// SamplerRatio is the sampling ratio (0.0 to 1.0) when using ratio-based sampling.
	SamplerRatio float64 `json:"sampler-ratio" mapstructure:"sampler-ratio"`

	// BatchTimeout is the maximum time to wait before exporting a batch.
	BatchTimeout time.Duration `json:"batch-timeout" mapstructure:"batch-timeout"`

	// BatchMaxSize is the maximum number of spans to export in a batch.
	BatchMaxSize int `json:"batch-max-size" mapstructure:"batch-max-size"`

	// ExportTimeout is the maximum time allowed for exporting spans.
	ExportTimeout time.Duration `json:"export-timeout" mapstructure:"export-timeout"`

	// MaxQueueSize is the maximum queue size for spans awaiting export.
	MaxQueueSize int `json:"max-queue-size" mapstructure:"max-queue-size"`

	// ResourceAttributes are additional resource attributes to attach to all spans.
	ResourceAttributes map[string]string `json:"resource-attributes" mapstructure:"resource-attributes"`
}

// NewOptions creates default tracing options.
func NewOptions() *Options {
	return &Options{
		Enabled:            false, // Disabled by default
		ServiceName:        "sentinel-x",
		ServiceVersion:     "1.0.0",
		ServiceNamespace:   "",
		Environment:        "development",
		ExporterType:       ExporterOTLPGRPC,
		Endpoint:           "localhost:4317",
		Insecure:           true,
		Headers:            make(map[string]string),
		SamplerType:        SamplerParentBased,
		SamplerRatio:       1.0,
		BatchTimeout:       5 * time.Second,
		BatchMaxSize:       512,
		ExportTimeout:      30 * time.Second,
		MaxQueueSize:       2048,
		ResourceAttributes: make(map[string]string),
	}
}

// AddFlags adds flags for tracing options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.Enabled, "tracing.enabled", o.Enabled, "Enable OpenTelemetry tracing")
	fs.StringVar(&o.ServiceName, "tracing.service-name", o.ServiceName, "Service name for tracing")
	fs.StringVar(&o.ServiceVersion, "tracing.service-version", o.ServiceVersion, "Service version for tracing")
	fs.StringVar(&o.ServiceNamespace, "tracing.service-namespace", o.ServiceNamespace, "Service namespace for tracing")
	fs.StringVar(&o.Environment, "tracing.environment", o.Environment, "Deployment environment")
	fs.StringVar((*string)(&o.ExporterType), "tracing.exporter-type", string(o.ExporterType), "Exporter type (otlp_grpc, otlp_http, stdout, noop)")
	fs.StringVar(&o.Endpoint, "tracing.endpoint", o.Endpoint, "OTLP exporter endpoint")
	fs.BoolVar(&o.Insecure, "tracing.insecure", o.Insecure, "Disable TLS for OTLP connection")
	fs.StringVar((*string)(&o.SamplerType), "tracing.sampler-type", string(o.SamplerType), "Sampler type (always_on, always_off, ratio, parent_based)")
	fs.Float64Var(&o.SamplerRatio, "tracing.sampler-ratio", o.SamplerRatio, "Sampling ratio (0.0 to 1.0)")
	fs.DurationVar(&o.BatchTimeout, "tracing.batch-timeout", o.BatchTimeout, "Maximum time to wait before exporting a batch")
	fs.IntVar(&o.BatchMaxSize, "tracing.batch-max-size", o.BatchMaxSize, "Maximum number of spans to export in a batch")
	fs.DurationVar(&o.ExportTimeout, "tracing.export-timeout", o.ExportTimeout, "Maximum time allowed for exporting spans")
	fs.IntVar(&o.MaxQueueSize, "tracing.max-queue-size", o.MaxQueueSize, "Maximum queue size for spans awaiting export")
}

// Validate validates the tracing options.
func (o *Options) Validate() error {
	if !o.Enabled {
		return nil
	}

	if o.ServiceName == "" {
		return fmt.Errorf("tracing: service name is required when tracing is enabled")
	}

	if o.ExporterType != ExporterNoop && o.ExporterType != ExporterStdout {
		if o.Endpoint == "" {
			return fmt.Errorf("tracing: endpoint is required for exporter type %s", o.ExporterType)
		}
	}

	switch o.ExporterType {
	case ExporterOTLPGRPC, ExporterOTLPHTTP, ExporterStdout, ExporterNoop:
		// Valid exporter types
	default:
		return fmt.Errorf("tracing: invalid exporter type: %s", o.ExporterType)
	}

	switch o.SamplerType {
	case SamplerAlwaysOn, SamplerAlwaysOff, SamplerRatio, SamplerParentBased:
		// Valid sampler types
	default:
		return fmt.Errorf("tracing: invalid sampler type: %s", o.SamplerType)
	}

	if o.SamplerType == SamplerRatio {
		if o.SamplerRatio < 0.0 || o.SamplerRatio > 1.0 {
			return fmt.Errorf("tracing: sampler ratio must be between 0.0 and 1.0, got %f", o.SamplerRatio)
		}
	}

	if o.BatchTimeout <= 0 {
		return fmt.Errorf("tracing: batch timeout must be positive")
	}

	if o.BatchMaxSize <= 0 {
		return fmt.Errorf("tracing: batch max size must be positive")
	}

	if o.ExportTimeout <= 0 {
		return fmt.Errorf("tracing: export timeout must be positive")
	}

	if o.MaxQueueSize <= 0 {
		return fmt.Errorf("tracing: max queue size must be positive")
	}

	return nil
}

// Complete fills in any missing values with defaults.
func (o *Options) Complete() error {
	if o.Headers == nil {
		o.Headers = make(map[string]string)
	}
	if o.ResourceAttributes == nil {
		o.ResourceAttributes = make(map[string]string)
	}
	return nil
}
