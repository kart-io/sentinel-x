package middleware

// HealthOptions defines health check options.
type HealthOptions struct {
	Path          string       `json:"path" mapstructure:"path"`
	LivenessPath  string       `json:"liveness-path" mapstructure:"liveness-path"`
	ReadinessPath string       `json:"readiness-path" mapstructure:"readiness-path"`
	Checker       func() error `json:"-" mapstructure:"-"`
}

// WithHealth configures and enables health check endpoints.
func WithHealth(path, livenessPath, readinessPath string, checker func() error) Option {
	return func(o *Options) {
		o.DisableHealth = false
		if path != "" {
			o.Health.Path = path
		}
		if livenessPath != "" {
			o.Health.LivenessPath = livenessPath
		}
		if readinessPath != "" {
			o.Health.ReadinessPath = readinessPath
		}
		if checker != nil {
			o.Health.Checker = checker
		}
	}
}

// WithoutHealth disables health check endpoints.
func WithoutHealth() Option {
	return func(o *Options) { o.DisableHealth = true }
}
