package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareLogger, func() MiddlewareConfig {
		return NewLoggerOptions()
	})
}

// 确保 LoggerOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*LoggerOptions)(nil)

// LoggerOptions defines logger middleware options.
// 此结构体必须保持可 JSON 序列化，运行时依赖（如 Output）应通过函数参数注入。
type LoggerOptions struct {
	SkipPaths           []string `json:"skip-paths" mapstructure:"skip-paths"`
	UseStructuredLogger bool     `json:"use-structured-logger" mapstructure:"use-structured-logger"`
}

// NewLoggerOptions creates default logger middleware options.
func NewLoggerOptions() *LoggerOptions {
	return &LoggerOptions{
		SkipPaths:           []string{"/health", "/ready", "/live", "/metrics"},
		UseStructuredLogger: true,
	}
}

// AddFlags adds flags for logger options to the specified FlagSet.
func (o *LoggerOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringSliceVar(&o.SkipPaths, options.Join(prefixes...)+"middleware.logger.skip-paths", o.SkipPaths, "Paths to skip logging.")
	fs.BoolVar(&o.UseStructuredLogger, options.Join(prefixes...)+"middleware.logger.use-structured-logger", o.UseStructuredLogger, "Use structured logger.")
}

// Validate validates the logger options.
func (o *LoggerOptions) Validate() []error {
	if o == nil {
		return nil
	}
	return nil
}

// Complete completes the logger options with defaults.
func (o *LoggerOptions) Complete() error {
	return nil
}
