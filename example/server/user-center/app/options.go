package app

import (
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	"github.com/spf13/pflag"
)

// Options contains all server options.
type Options struct {
	Server *serveropts.Options `json:"server" mapstructure:"server"`
	Log    *logopts.Options    `json:"log" mapstructure:"log"`
	JWT    *jwtopts.Options    `json:"jwt" mapstructure:"jwt"`
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		Server: serveropts.NewOptions(),
		Log:    logopts.NewOptions(),
		JWT:    jwtopts.NewOptions(),
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.Server.AddFlags(fs)
	o.Log.AddFlags(fs)
	o.JWT.AddFlags(fs)
}

// Validate validates the options.
func (o *Options) Validate() error {
	if err := o.Log.Validate(); err != nil {
		return err
	}
	if err := o.JWT.Validate(); err != nil {
		return err
	}
	return o.Server.Validate()
}

// Complete completes the options.
func (o *Options) Complete() error {
	return o.Server.Complete()
}
