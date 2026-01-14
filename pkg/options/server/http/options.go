// Package http provides HTTP server configuration options.
package http

import (
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

// Options contains HTTP server configuration.
type Options struct {
	// Addr is the address to listen on.
	Addr string `json:"addr" mapstructure:"addr"`
	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout time.Duration `json:"read-timeout" mapstructure:"read-timeout"`
	// WriteTimeout is the maximum duration before timing out writes of the response.
	WriteTimeout time.Duration `json:"write-timeout" mapstructure:"write-timeout"`
	// IdleTimeout is the maximum amount of time to wait for the next request.
	IdleTimeout time.Duration `json:"idle-timeout" mapstructure:"idle-timeout"`
}

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions creates a new Options with default values.
func NewOptions() *Options {
	return &Options{
		Addr:         ":8100",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// AddFlags adds flags for HTTP options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Addr, options.Join(prefixes...)+"http.addr", o.Addr, "Specify the HTTP server bind address and port.")
	fs.DurationVar(&o.ReadTimeout, options.Join(prefixes...)+"http.read-timeout", o.ReadTimeout, "Timeout for reading the entire request.")
	fs.DurationVar(&o.WriteTimeout, options.Join(prefixes...)+"http.write-timeout", o.WriteTimeout, "Timeout before timing out writes of the response.")
	fs.DurationVar(&o.IdleTimeout, options.Join(prefixes...)+"http.idle-timeout", o.IdleTimeout, "Maximum amount of time to wait for the next request.")
}

// Validate validates the HTTP options.
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error

	if o.Addr == "" {
		errs = append(errs, fmt.Errorf("http.addr cannot be empty"))
	}
	if o.ReadTimeout <= 0 {
		errs = append(errs, fmt.Errorf("http.read-timeout must be positive"))
	}
	if o.WriteTimeout <= 0 {
		errs = append(errs, fmt.Errorf("http.write-timeout must be positive"))
	}

	return errs
}

// Complete completes the HTTP options with defaults.
func (o *Options) Complete() error {
	return nil
}

// WithAddr sets the listen address.
func WithAddr(addr string) Option {
	return func(o *Options) {
		o.Addr = addr
	}
}

// WithReadTimeout sets the read timeout.
func WithReadTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.ReadTimeout = d
	}
}

// WithWriteTimeout sets the write timeout.
func WithWriteTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.WriteTimeout = d
	}
}

// WithIdleTimeout sets the idle timeout.
func WithIdleTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.IdleTimeout = d
	}
}

// ApplyOptions applies the given options to the Options.
func (o *Options) ApplyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}
