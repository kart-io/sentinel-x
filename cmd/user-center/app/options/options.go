// Package options contains flags and options for initializing the user-center server.
package options

import (
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	usercenter "github.com/kart-io/sentinel-x/internal/user-center"
	cliflag "github.com/kart-io/sentinel-x/pkg/app/cliflag"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
)

// ServerOptions contains the configuration options for the server.
type ServerOptions struct {
	// HTTPOptions contains HTTP server configuration.
	HTTPOptions *httpopts.Options `json:"http" mapstructure:"http"`

	// GRPCOptions contains gRPC server configuration.
	GRPCOptions *grpcopts.Options `json:"grpc" mapstructure:"grpc"`

	// LogOptions contains logger configuration.
	LogOptions *logopts.Options `json:"log" mapstructure:"log"`

	// JWTOptions contains JWT authentication configuration.
	JWTOptions *jwtopts.Options `json:"jwt" mapstructure:"jwt"`

	// MySQLOptions contains MySQL database configuration.
	MySQLOptions *mysqlopts.Options `json:"mysql" mapstructure:"mysql"`

	// RedisOptions contains Redis configuration.
	RedisOptions *redisopts.Options `json:"redis" mapstructure:"redis"`

	// RecoveryOptions contains recovery middleware configuration.
	RecoveryOptions *middlewareopts.RecoveryOptions `json:"recovery" mapstructure:"recovery"`

	// RequestIDOptions contains request ID middleware configuration.
	RequestIDOptions *middlewareopts.RequestIDOptions `json:"request-id" mapstructure:"request-id"`

	// LoggerOptions contains logger middleware configuration.
	LoggerOptions *middlewareopts.LoggerOptions `json:"logger" mapstructure:"logger"`

	// CORSOptions contains CORS middleware configuration.
	CORSOptions *middlewareopts.CORSOptions `json:"cors" mapstructure:"cors"`

	// TimeoutOptions contains timeout middleware configuration.
	TimeoutOptions *middlewareopts.TimeoutOptions `json:"timeout" mapstructure:"timeout"`

	// HealthOptions contains health check configuration.
	HealthOptions *middlewareopts.HealthOptions `json:"health" mapstructure:"health"`

	// MetricsOptions contains metrics configuration.
	MetricsOptions *middlewareopts.MetricsOptions `json:"metrics" mapstructure:"metrics"`

	// PprofOptions contains pprof configuration.
	PprofOptions *middlewareopts.PprofOptions `json:"pprof" mapstructure:"pprof"`

	// VersionOptions contains version endpoint configuration.
	VersionOptions *middlewareopts.VersionOptions `json:"version" mapstructure:"version"`

	// ShutdownTimeout is the timeout for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" mapstructure:"shutdown-timeout"`
}

// NewServerOptions creates a ServerOptions instance with default values.
func NewServerOptions() *ServerOptions {
	return &ServerOptions{
		HTTPOptions:      httpopts.NewOptions(),
		GRPCOptions:      grpcopts.NewOptions(),
		LogOptions:       logopts.NewOptions(),
		JWTOptions:       jwtopts.NewOptions(),
		MySQLOptions:     mysqlopts.NewOptions(),
		RedisOptions:     redisopts.NewOptions(),
		RecoveryOptions:  middlewareopts.NewRecoveryOptions(),
		RequestIDOptions: middlewareopts.NewRequestIDOptions(),
		LoggerOptions:    middlewareopts.NewLoggerOptions(),
		HealthOptions:    middlewareopts.NewHealthOptions(),
		MetricsOptions:   middlewareopts.NewMetricsOptions(),
		VersionOptions:   middlewareopts.NewVersionOptions(),
		ShutdownTimeout:  30 * time.Second,
		// CORSOptions, TimeoutOptions, PprofOptions 默认禁用（nil）
	}
}

// Flags returns flags for a specific server by section name.
func (o *ServerOptions) Flags() (fss cliflag.NamedFlagSets) {
	o.HTTPOptions.AddFlags(fss.FlagSet("http"))
	o.GRPCOptions.AddFlags(fss.FlagSet("grpc"))
	o.LogOptions.AddFlags(fss.FlagSet("log"))
	o.JWTOptions.AddFlags(fss.FlagSet("jwt"))
	o.MySQLOptions.AddFlags(fss.FlagSet("mysql"), "mysql.")
	o.RedisOptions.AddFlags(fss.FlagSet("redis"), "redis.")

	// misc flags
	fs := fss.FlagSet("misc")
	fs.DurationVar(&o.ShutdownTimeout, "shutdown-timeout", o.ShutdownTimeout, "Graceful shutdown timeout")

	return fss
}

// Complete completes all the required options.
func (o *ServerOptions) Complete() error {
	if err := o.HTTPOptions.Complete(); err != nil {
		return err
	}
	if err := o.GRPCOptions.Complete(); err != nil {
		return err
	}
	return nil
}

// Validate checks whether the options in ServerOptions are valid.
func (o *ServerOptions) Validate() error {
	errs := []error{}

	errs = append(errs, o.HTTPOptions.Validate()...)
	errs = append(errs, o.GRPCOptions.Validate()...)
	errs = append(errs, o.LogOptions.Validate()...)
	errs = append(errs, o.JWTOptions.Validate()...)
	errs = append(errs, o.MySQLOptions.Validate()...)
	errs = append(errs, o.RedisOptions.Validate()...)

	return utilerrors.NewAggregate(errs)
}

// Config builds a usercenter.Config based on ServerOptions.
func (o *ServerOptions) Config() (*usercenter.Config, error) {
	return &usercenter.Config{
		HTTPOptions:      o.HTTPOptions,
		GRPCOptions:      o.GRPCOptions,
		LogOptions:       o.LogOptions,
		JWTOptions:       o.JWTOptions,
		MySQLOptions:     o.MySQLOptions,
		RedisOptions:     o.RedisOptions,
		RecoveryOptions:  o.RecoveryOptions,
		RequestIDOptions: o.RequestIDOptions,
		LoggerOptions:    o.LoggerOptions,
		CORSOptions:      o.CORSOptions,
		TimeoutOptions:   o.TimeoutOptions,
		HealthOptions:    o.HealthOptions,
		MetricsOptions:   o.MetricsOptions,
		PprofOptions:     o.PprofOptions,
		VersionOptions:   o.VersionOptions,
		ShutdownTimeout:  o.ShutdownTimeout,
	}, nil
}
