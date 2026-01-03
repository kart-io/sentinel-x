package middleware

import (
	"errors"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareTimeout, func() MiddlewareConfig {
		return NewTimeoutOptions()
	})
}

// 确保 TimeoutOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*TimeoutOptions)(nil)

// TimeoutOptions defines timeout middleware options.
type TimeoutOptions struct {
	Timeout   time.Duration `json:"timeout" mapstructure:"timeout"`
	SkipPaths []string      `json:"skip-paths" mapstructure:"skip-paths"`
}

// NewTimeoutOptions creates default timeout middleware options.
func NewTimeoutOptions() *TimeoutOptions {
	return &TimeoutOptions{
		Timeout:   30 * time.Second,
		SkipPaths: []string{},
	}
}

// AddFlags adds flags for timeout options to the specified FlagSet.
func (o *TimeoutOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.DurationVar(&o.Timeout, options.Join(prefixes...)+"middleware.timeout.timeout", o.Timeout, "Request timeout duration.")
	fs.StringSliceVar(&o.SkipPaths, options.Join(prefixes...)+"middleware.timeout.skip-paths", o.SkipPaths, "Skip paths for timeout middleware.")
}

// Validate validates the timeout options.
func (o *TimeoutOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if o.Timeout <= 0 {
		errs = append(errs, errors.New("timeout duration must be greater than 0"))
	}
	return errs
}

// Complete completes the timeout options with defaults.
func (o *TimeoutOptions) Complete() error {
	return nil
}
