package middleware

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

// TimeoutOptions defines timeout middleware options.
type TimeoutOptions struct {
	Timeout   time.Duration `json:"timeout" mapstructure:"timeout"`
	SkipPaths []string      `json:"skip-paths" mapstructure:"skip-paths"`
}

func NewTimeoutOptions() *TimeoutOptions {
	return &TimeoutOptions{
		Timeout:   30 * time.Second,
		SkipPaths: []string{},
	}
}

func (o *TimeoutOptions) AddFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&o.Timeout, "middleware.timeout.timeout", o.Timeout, "Request timeout duration")
	fs.StringSliceVar(&o.SkipPaths, "middleware.timeout.skip-paths", o.SkipPaths, "Skip paths for timeout middleware")
}

func (o *TimeoutOptions) Validate() error {
	if o.Timeout <= 0 {
		return errors.New("timeout duration must be greater than 0")
	}
	return nil
}

func (o *TimeoutOptions) Complete() error {
	return nil
}
