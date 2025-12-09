package redis

import (
	"time"

	"github.com/spf13/pflag"
)

// Options defines configuration options for Redis.
type Options struct {
	Host         string        `json:"host" mapstructure:"host"`
	Port         int           `json:"port" mapstructure:"port"`
	Password     string        `json:"password" mapstructure:"password"`
	Database     int           `json:"database" mapstructure:"database"`
	MaxRetries   int           `json:"max-retries" mapstructure:"max-retries"`
	PoolSize     int           `json:"pool-size" mapstructure:"pool-size"`
	MinIdleConns int           `json:"min-idle-conns" mapstructure:"min-idle-conns"`
	DialTimeout  time.Duration `json:"dial-timeout" mapstructure:"dial-timeout"`
	ReadTimeout  time.Duration `json:"read-timeout" mapstructure:"read-timeout"`
	WriteTimeout time.Duration `json:"write-timeout" mapstructure:"write-timeout"`
	PoolTimeout  time.Duration `json:"pool-timeout" mapstructure:"pool-timeout"`
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Host:         "127.0.0.1",
		Port:         6379,
		Password:     "",
		Database:     0,
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}
}

// Validate checks if the options are valid.
func (o *Options) Validate() error {
	return nil
}

// AddFlags adds flags for Redis options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Host, "redis.host", o.Host, "Redis host")
	fs.IntVar(&o.Port, "redis.port", o.Port, "Redis port")
	fs.StringVar(&o.Password, "redis.password", o.Password, "Redis password")
	fs.IntVar(&o.Database, "redis.database", o.Database, "Redis database")
	fs.IntVar(&o.MaxRetries, "redis.max-retries", o.MaxRetries, "Redis max retries")
	fs.IntVar(&o.PoolSize, "redis.pool-size", o.PoolSize, "Redis pool size")
	fs.IntVar(&o.MinIdleConns, "redis.min-idle-conns", o.MinIdleConns, "Redis min idle connections")
	fs.DurationVar(&o.DialTimeout, "redis.dial-timeout", o.DialTimeout, "Redis dial timeout")
	fs.DurationVar(&o.ReadTimeout, "redis.read-timeout", o.ReadTimeout, "Redis read timeout")
	fs.DurationVar(&o.WriteTimeout, "redis.write-timeout", o.WriteTimeout, "Redis write timeout")
	fs.DurationVar(&o.PoolTimeout, "redis.pool-timeout", o.PoolTimeout, "Redis pool timeout")
}
