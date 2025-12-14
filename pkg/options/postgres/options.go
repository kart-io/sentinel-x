// Package postgres provides PostgreSQL configuration options.
package postgres

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
// This includes reading sensitive information from environment variables and
// setting default values for connection pool parameters.
func (o *Options) Complete() error {
	// Read password from environment variable if not set
	if o.Password == "" {
		o.Password = os.Getenv("POSTGRES_PASSWORD")
	}

	return nil
}

// Validate checks if the options are valid.
// This method is idempotent and has no side effects.
func (o *Options) Validate() error {
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

// BuildDSN creates a PostgreSQL DSN (Data Source Name) from the provided options.
//
// SECURITY NOTE: This function properly escapes the password to prevent
// DSN injection attacks when passwords contain special characters.
//
// The DSN format is:
// host=<host> port=<port> user=<username> password=<password> dbname=<database> sslmode=<sslmode>
//
// Example:
//
//	host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=disable
func BuildDSN(opts *Options) string {
	if opts == nil {
		return ""
	}

	// Escape password for PostgreSQL DSN format.
	// PostgreSQL uses space-separated key=value pairs, so we need to:
	// 1. Escape single quotes by doubling them
	// 2. Wrap the password in single quotes if it contains special characters
	escapedPassword := escapePostgresValue(opts.Password)

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		opts.Host,
		opts.Port,
		opts.Username,
		escapedPassword,
		opts.Database,
		opts.SSLMode,
	)
}

// escapePostgresValue escapes a value for PostgreSQL DSN format.
// If the value contains spaces or special characters, it wraps the value in single quotes
// and escapes any existing single quotes by doubling them.
func escapePostgresValue(value string) string {
	// If empty, return empty quotes
	if value == "" {
		return "''"
	}

	// Check if value needs quoting
	needsQuoting := strings.ContainsAny(value, " '\\")

	if needsQuoting {
		// Escape single quotes by doubling them
		escaped := strings.ReplaceAll(value, "'", "''")
		// Escape backslashes
		escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
		return "'" + escaped + "'"
	}

	return value
}
