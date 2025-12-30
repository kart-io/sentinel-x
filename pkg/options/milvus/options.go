// Package milvusopts provides options for Milvus client configuration.
package milvusopts

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

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
func (o *Options) AddFlags(fs *pflag.FlagSet, prefix string) {
	fs.StringVar(&o.Address, prefix+"address", o.Address, "Milvus server address")
	fs.StringVar(&o.Database, prefix+"database", o.Database, "Milvus database name")
	fs.StringVar(&o.Username, prefix+"username", o.Username, "Milvus username")
	fs.StringVar(&o.Password, prefix+"password", o.Password, "Milvus password")
	fs.DurationVar(&o.Timeout, prefix+"timeout", o.Timeout, "Connection timeout")
	fs.IntVar(&o.PoolSize, prefix+"pool-size", o.PoolSize, "Connection pool size")
}

// Validate validates the options.
func (o *Options) Validate() error {
	if o.Address == "" {
		return fmt.Errorf("milvus address is required")
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("milvus timeout must be positive")
	}
	return nil
}
