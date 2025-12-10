package postgres

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
)

// redactedPassword is the placeholder used when serializing passwords.
const redactedPassword = "[REDACTED]"

// Options defines configuration options for PostgreSQL.
type Options struct {
	Host                  string        `json:"host" mapstructure:"host"`
	Port                  int           `json:"port" mapstructure:"port"`
	Username              string        `json:"username" mapstructure:"username"`
	Password              string        `json:"-" mapstructure:"password"` // Excluded from JSON serialization
	Database              string        `json:"database" mapstructure:"database"`
	SSLMode               string        `json:"ssl-mode" mapstructure:"ssl-mode"`
	MaxIdleConnections    int           `json:"max-idle-connections" mapstructure:"max-idle-connections"`
	MaxOpenConnections    int           `json:"max-open-connections" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
}

// optionsForJSON is used for JSON marshaling with password redacted.
type optionsForJSON struct {
	Host                  string        `json:"host"`
	Port                  int           `json:"port"`
	Username              string        `json:"username"`
	Password              string        `json:"password"`
	Database              string        `json:"database"`
	SSLMode               string        `json:"ssl-mode"`
	MaxIdleConnections    int           `json:"max-idle-connections"`
	MaxOpenConnections    int           `json:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time"`
	LogLevel              int           `json:"log-level"`
}

// MarshalJSON implements json.Marshaler with password redaction.
// This prevents accidental password exposure in logs or debug output.
func (o *Options) MarshalJSON() ([]byte, error) {
	password := redactedPassword
	if o.Password == "" {
		password = ""
	}

	return json.Marshal(optionsForJSON{
		Host:                  o.Host,
		Port:                  o.Port,
		Username:              o.Username,
		Password:              password,
		Database:              o.Database,
		SSLMode:               o.SSLMode,
		MaxIdleConnections:    o.MaxIdleConnections,
		MaxOpenConnections:    o.MaxOpenConnections,
		MaxConnectionLifeTime: o.MaxConnectionLifeTime,
		LogLevel:              o.LogLevel,
	})
}

// String returns a string representation with password redacted.
// Safe for logging and debugging.
func (o *Options) String() string {
	password := redactedPassword
	if o.Password == "" {
		password = ""
	}
	return fmt.Sprintf("PostgreSQL{host=%s, port=%d, user=%s, password=%s, database=%s, sslmode=%s}",
		o.Host, o.Port, o.Username, password, o.Database, o.SSLMode)
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Host:                  "127.0.0.1",
		Port:                  5432,
		Username:              "postgres",
		Password:              "",
		Database:              "",
		SSLMode:               "disable",
		MaxIdleConnections:    10,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: 10 * time.Second,
		LogLevel:              1, // Silent
	}
}

// Complete fills in any fields not set that are required to have valid data.
// For PostgreSQL options, this method currently has no completion logic as all
// defaults are set in NewOptions(). This method is provided to satisfy the
// component.ConfigOptions interface.
func (o *Options) Complete() error {
	return nil
}

// Validate checks if the options are valid.
func (o *Options) Validate() error {
	// 如果 CLI 参数为空，从环境变量读取
	if o.Password == "" {
		o.Password = os.Getenv("POSTGRES_PASSWORD")
	}

	// 警告使用 CLI 参数传递密码
	// 如果密码非空但环境变量为空，说明密码是通过 CLI 传递的
	if o.Password != "" && os.Getenv("POSTGRES_PASSWORD") == "" {
		fmt.Fprintf(os.Stderr, "WARNING: Passing PostgreSQL password via CLI is insecure. Use POSTGRES_PASSWORD environment variable instead.\n")
	}

	return nil
}

// AddFlags adds flags for PostgreSQL options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, namePrefix string) {
	fs.StringVar(&o.Host, namePrefix+"host", o.Host, "PostgreSQL host")
	fs.IntVar(&o.Port, namePrefix+"port", o.Port, "PostgreSQL port")
	fs.StringVar(&o.Username, namePrefix+"username", o.Username, "PostgreSQL username")
	fs.StringVar(&o.Password, namePrefix+"password", o.Password, "PostgreSQL password (DEPRECATED: use POSTGRES_PASSWORD env var instead)")
	fs.StringVar(&o.Database, namePrefix+"database", o.Database, "PostgreSQL database")
	fs.StringVar(&o.SSLMode, namePrefix+"ssl-mode", o.SSLMode, "PostgreSQL SSL mode")
	fs.IntVar(&o.MaxIdleConnections, namePrefix+"max-idle-connections", o.MaxIdleConnections, "PostgreSQL max idle connections")
	fs.IntVar(&o.MaxOpenConnections, namePrefix+"max-open-connections", o.MaxOpenConnections, "PostgreSQL max open connections")
	fs.DurationVar(&o.MaxConnectionLifeTime, namePrefix+"max-connection-life-time", o.MaxConnectionLifeTime, "PostgreSQL max connection life time")
	fs.IntVar(&o.LogLevel, namePrefix+"log-level", o.LogLevel, "PostgreSQL log level")
}
