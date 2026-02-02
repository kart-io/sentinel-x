package options

import (
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/kart-io/sentinel-x/internal/scheduler"
	cliflag "github.com/kart-io/sentinel-x/pkg/app/cliflag"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	etcdopts "github.com/kart-io/sentinel-x/pkg/options/etcd"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
)

// ServerOptions contains the configuration options for the server.
type ServerOptions struct {
	HTTPOptions     *httpopts.Options               `json:"http" mapstructure:"http"`
	GRPCOptions     *grpcopts.Options               `json:"grpc" mapstructure:"grpc"`
	LogOptions      *logopts.Options                `json:"log" mapstructure:"log"`
	MySQLOptions    *mysqlopts.Options              `json:"mysql" mapstructure:"mysql"`
	RedisOptions    *redisopts.Options              `json:"redis" mapstructure:"redis"`
	EtcdOptions     *etcdopts.Options               `json:"etcd" mapstructure:"etcd"`
	RecoveryOptions *middlewareopts.RecoveryOptions `json:"recovery" mapstructure:"recovery"`
	ShutdownTimeout time.Duration                   `json:"shutdown-timeout" mapstructure:"shutdown-timeout"`
}

// NewServerOptions creates a ServerOptions instance with default values.
func NewServerOptions() *ServerOptions {
	return &ServerOptions{
		HTTPOptions:     httpopts.NewOptions(),
		GRPCOptions:     grpcopts.NewOptions(),
		LogOptions:      logopts.NewOptions(),
		MySQLOptions:    mysqlopts.NewOptions(),
		RedisOptions:    redisopts.NewOptions(),
		EtcdOptions:     etcdopts.NewOptions(),
		RecoveryOptions: middlewareopts.NewRecoveryOptions(),
		ShutdownTimeout: 30 * time.Second,
	}
}

// Flags returns flags for a specific server by section name.
func (o *ServerOptions) Flags() (fss cliflag.NamedFlagSets) {
	o.HTTPOptions.AddFlags(fss.FlagSet("http"))
	o.GRPCOptions.AddFlags(fss.FlagSet("grpc"))
	o.LogOptions.AddFlags(fss.FlagSet("log"))
	o.MySQLOptions.AddFlags(fss.FlagSet("mysql"), "mysql.")
	o.RedisOptions.AddFlags(fss.FlagSet("redis"), "redis.")
	o.EtcdOptions.AddFlags(fss.FlagSet("etcd"), "etcd.")

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
	errs = append(errs, o.MySQLOptions.Validate()...)
	errs = append(errs, o.RedisOptions.Validate()...)
	errs = append(errs, o.EtcdOptions.Validate()...)

	return utilerrors.NewAggregate(errs)
}

// Config builds a scheduler.Config based on ServerOptions.
func (o *ServerOptions) Config() (*scheduler.Config, error) {
	return &scheduler.Config{
		HTTPOptions:     o.HTTPOptions,
		GRPCOptions:     o.GRPCOptions,
		LogOptions:      o.LogOptions,
		MySQLOptions:    o.MySQLOptions,
		RedisOptions:    o.RedisOptions,
		EtcdOptions:     o.EtcdOptions,
		RecoveryOptions: o.RecoveryOptions,
		ShutdownTimeout: o.ShutdownTimeout,
	}, nil
}
