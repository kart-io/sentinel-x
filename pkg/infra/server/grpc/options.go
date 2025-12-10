// Package grpc provides gRPC server configuration options.
package grpc

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

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
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Addr, "grpc.addr", o.Addr, "gRPC server listen address")
	fs.DurationVar(&o.Timeout, "grpc.timeout", o.Timeout, "gRPC server request timeout")
	fs.IntVar(&o.MaxRecvMsgSize, "grpc.max-recv-msg-size", o.MaxRecvMsgSize, "gRPC max receive message size in bytes")
	fs.IntVar(&o.MaxSendMsgSize, "grpc.max-send-msg-size", o.MaxSendMsgSize, "gRPC max send message size in bytes")
	fs.BoolVar(&o.EnableReflection, "grpc.enable-reflection", o.EnableReflection, "Enable gRPC server reflection")
}

// Validate validates the gRPC options.
func (o *Options) Validate() error {
	if o.Addr == "" {
		return fmt.Errorf("grpc.addr cannot be empty")
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("grpc.timeout must be positive")
	}
	if o.MaxRecvMsgSize <= 0 {
		return fmt.Errorf("grpc.max-recv-msg-size must be positive")
	}
	if o.MaxSendMsgSize <= 0 {
		return fmt.Errorf("grpc.max-send-msg-size must be positive")
	}
	return nil
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
