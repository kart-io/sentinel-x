// Package http provides HTTP server configuration options.
package http

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/kart-io/sentinel-x/pkg/middleware"
)

// AdapterType represents the HTTP framework adapter type.
type AdapterType string

const (
	// AdapterGin uses Gin as the HTTP framework.
	AdapterGin AdapterType = "gin"
	// AdapterEcho uses Echo as the HTTP framework.
	AdapterEcho AdapterType = "echo"
)

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
	// Adapter specifies which HTTP framework adapter to use.
	Adapter AdapterType `json:"adapter" mapstructure:"adapter"`
	// Middleware contains all middleware configuration.
	Middleware *middleware.Options `json:"middleware" mapstructure:"middleware"`
}

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions creates a new Options with default values.
func NewOptions() *Options {
	return &Options{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		Adapter:      AdapterGin,
		Middleware:   middleware.NewOptions(),
	}
}

// AddFlags adds flags for HTTP options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Addr, "http.addr", o.Addr, "HTTP server listen address")
	fs.DurationVar(&o.ReadTimeout, "http.read-timeout", o.ReadTimeout, "HTTP server read timeout")
	fs.DurationVar(&o.WriteTimeout, "http.write-timeout", o.WriteTimeout, "HTTP server write timeout")
	fs.DurationVar(&o.IdleTimeout, "http.idle-timeout", o.IdleTimeout, "HTTP server idle timeout")
	fs.StringVar((*string)(&o.Adapter), "http.adapter", string(o.Adapter), "HTTP framework adapter (gin, echo)")
}

// Validate validates the HTTP options.
func (o *Options) Validate() error {
	if o.Addr == "" {
		return fmt.Errorf("http.addr cannot be empty")
	}
	if o.ReadTimeout <= 0 {
		return fmt.Errorf("http.read-timeout must be positive")
	}
	if o.WriteTimeout <= 0 {
		return fmt.Errorf("http.write-timeout must be positive")
	}
	if o.Adapter != AdapterGin && o.Adapter != AdapterEcho {
		return fmt.Errorf("http.adapter must be 'gin' or 'echo'")
	}
	return nil
}

// Complete completes the HTTP options with defaults.
func (o *Options) Complete() error {
	if o.Middleware == nil {
		o.Middleware = middleware.NewOptions()
	}
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

// WithAdapter sets the HTTP framework adapter.
func WithAdapter(adapter AdapterType) Option {
	return func(o *Options) {
		o.Adapter = adapter
	}
}

// WithMiddleware configures middleware options.
func WithMiddleware(opts ...middleware.Option) Option {
	return func(o *Options) {
		for _, opt := range opts {
			opt(o.Middleware)
		}
	}
}

// ApplyOptions applies the given options to the Options.
func (o *Options) ApplyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}
