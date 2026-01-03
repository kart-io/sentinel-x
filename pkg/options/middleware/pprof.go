package middleware

import (
	"errors"

	"github.com/spf13/pflag"
)

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

func (o *PprofOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Prefix, "middleware.pprof.prefix", o.Prefix, "Pprof URL prefix")
	fs.BoolVar(&o.EnableCmdline, "middleware.pprof.enable-cmdline", o.EnableCmdline, "Enable cmdline pprof")
	fs.BoolVar(&o.EnableProfile, "middleware.pprof.enable-profile", o.EnableProfile, "Enable profile pprof")
	fs.BoolVar(&o.EnableSymbol, "middleware.pprof.enable-symbol", o.EnableSymbol, "Enable symbol pprof")
	fs.BoolVar(&o.EnableTrace, "middleware.pprof.enable-trace", o.EnableTrace, "Enable trace pprof")
	fs.IntVar(&o.BlockProfileRate, "middleware.pprof.block-profile-rate", o.BlockProfileRate, "Block profile rate")
	fs.IntVar(&o.MutexProfileFraction, "middleware.pprof.mutex-profile-fraction", o.MutexProfileFraction, "Mutex profile fraction")
}

func (o *PprofOptions) Validate() error {
	if o.Prefix == "" {
		return errors.New("pprof prefix is required")
	}
	return nil
}

func (o *PprofOptions) Complete() error {
	return nil
}
