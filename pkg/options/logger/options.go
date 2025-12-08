// Package logger provides logger configuration options for the sentinel-x framework.
package logger

import (
	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/option"
	"github.com/spf13/pflag"
)

// Options wraps the logger option.LogOption with sentinel-x specific additions.
type Options struct {
	*option.LogOption
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		LogOption: option.DefaultLogOption(),
	}
}

// AddFlags adds flags for logger options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Engine, "log.engine", o.Engine, "Logging engine (zap|slog)")
	fs.StringVar(&o.Level, "log.level", o.Level, "Log level (DEBUG|INFO|WARN|ERROR|FATAL)")
	fs.StringVar(&o.Format, "log.format", o.Format, "Log format (json|console)")
	fs.StringSliceVar(&o.OutputPaths, "log.output-paths", o.OutputPaths, "Output paths for logs")
	fs.BoolVar(&o.Development, "log.development", o.Development, "Enable development mode")
	fs.BoolVar(&o.DisableCaller, "log.disable-caller", o.DisableCaller, "Disable caller detection")
	fs.BoolVar(&o.DisableStacktrace, "log.disable-stacktrace", o.DisableStacktrace, "Disable stacktrace capture")

	// OTLP options
	fs.StringVar(&o.OTLPEndpoint, "log.otlp-endpoint", o.OTLPEndpoint, "OTLP endpoint URL")
	if o.OTLP == nil {
		o.OTLP = &option.OTLPOption{}
	}
	fs.StringVar(&o.OTLP.Protocol, "log.otlp.protocol", "grpc", "OTLP protocol (grpc|http)")

	// Rotation options
	if o.Rotation == nil {
		o.Rotation = &option.RotationOption{}
	}
	fs.IntVar(&o.Rotation.MaxSize, "log.rotation.max-size", 100, "Maximum size in MB of the log file before rotation")
	fs.IntVar(&o.Rotation.MaxAge, "log.rotation.max-age", 15, "Maximum number of days to retain old log files")
	fs.IntVar(&o.Rotation.MaxBackups, "log.rotation.max-backups", 30, "Maximum number of old log files to retain")
	fs.BoolVar(&o.Rotation.Compress, "log.rotation.compress", true, "Compress rotated log files using gzip")
}

// Validate validates the logger options.
func (o *Options) Validate() error {
	return o.LogOption.Validate()
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
