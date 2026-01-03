package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareMetrics, func() MiddlewareConfig {
		return NewMetricsOptions()
	})
}

// 确保 MetricsOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*MetricsOptions)(nil)

// MetricsOptions defines metrics options.
type MetricsOptions struct {
	Path      string `json:"path" mapstructure:"path"`
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Subsystem string `json:"subsystem" mapstructure:"subsystem"`
}

// NewMetricsOptions creates default metrics options.
func NewMetricsOptions() *MetricsOptions {
	return &MetricsOptions{
		Path:      "/metrics",
		Namespace: "sentinel",
		Subsystem: "http",
	}
}

// AddFlags adds flags for metrics options to the specified FlagSet.
func (o *MetricsOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Path, options.Join(prefixes...)+"middleware.metrics.path", o.Path, "Metrics endpoint path.")
	fs.StringVar(&o.Namespace, options.Join(prefixes...)+"middleware.metrics.namespace", o.Namespace, "Metrics namespace.")
	fs.StringVar(&o.Subsystem, options.Join(prefixes...)+"middleware.metrics.subsystem", o.Subsystem, "Metrics subsystem.")
}

// Validate validates the metrics options.
func (o *MetricsOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if o.Path == "" {
		errs = append(errs, errors.New("metrics path is required"))
	}
	if o.Namespace == "" {
		errs = append(errs, errors.New("metrics namespace is required"))
	}
	if o.Subsystem == "" {
		errs = append(errs, errors.New("metrics subsystem is required"))
	}
	return errs
}

// Complete completes the metrics options with defaults.
func (o *MetricsOptions) Complete() error {
	return nil
}
