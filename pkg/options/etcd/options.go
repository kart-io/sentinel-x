package etcd

import (
	"time"

	"github.com/spf13/pflag"
)

// Options defines configuration options for Etcd.
type Options struct {
	Endpoints      []string      `json:"endpoints" mapstructure:"endpoints"`
	Username       string        `json:"username" mapstructure:"username"`
	Password       string        `json:"password" mapstructure:"password"`
	DialTimeout    time.Duration `json:"dial-timeout" mapstructure:"dial-timeout"`
	RequestTimeout time.Duration `json:"request-timeout" mapstructure:"request-timeout"`
	LeaseTTL       int64         `json:"lease-ttl" mapstructure:"lease-ttl"`
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Endpoints:      []string{"127.0.0.1:2379"},
		Username:       "",
		Password:       "",
		DialTimeout:    5 * time.Second,
		RequestTimeout: 2 * time.Second,
		LeaseTTL:       60,
	}
}

// Validate checks if the options are valid.
func (o *Options) Validate() error {
	return nil
}

// AddFlags adds flags for Etcd options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.Endpoints, "etcd.endpoints", o.Endpoints, "Etcd endpoints")
	fs.StringVar(&o.Username, "etcd.username", o.Username, "Etcd username")
	fs.StringVar(&o.Password, "etcd.password", o.Password, "Etcd password")
	fs.DurationVar(&o.DialTimeout, "etcd.dial-timeout", o.DialTimeout, "Etcd dial timeout")
	fs.DurationVar(&o.RequestTimeout, "etcd.request-timeout", o.RequestTimeout, "Etcd request timeout")
	fs.Int64Var(&o.LeaseTTL, "etcd.lease-ttl", o.LeaseTTL, "Etcd lease TTL")
}
