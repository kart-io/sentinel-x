// Package logger provides logger configuration options for the sentinel-x framework.
package logger

import (
	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/option"
	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

// Options wraps the logger option.LogOption with sentinel-x specific additions.
type Options struct {
	*option.LogOption
	// Enhanced holds enhanced logging configuration for middleware.
	Enhanced *EnhancedLoggerConfig
}

// EnhancedLoggerConfig provides enhanced logging features for HTTP middleware.
type EnhancedLoggerConfig struct {
	// EnableTraceCorrelation enables automatic extraction and logging of OpenTelemetry trace/span IDs.
	EnableTraceCorrelation bool

	// EnableResponseLogging enables logging of response status, size, and latency.
	EnableResponseLogging bool

	// EnableRequestLogging enables logging of request method, path, and headers.
	EnableRequestLogging bool

	// SensitiveHeaders is a list of header names to redact from logs (e.g., "Authorization", "Cookie").
	SensitiveHeaders []string

	// MaxBodyLogSize is the maximum size in bytes of request/response body to log.
	// Set to 0 to disable body logging, -1 for unlimited.
	MaxBodyLogSize int

	// CaptureStackTrace enables stack trace capture for error responses (5xx).
	CaptureStackTrace bool

	// ErrorStackTraceMinStatus is the minimum HTTP status code to capture stack traces (default: 500).
	ErrorStackTraceMinStatus int

	// SkipPaths is a list of paths to skip enhanced logging (e.g., "/health", "/metrics").
	SkipPaths []string

	// LogRequestBody enables logging of request body for debugging.
	LogRequestBody bool

	// LogResponseBody enables logging of response body for debugging.
	LogResponseBody bool

	// SkipHealthChecks disables logging for health check endpoints.
	SkipHealthChecks bool

	// CaptureHeaders is a list of headers to capture in the logs.
	CaptureHeaders []string
}

// DefaultEnhancedLoggerConfig returns default enhanced logger configuration.
func DefaultEnhancedLoggerConfig() *EnhancedLoggerConfig {
	return &EnhancedLoggerConfig{
		EnableTraceCorrelation:   true,
		EnableResponseLogging:    true,
		EnableRequestLogging:     true,
		SensitiveHeaders:         []string{"Authorization", "Cookie", "X-Api-Key", "X-Auth-Token"},
		MaxBodyLogSize:           1024, // 1KB default
		CaptureStackTrace:        false,
		ErrorStackTraceMinStatus: 500,
		SkipPaths:                []string{"/health", "/ready", "/metrics"},
		LogRequestBody:           false,
		LogResponseBody:          false,
	}
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		LogOption: option.DefaultLogOption(),
		Enhanced:  DefaultEnhancedLoggerConfig(),
	}
}

// AddFlags adds flags for logger options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Engine, options.Join(prefixes...)+"log.engine", o.Engine, "Logging engine (zap|slog).")
	fs.StringVar(&o.Level, options.Join(prefixes...)+"log.level", o.Level, "Log level (DEBUG|INFO|WARN|ERROR|FATAL).")
	fs.StringVar(&o.Format, options.Join(prefixes...)+"log.format", o.Format, "Log format (json|console).")
	fs.StringSliceVar(&o.OutputPaths, options.Join(prefixes...)+"log.output-paths", o.OutputPaths, "Output paths for logs.")
	fs.BoolVar(&o.Development, options.Join(prefixes...)+"log.development", o.Development, "Enable development mode.")
	fs.BoolVar(&o.DisableCaller, options.Join(prefixes...)+"log.disable-caller", o.DisableCaller, "Disable caller detection.")
	fs.BoolVar(&o.DisableStacktrace, options.Join(prefixes...)+"log.disable-stacktrace", o.DisableStacktrace, "Disable stacktrace capture.")

	// OTLP options
	fs.StringVar(&o.OTLPEndpoint, options.Join(prefixes...)+"log.otlp-endpoint", o.OTLPEndpoint, "OTLP endpoint URL.")
	if o.OTLP == nil {
		o.OTLP = &option.OTLPOption{}
	}
	fs.StringVar(&o.OTLP.Protocol, options.Join(prefixes...)+"log.otlp.protocol", "grpc", "OTLP protocol (grpc|http).")

	// Rotation options
	if o.Rotation == nil {
		o.Rotation = &option.RotationOption{}
	}
	fs.IntVar(&o.Rotation.MaxSize, options.Join(prefixes...)+"log.rotation.max-size", 100, "Maximum size in MB of the log file before rotation.")
	fs.IntVar(&o.Rotation.MaxAge, options.Join(prefixes...)+"log.rotation.max-age", 15, "Maximum number of days to retain old log files.")
	fs.IntVar(&o.Rotation.MaxBackups, options.Join(prefixes...)+"log.rotation.max-backups", 30, "Maximum number of old log files to retain.")
	fs.BoolVar(&o.Rotation.Compress, options.Join(prefixes...)+"log.rotation.compress", true, "Compress rotated log files using gzip.")

	// Enhanced logging options
	if o.Enhanced == nil {
		o.Enhanced = DefaultEnhancedLoggerConfig()
	}
	fs.BoolVar(&o.Enhanced.EnableTraceCorrelation, options.Join(prefixes...)+"log.enhanced.trace-correlation", o.Enhanced.EnableTraceCorrelation, "Enable OpenTelemetry trace/span ID correlation.")
	fs.BoolVar(&o.Enhanced.EnableResponseLogging, options.Join(prefixes...)+"log.enhanced.response-logging", o.Enhanced.EnableResponseLogging, "Enable response status, size, and latency logging.")
	fs.BoolVar(&o.Enhanced.EnableRequestLogging, options.Join(prefixes...)+"log.enhanced.request-logging", o.Enhanced.EnableRequestLogging, "Enable request method, path, and headers logging.")
	fs.IntVar(&o.Enhanced.MaxBodyLogSize, options.Join(prefixes...)+"log.enhanced.max-body-size", o.Enhanced.MaxBodyLogSize, "Maximum request/response body size to log in bytes.")
	fs.BoolVar(&o.Enhanced.CaptureStackTrace, options.Join(prefixes...)+"log.enhanced.capture-stack", o.Enhanced.CaptureStackTrace, "Capture stack traces for error responses.")
	fs.IntVar(&o.Enhanced.ErrorStackTraceMinStatus, options.Join(prefixes...)+"log.enhanced.stack-min-status", o.Enhanced.ErrorStackTraceMinStatus, "Minimum HTTP status code to capture stack traces.")
	fs.BoolVar(&o.Enhanced.LogRequestBody, options.Join(prefixes...)+"log.enhanced.log-request-body", o.Enhanced.LogRequestBody, "Enable request body logging.")
	fs.BoolVar(&o.Enhanced.LogResponseBody, options.Join(prefixes...)+"log.enhanced.log-response-body", o.Enhanced.LogResponseBody, "Enable response body logging.")
}

// Validate validates the logger options.
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	if err := o.LogOption.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errs
}

// Complete completes the logger options with defaults.
func (o *Options) Complete() error {
	return nil
}

// CreateLogger creates a new logger instance based on the options.
func (o *Options) CreateLogger() (core.Logger, error) {
	return logger.New(o.LogOption)
}

// Init initializes the global logger with the options.
func (o *Options) Init() error {
	log, err := o.CreateLogger()
	if err != nil {
		return err
	}
	logger.SetGlobal(log)
	return nil
}

// NewEnhancedLoggerConfig creates a new EnhancedLoggerConfig with default values.
func NewEnhancedLoggerConfig() *EnhancedLoggerConfig {
	return DefaultEnhancedLoggerConfig()
}
