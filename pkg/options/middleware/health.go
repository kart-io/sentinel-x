package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareHealth, func() MiddlewareConfig {
		return NewHealthOptions()
	})
}

// 确保 HealthOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*HealthOptions)(nil)

// HealthOptions defines health check options.
type HealthOptions struct {
	Path          string       `json:"path" mapstructure:"path"`
	LivenessPath  string       `json:"liveness-path" mapstructure:"liveness-path"`
	ReadinessPath string       `json:"readiness-path" mapstructure:"readiness-path"`
	Checker       func() error `json:"-" mapstructure:"-"`
}

// NewHealthOptions creates default health check options.
func NewHealthOptions() *HealthOptions {
	return &HealthOptions{
		Path:          "/health",
		LivenessPath:  "/live",
		ReadinessPath: "/ready",
	}
}

// AddFlags adds flags for health options to the specified FlagSet.
func (o *HealthOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Path, options.Join(prefixes...)+"middleware.health.path", o.Path, "Health check endpoint path.")
	fs.StringVar(&o.LivenessPath, options.Join(prefixes...)+"middleware.health.liveness-path", o.LivenessPath, "Liveness probe path.")
	fs.StringVar(&o.ReadinessPath, options.Join(prefixes...)+"middleware.health.readiness-path", o.ReadinessPath, "Readiness probe path.")
}

// Validate validates the health options.
func (o *HealthOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if o.Path == "" && o.LivenessPath == "" && o.ReadinessPath == "" {
		errs = append(errs, errors.New("health check path is required"))
	}
	return errs
}

// Complete completes the health options with defaults.
func (o *HealthOptions) Complete() error {
	return nil
}
