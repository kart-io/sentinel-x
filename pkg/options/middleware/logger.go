package middleware

// LoggerOptions defines logger middleware options.
type LoggerOptions struct {
	SkipPaths           []string                                 `json:"skip-paths" mapstructure:"skip-paths"`
	UseStructuredLogger bool                                     `json:"use-structured-logger" mapstructure:"use-structured-logger"`
	Output              func(format string, args ...interface{}) `json:"-" mapstructure:"-"`
}

// WithLogger configures logger middleware.
func WithLogger(skipPaths []string, output func(format string, args ...interface{})) Option {
	return func(o *Options) {
		o.DisableLogger = false
		if skipPaths != nil {
			o.Logger.SkipPaths = skipPaths
		}
		if output != nil {
			o.Logger.Output = output
		}
	}
}

// WithoutLogger disables logger middleware.
func WithoutLogger() Option {
	return func(o *Options) { o.DisableLogger = true }
}
