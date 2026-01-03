// Package milvusopts provides options for Milvus client configuration.
package milvusopts

import (
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

// Options contains Milvus client configuration.
type Options struct {
	// Address is the Milvus server address (host:port).
	Address string `json:"address" mapstructure:"address"`

	// Database is the database name to use.
	Database string `json:"database" mapstructure:"database"`

	// Username for authentication.
	Username string `json:"username" mapstructure:"username"`

	// Password for authentication.
	Password string `json:"password" mapstructure:"password"`

	// Timeout for connection and operations.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// PoolSize is the connection pool size.
	PoolSize int `json:"pool-size" mapstructure:"pool-size"`
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		Address:  "localhost:19530",
		Database: "default",
		Timeout:  30 * time.Second,
		PoolSize: 10,
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Address, options.Join(prefixes...)+"milvus.address", o.Address, "Milvus server address (host:port).")
	fs.StringVar(&o.Database, options.Join(prefixes...)+"milvus.database", o.Database, "Milvus database name.")
	fs.StringVar(&o.Username, options.Join(prefixes...)+"milvus.username", o.Username, "Milvus username for authentication.")
	fs.StringVar(&o.Password, options.Join(prefixes...)+"milvus.password", o.Password, "Milvus password for authentication.")
	fs.DurationVar(&o.Timeout, options.Join(prefixes...)+"milvus.timeout", o.Timeout, "Connection and operation timeout.")
	fs.IntVar(&o.PoolSize, options.Join(prefixes...)+"milvus.pool-size", o.PoolSize, "Connection pool size.")
}

// Validate validates the options.
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	if o.Address == "" {
		errs = append(errs, fmt.Errorf("milvus address is required"))
	}
	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("milvus timeout must be positive"))
	}
	return errs
}
