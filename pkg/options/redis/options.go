// Package redis provides Redis configuration options.
package redis

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

// redactedPassword is the placeholder used when serializing passwords.
const redactedPassword = "[REDACTED]"

// Options defines configuration options for Redis.
type Options struct {
	Host         string        `json:"host" mapstructure:"host"`
	Port         int           `json:"port" mapstructure:"port"`
	Password     string        `json:"-" mapstructure:"password"` // Excluded from JSON serialization
	Database     int           `json:"database" mapstructure:"database"`
	MaxRetries   int           `json:"max-retries" mapstructure:"max-retries"`
	PoolSize     int           `json:"pool-size" mapstructure:"pool-size"`
	MinIdleConns int           `json:"min-idle-conns" mapstructure:"min-idle-conns"`
	DialTimeout  time.Duration `json:"dial-timeout" mapstructure:"dial-timeout"`
	ReadTimeout  time.Duration `json:"read-timeout" mapstructure:"read-timeout"`
	WriteTimeout time.Duration `json:"write-timeout" mapstructure:"write-timeout"`
	PoolTimeout  time.Duration `json:"pool-timeout" mapstructure:"pool-timeout"`
}

// optionsForJSON is used for JSON marshaling with password redacted.
type optionsForJSON struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Password     string        `json:"password"`
	Database     int           `json:"database"`
	MaxRetries   int           `json:"max-retries"`
	PoolSize     int           `json:"pool-size"`
	MinIdleConns int           `json:"min-idle-conns"`
	DialTimeout  time.Duration `json:"dial-timeout"`
	ReadTimeout  time.Duration `json:"read-timeout"`
	WriteTimeout time.Duration `json:"write-timeout"`
	PoolTimeout  time.Duration `json:"pool-timeout"`
}

// MarshalJSON implements json.Marshaler with password redaction.
// This prevents accidental password exposure in logs or debug output.
func (o *Options) MarshalJSON() ([]byte, error) {
	password := redactedPassword
	if o.Password == "" {
		password = ""
	}

	return json.Marshal(optionsForJSON{
		Host:         o.Host,
		Port:         o.Port,
		Password:     password,
		Database:     o.Database,
		MaxRetries:   o.MaxRetries,
		PoolSize:     o.PoolSize,
		MinIdleConns: o.MinIdleConns,
		DialTimeout:  o.DialTimeout,
		ReadTimeout:  o.ReadTimeout,
		WriteTimeout: o.WriteTimeout,
		PoolTimeout:  o.PoolTimeout,
	})
}

// String returns a string representation with password redacted.
// Safe for logging and debugging.
func (o *Options) String() string {
	password := redactedPassword
	if o.Password == "" {
		password = ""
	}
	return fmt.Sprintf("Redis{host=%s, port=%d, password=%s, database=%d}",
		o.Host, o.Port, password, o.Database)
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Host:         "127.0.0.1",
		Port:         6379,
		Password:     "",
		Database:     0,
		MaxRetries:   3,
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}
}

// Complete fills in any fields not set that are required to have valid data.
// This includes reading sensitive information from environment variables and
// setting default values for connection pool parameters.
func (o *Options) Complete() error {
	// Read password from environment variable if not set
	if o.Password == "" {
		o.Password = os.Getenv("REDIS_PASSWORD")
	}

	return nil
}

// Validate checks if the options are valid.
// This method is idempotent and has no side effects.
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	return nil
}

// AddFlags adds flags for Redis options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Host, options.Join(prefixes...)+"redis.host", o.Host, "Redis service host address.")
	fs.IntVar(&o.Port, options.Join(prefixes...)+"redis.port", o.Port, "Redis service port.")
	fs.StringVar(&o.Password, options.Join(prefixes...)+"redis.password", o.Password, "Password for access to redis (DEPRECATED: use REDIS_PASSWORD env var instead).")
	fs.IntVar(&o.Database, options.Join(prefixes...)+"redis.database", o.Database, "Redis database index.")
	fs.IntVar(&o.MaxRetries, options.Join(prefixes...)+"redis.max-retries", o.MaxRetries, "Maximum number of retries before giving up.")
	fs.IntVar(&o.PoolSize, options.Join(prefixes...)+"redis.pool-size", o.PoolSize, "Maximum number of socket connections.")
	fs.IntVar(&o.MinIdleConns, options.Join(prefixes...)+"redis.min-idle-conns", o.MinIdleConns, "Minimum number of idle connections.")
	fs.DurationVar(&o.DialTimeout, options.Join(prefixes...)+"redis.dial-timeout", o.DialTimeout, "Dial timeout for establishing new connections.")
	fs.DurationVar(&o.ReadTimeout, options.Join(prefixes...)+"redis.read-timeout", o.ReadTimeout, "Timeout for socket reads.")
	fs.DurationVar(&o.WriteTimeout, options.Join(prefixes...)+"redis.write-timeout", o.WriteTimeout, "Timeout for socket writes.")
	fs.DurationVar(&o.PoolTimeout, options.Join(prefixes...)+"redis.pool-timeout", o.PoolTimeout, "Amount of time client waits for connection if all connections are busy.")
}
