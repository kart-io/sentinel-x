// Package grpc provides gRPC server configuration options.
package grpc

import (
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

// Options contains gRPC server configuration.
type Options struct {
	// Addr is the address to listen on.
	Addr string `json:"addr" mapstructure:"addr"`
	// Timeout is the default timeout for requests.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
	// MaxRecvMsgSize is the maximum message size in bytes the server can receive.
	MaxRecvMsgSize int `json:"max-recv-msg-size" mapstructure:"max-recv-msg-size"`
	// MaxSendMsgSize is the maximum message size in bytes the server can send.
	MaxSendMsgSize int `json:"max-send-msg-size" mapstructure:"max-send-msg-size"`
	// EnableReflection enables gRPC server reflection for tools like grpcurl.
	EnableReflection bool `json:"enable-reflection" mapstructure:"enable-reflection"`
}

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions creates a new Options with default values.
func NewOptions() *Options {
	return &Options{
		Addr:             ":9090",
		Timeout:          30 * time.Second,
		MaxRecvMsgSize:   16 * 1024 * 1024, // 16MB
		MaxSendMsgSize:   16 * 1024 * 1024, // 16MB
		EnableReflection: true,
	}
}

// AddFlags adds flags for gRPC options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Addr, options.Join(prefixes...)+"grpc.addr", o.Addr, "Specify the gRPC server bind address and port.")
	fs.DurationVar(&o.Timeout, options.Join(prefixes...)+"grpc.timeout", o.Timeout, "Timeout for server connections.")
	fs.IntVar(&o.MaxRecvMsgSize, options.Join(prefixes...)+"grpc.max-recv-msg-size", o.MaxRecvMsgSize, "Maximum message size in bytes the server can receive.")
	fs.IntVar(&o.MaxSendMsgSize, options.Join(prefixes...)+"grpc.max-send-msg-size", o.MaxSendMsgSize, "Maximum message size in bytes the server can send.")
	fs.BoolVar(&o.EnableReflection, options.Join(prefixes...)+"grpc.enable-reflection", o.EnableReflection, "Enable gRPC server reflection for tools like grpcurl.")
}

// Validate validates the gRPC options.
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error

	if o.Addr == "" {
		errs = append(errs, fmt.Errorf("grpc.addr cannot be empty"))
	}
	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("grpc.timeout must be positive"))
	}
	if o.MaxRecvMsgSize <= 0 {
		errs = append(errs, fmt.Errorf("grpc.max-recv-msg-size must be positive"))
	}
	if o.MaxSendMsgSize <= 0 {
		errs = append(errs, fmt.Errorf("grpc.max-send-msg-size must be positive"))
	}

	return errs
}

// Complete completes the gRPC options with defaults.
func (o *Options) Complete() error {
	return nil
}

// WithAddr sets the listen address.
func WithAddr(addr string) Option {
	return func(o *Options) {
		o.Addr = addr
	}
}

// WithTimeout sets the timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithMaxRecvMsgSize sets the max receive message size.
func WithMaxRecvMsgSize(size int) Option {
	return func(o *Options) {
		o.MaxRecvMsgSize = size
	}
}

// WithMaxSendMsgSize sets the max send message size.
func WithMaxSendMsgSize(size int) Option {
	return func(o *Options) {
		o.MaxSendMsgSize = size
	}
}

// WithReflection enables or disables gRPC reflection.
func WithReflection(enable bool) Option {
	return func(o *Options) {
		o.EnableReflection = enable
	}
}

// ApplyOptions applies the given options to the Options.
func (o *Options) ApplyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}
