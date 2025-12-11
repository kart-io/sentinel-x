package middleware

import "time"

// TimeoutOptions defines timeout middleware options.
type TimeoutOptions struct {
	Timeout   time.Duration `json:"timeout" mapstructure:"timeout"`
	SkipPaths []string      `json:"skip-paths" mapstructure:"skip-paths"`
}

// WithTimeout configures and enables timeout middleware.
func WithTimeout(timeout time.Duration, skipPaths []string) Option {
	return func(o *Options) {
		o.DisableTimeout = false
		o.Timeout.Timeout = timeout
		if skipPaths != nil {
			o.Timeout.SkipPaths = skipPaths
		}
	}
}

// WithoutTimeout disables timeout middleware.
func WithoutTimeout() Option {
	return func(o *Options) { o.DisableTimeout = true }
}
