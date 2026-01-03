package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewarePprof, func() MiddlewareConfig {
		return NewPprofOptions()
	})
}

// 确保 PprofOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*PprofOptions)(nil)

// PprofOptions defines pprof options.
type PprofOptions struct {
	Prefix               string `json:"prefix" mapstructure:"prefix"`
	EnableCmdline        bool   `json:"enable-cmdline" mapstructure:"enable-cmdline"`
	EnableProfile        bool   `json:"enable-profile" mapstructure:"enable-profile"`
	EnableSymbol         bool   `json:"enable-symbol" mapstructure:"enable-symbol"`
	EnableTrace          bool   `json:"enable-trace" mapstructure:"enable-trace"`
	BlockProfileRate     int    `json:"block-profile-rate" mapstructure:"block-profile-rate"`
	MutexProfileFraction int    `json:"mutex-profile-fraction" mapstructure:"mutex-profile-fraction"`
}

// NewPprofOptions creates default pprof options.
func NewPprofOptions() *PprofOptions {
	return &PprofOptions{
		Prefix:               "/debug/pprof",
		EnableCmdline:        true,
		EnableProfile:        true,
		EnableSymbol:         true,
		EnableTrace:          true,
		BlockProfileRate:     0,
		MutexProfileFraction: 0,
	}
}

// AddFlags adds flags for pprof options to the specified FlagSet.
func (o *PprofOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Prefix, options.Join(prefixes...)+"middleware.pprof.prefix", o.Prefix, "Pprof URL prefix.")
	fs.BoolVar(&o.EnableCmdline, options.Join(prefixes...)+"middleware.pprof.enable-cmdline", o.EnableCmdline, "Enable cmdline pprof.")
	fs.BoolVar(&o.EnableProfile, options.Join(prefixes...)+"middleware.pprof.enable-profile", o.EnableProfile, "Enable profile pprof.")
	fs.BoolVar(&o.EnableSymbol, options.Join(prefixes...)+"middleware.pprof.enable-symbol", o.EnableSymbol, "Enable symbol pprof.")
	fs.BoolVar(&o.EnableTrace, options.Join(prefixes...)+"middleware.pprof.enable-trace", o.EnableTrace, "Enable trace pprof.")
	fs.IntVar(&o.BlockProfileRate, options.Join(prefixes...)+"middleware.pprof.block-profile-rate", o.BlockProfileRate, "Block profile rate.")
	fs.IntVar(&o.MutexProfileFraction, options.Join(prefixes...)+"middleware.pprof.mutex-profile-fraction", o.MutexProfileFraction, "Mutex profile fraction.")
}

// Validate validates the pprof options.
func (o *PprofOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if o.Prefix == "" {
		errs = append(errs, errors.New("pprof prefix is required"))
	}
	return errs
}

// Complete completes the pprof options with defaults.
func (o *PprofOptions) Complete() error {
	return nil
}
