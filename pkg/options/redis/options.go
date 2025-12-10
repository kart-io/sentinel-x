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
	// 如果 CLI 参数为空，从环境变量读取
	if o.Password == "" {
		o.Password = os.Getenv("REDIS_PASSWORD")
	}

	// 警告使用 CLI 参数传递密码
	// 如果密码非空但环境变量为空，说明密码是通过 CLI 传递的
	if o.Password != "" && os.Getenv("REDIS_PASSWORD") == "" {
		fmt.Fprintf(os.Stderr, "WARNING: Passing Redis password via CLI is insecure. Use REDIS_PASSWORD environment variable instead.\n")
	}

	return nil
}

// AddFlags adds flags for Redis options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Host, "redis.host", o.Host, "Redis host")
	fs.IntVar(&o.Port, "redis.port", o.Port, "Redis port")
	fs.StringVar(&o.Password, "redis.password", o.Password, "Redis password (DEPRECATED: use REDIS_PASSWORD env var instead)")
	fs.IntVar(&o.Database, "redis.database", o.Database, "Redis database")
	fs.IntVar(&o.MaxRetries, "redis.max-retries", o.MaxRetries, "Redis max retries")
	fs.IntVar(&o.PoolSize, "redis.pool-size", o.PoolSize, "Redis pool size")
	fs.IntVar(&o.MinIdleConns, "redis.min-idle-conns", o.MinIdleConns, "Redis min idle connections")
	fs.DurationVar(&o.DialTimeout, "redis.dial-timeout", o.DialTimeout, "Redis dial timeout")
	fs.DurationVar(&o.ReadTimeout, "redis.read-timeout", o.ReadTimeout, "Redis read timeout")
	fs.DurationVar(&o.WriteTimeout, "redis.write-timeout", o.WriteTimeout, "Redis write timeout")
	fs.DurationVar(&o.PoolTimeout, "redis.pool-timeout", o.PoolTimeout, "Redis pool timeout")
}
