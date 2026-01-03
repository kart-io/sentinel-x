package middleware

import (
	"errors"

	"github.com/spf13/pflag"
)

// HealthOptions defines health check options.
type HealthOptions struct {
	Path          string       `json:"path" mapstructure:"path"`
	LivenessPath  string       `json:"liveness-path" mapstructure:"liveness-path"`
	ReadinessPath string       `json:"readiness-path" mapstructure:"readiness-path"`
	Checker       func() error `json:"-" mapstructure:"-"`
}

func NewHealthOptions() *HealthOptions {
	return &HealthOptions{
		Path:          "/health",
		LivenessPath:  "/live",
		ReadinessPath: "/ready",
	}
}

func (o *HealthOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Path, "middleware.health.path", o.Path, "Health check endpoint path")
	fs.StringVar(&o.LivenessPath, "middleware.health.liveness-path", o.LivenessPath, "Liveness probe path")
	fs.StringVar(&o.ReadinessPath, "middleware.health.readiness-path", o.ReadinessPath, "Readiness probe path")
}

func (o *HealthOptions) Validate() error {
	if o.Path == "" && o.LivenessPath == "" && o.ReadinessPath == "" {
		return errors.New("health check path is required")
	}
	return nil
}

func (o *HealthOptions) Complete() error {
	return nil
}
