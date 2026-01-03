package middleware

import (
	"errors"

	"github.com/spf13/pflag"
)

// MetricsOptions defines metrics options.
type MetricsOptions struct {
	Path      string `json:"path" mapstructure:"path"`
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Subsystem string `json:"subsystem" mapstructure:"subsystem"`
}

// WithMetrics configures and enables metrics endpoint.

func NewMetricsOptions() *MetricsOptions {
	return &MetricsOptions{
		Path:      "/metrics",
		Namespace: "sentinel",
		Subsystem: "http",
	}
}

func (o *MetricsOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Path, "middleware.metrics.path", o.Path, "Metrics endpoint path")
	fs.StringVar(&o.Namespace, "middleware.metrics.namespace", o.Namespace, "Metrics namespace")
	fs.StringVar(&o.Subsystem, "middleware.metrics.subsystem", o.Subsystem, "Metrics subsystem")
}

func (o *MetricsOptions) Validate() error {
	if o.Path == "" {
		return errors.New("metrics path is required")
	}
	if o.Namespace == "" {
		return errors.New("metrics namespace is required")
	}
	if o.Subsystem == "" {
		return errors.New("metrics subsystem is required")
	}
	return nil
}

// Complete completes the metrics options with defaults.
func (o *MetricsOptions) Complete() error {
	return nil
}
