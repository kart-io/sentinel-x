// Package app provides the API server application.
package app

import (
	mysqlopts "github.com/kart-io/sentinel-x/pkg/component/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/component/redis"
	logopts "github.com/kart-io/sentinel-x/pkg/infra/logger"
	serveropts "github.com/kart-io/sentinel-x/pkg/infra/server"
	jwtopts "github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/spf13/pflag"
)

// Options contains all API server options.
type Options struct {
	// Server contains server configuration (HTTP/gRPC).
	Server *serveropts.Options `json:"server" mapstructure:"server"`

	// Log contains logger configuration.
	Log *logopts.Options `json:"log" mapstructure:"log"`

	// JWT contains JWT authentication configuration.
	JWT *jwtopts.Options `json:"jwt" mapstructure:"jwt"`

	// MySQL contains MySQL database configuration.
	MySQL *mysqlopts.Options `json:"mysql" mapstructure:"mysql"`

	// Redis contains Redis configuration.
	Redis *redisopts.Options `json:"redis" mapstructure:"redis"`
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		Server: serveropts.NewOptions(),
		Log:    logopts.NewOptions(),
		JWT:    jwtopts.NewOptions(),
		MySQL:  mysqlopts.NewOptions(),
		Redis:  redisopts.NewOptions(),
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.Server.AddFlags(fs)
	o.Log.AddFlags(fs)
	o.JWT.AddFlags(fs)
	o.MySQL.AddFlags(fs, "mysql.")
	o.Redis.AddFlags(fs, "redis.")
}

// Validate validates the options.
func (o *Options) Validate() error {
	if err := o.Log.Validate(); err != nil {
		return err
	}
	if err := o.Server.Validate(); err != nil {
		return err
	}
	if err := o.JWT.Validate(); err != nil {
		return err
	}
	if err := o.MySQL.Validate(); err != nil {
		return err
	}
	if err := o.Redis.Validate(); err != nil {
		return err
	}
	return nil
}

// Complete completes the options.
func (o *Options) Complete() error {
	if err := o.Server.Complete(); err != nil {
		return err
	}
	return nil
}
