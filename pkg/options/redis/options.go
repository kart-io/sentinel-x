package redis

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
)

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
func (o *Options) Validate() error {
	return nil
}

// AddFlags adds flags for Redis options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, namePrefix string) {
	fs.StringVar(&o.Host, namePrefix+"host", o.Host, "Redis host")
	fs.IntVar(&o.Port, namePrefix+"port", o.Port, "Redis port")
	fs.StringVar(&o.Password, namePrefix+"password", o.Password, "Redis password (DEPRECATED: use REDIS_PASSWORD env var instead)")
	fs.IntVar(&o.Database, namePrefix+"database", o.Database, "Redis database")
	fs.IntVar(&o.MaxRetries, namePrefix+"max-retries", o.MaxRetries, "Redis max retries")
	fs.IntVar(&o.PoolSize, namePrefix+"pool-size", o.PoolSize, "Redis pool size")
	fs.IntVar(&o.MinIdleConns, namePrefix+"min-idle-conns", o.MinIdleConns, "Redis min idle connections")
	fs.DurationVar(&o.DialTimeout, namePrefix+"dial-timeout", o.DialTimeout, "Redis dial timeout")
	fs.DurationVar(&o.ReadTimeout, namePrefix+"read-timeout", o.ReadTimeout, "Redis read timeout")
	fs.DurationVar(&o.WriteTimeout, namePrefix+"write-timeout", o.WriteTimeout, "Redis write timeout")
	fs.DurationVar(&o.PoolTimeout, namePrefix+"pool-timeout", o.PoolTimeout, "Redis pool timeout")
}
