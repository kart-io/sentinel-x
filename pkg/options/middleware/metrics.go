package middleware

// MetricsOptions defines metrics options.
type MetricsOptions struct {
	Path      string `json:"path" mapstructure:"path"`
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Subsystem string `json:"subsystem" mapstructure:"subsystem"`
}

// WithMetrics configures and enables metrics endpoint.
func WithMetrics(path, namespace, subsystem string) Option {
	return func(o *Options) {
		o.DisableMetrics = false
		if path != "" {
			o.Metrics.Path = path
		}
		if namespace != "" {
			o.Metrics.Namespace = namespace
		}
		if subsystem != "" {
			o.Metrics.Subsystem = subsystem
		}
	}
}

// WithoutMetrics disables metrics endpoint.
func WithoutMetrics() Option {
	return func(o *Options) { o.DisableMetrics = true }
}
