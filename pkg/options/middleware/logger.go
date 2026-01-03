package middleware

import (
	"github.com/spf13/pflag"
)

// LoggerOptions defines logger middleware options.
type LoggerOptions struct {
	SkipPaths           []string                                 `json:"skip-paths" mapstructure:"skip-paths"`
	UseStructuredLogger bool                                     `json:"use-structured-logger" mapstructure:"use-structured-logger"`
	Output              func(format string, args ...interface{}) `json:"-" mapstructure:"-"`
}

func NewLoggerOptions() *LoggerOptions {
	return &LoggerOptions{
		SkipPaths:           []string{"/health", "/ready", "/live", "/metrics"},
		UseStructuredLogger: true,
		Output:              nil,
	}
}

func (o *LoggerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.SkipPaths, "middleware.logger.skip-paths", o.SkipPaths, "Paths to skip logging")
	fs.BoolVar(&o.UseStructuredLogger, "middleware.logger.use-structured-logger", o.UseStructuredLogger, "Use structured logger")
}

func (o *LoggerOptions) Validate() error {
	return nil
}

func (o *LoggerOptions) Complete() error {
	return nil
}
