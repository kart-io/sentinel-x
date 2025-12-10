// Package server provides server manager configuration options.
package server

import (
	"fmt"
	"time"

	grpcopts "github.com/kart-io/sentinel-x/pkg/infra/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/infra/server/http"
	"github.com/spf13/pflag"
)

// Mode represents the server startup mode.
type Mode int

const (
	// ModeHTTPOnly starts only HTTP server.
	ModeHTTPOnly Mode = 1 << iota
	// ModeGRPCOnly starts only gRPC server.
	ModeGRPCOnly
	// ModeBoth starts both HTTP and gRPC servers.
	ModeBoth = ModeHTTPOnly | ModeGRPCOnly
)

// String returns the string representation of the mode.
func (m Mode) String() string {
	switch m {
	case ModeHTTPOnly:
		return "http"
	case ModeGRPCOnly:
		return "grpc"
	case ModeBoth:
		return "both"
	default:
		return "unknown"
	}
}

// ParseMode parses a string to Mode.
func ParseMode(s string) (Mode, error) {
	switch s {
	case "http":
		return ModeHTTPOnly, nil
	case "grpc":
		return ModeGRPCOnly, nil
	case "both":
		return ModeBoth, nil
	default:
		return 0, fmt.Errorf("invalid mode: %s", s)
	}
}

// Options contains all configuration for the server manager.
type Options struct {
	// Mode determines which servers to start.
	Mode Mode `json:"mode" mapstructure:"-"`

	// ModeString is the string representation of mode for flags.
	ModeString string `json:"mode-string" mapstructure:"mode"`

	// HTTP contains HTTP server options.
	HTTP *httpopts.Options `json:"http" mapstructure:"http"`

	// GRPC contains gRPC server options.
	GRPC *grpcopts.Options `json:"grpc" mapstructure:"grpc"`

	// ShutdownTimeout is the timeout for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" mapstructure:"shutdown-timeout"`
}

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions creates a new Options with default values.
func NewOptions() *Options {
	return &Options{
		Mode:            ModeBoth,
		ModeString:      "both",
		HTTP:            httpopts.NewOptions(),
		GRPC:            grpcopts.NewOptions(),
		ShutdownTimeout: 30 * time.Second,
	}
}

// AddFlags adds flags for server options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ModeString, "server.mode", o.ModeString, "Server mode: http, grpc, or both")
	fs.DurationVar(&o.ShutdownTimeout, "server.shutdown-timeout", o.ShutdownTimeout, "Graceful shutdown timeout")

	// Add HTTP and gRPC flags
	o.HTTP.AddFlags(fs)
	o.GRPC.AddFlags(fs)
}

// Validate validates all server options.
func (o *Options) Validate() error {
	if o.ShutdownTimeout <= 0 {
		return fmt.Errorf("server.shutdown-timeout must be positive")
	}

	// Validate HTTP options if HTTP is enabled
	if o.EnableHTTP() {
		if err := o.HTTP.Validate(); err != nil {
			return err
		}
	}

	// Validate gRPC options if gRPC is enabled
	if o.EnableGRPC() {
		if err := o.GRPC.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Complete completes all server options with defaults.
func (o *Options) Complete() error {
	// Parse mode string
	if o.ModeString != "" {
		mode, err := ParseMode(o.ModeString)
		if err != nil {
			return err
		}
		o.Mode = mode
	}

	// Complete HTTP options
	if o.HTTP != nil {
		if err := o.HTTP.Complete(); err != nil {
			return err
		}
	}

	// Complete gRPC options
	if o.GRPC != nil {
		if err := o.GRPC.Complete(); err != nil {
			return err
		}
	}

	return nil
}

// WithMode sets the server mode.
func WithMode(mode Mode) Option {
	return func(o *Options) {
		o.Mode = mode
		o.ModeString = mode.String()
	}
}

// WithHTTPOptions sets HTTP server options.
func WithHTTPOptions(opts *httpopts.Options) Option {
	return func(o *Options) {
		o.HTTP = opts
	}
}

// WithGRPCOptions sets gRPC server options.
func WithGRPCOptions(opts *grpcopts.Options) Option {
	return func(o *Options) {
		o.GRPC = opts
	}
}

// WithShutdownTimeout sets the graceful shutdown timeout.
func WithShutdownTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.ShutdownTimeout = d
	}
}

// EnableHTTP returns true if HTTP server should be started.
func (o *Options) EnableHTTP() bool {
	return o.Mode&ModeHTTPOnly != 0
}

// EnableGRPC returns true if gRPC server should be started.
func (o *Options) EnableGRPC() bool {
	return o.Mode&ModeGRPCOnly != 0
}

// ApplyOptions applies the given options to the Options.
func (o *Options) ApplyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}
