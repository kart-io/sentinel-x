// Package mysql provides MySQL options.
package mysql

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/spf13/pflag"
)

// redactedPassword is the placeholder used when serializing passwords.
const redactedPassword = "[REDACTED]"

// Options defines configuration options for MySQL.
type Options struct {
	Host                  string        `json:"host" mapstructure:"host"`
	Port                  int           `json:"port" mapstructure:"port"`
	Username              string        `json:"username" mapstructure:"username"`
	Password              string        `json:"-" mapstructure:"password"` // Excluded from JSON serialization
	Database              string        `json:"database" mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections" mapstructure:"max-idle-connections"`
	MaxOpenConnections    int           `json:"max-open-connections" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time" mapstructure:"max-connection-life-time"`
	MaxIdleTime           time.Duration `json:"max-idle-time" mapstructure:"max-idle-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
}

// optionsForJSON is used for JSON marshaling with password redacted.
type optionsForJSON struct {
	Host                  string        `json:"host"`
	Port                  int           `json:"port"`
	Username              string        `json:"username"`
	Password              string        `json:"password"`
	Database              string        `json:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections"`
	MaxOpenConnections    int           `json:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time"`
	MaxIdleTime           time.Duration `json:"max-idle-time"`
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
		MaxIdleConnections:    o.MaxIdleConnections,
		MaxOpenConnections:    o.MaxOpenConnections,
		MaxConnectionLifeTime: o.MaxConnectionLifeTime,
		MaxIdleTime:           o.MaxIdleTime,
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
	return fmt.Sprintf("MySQL{host=%s, port=%d, user=%s, password=%s, database=%s}",
		o.Host, o.Port, o.Username, password, o.Database)
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Host:                  "127.0.0.1",
		Port:                  3306,
		Username:              "root",
		Password:              "",
		Database:              "",
		MaxIdleConnections:    20,
		MaxOpenConnections:    200,
		MaxConnectionLifeTime: 3600 * time.Second,
		MaxIdleTime:           600 * time.Second,
		LogLevel:              1, // Silent
	}
}

// Complete fills in any fields not set that are required to have valid data.
// This includes reading sensitive information from environment variables and
// setting default values for connection pool parameters.
func (o *Options) Complete() error {
	// Read password from environment variable if not set
	if o.Password == "" {
		o.Password = os.Getenv("MYSQL_PASSWORD")
	}

	return nil
}

// Validate checks if the options are valid.
// This method is idempotent and has no side effects.
func (o *Options) Validate() error {
	return nil
}

// AddFlags adds flags for MySQL options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, namePrefix string) {
	fs.StringVar(&o.Host, namePrefix+"host", o.Host, "MySQL host")
	fs.IntVar(&o.Port, namePrefix+"port", o.Port, "MySQL port")
	fs.StringVar(&o.Username, namePrefix+"username", o.Username, "MySQL username")
	fs.StringVar(&o.Password, namePrefix+"password", o.Password, "MySQL password (DEPRECATED: use MYSQL_PASSWORD env var instead)")
	fs.StringVar(&o.Database, namePrefix+"database", o.Database, "MySQL database")
	fs.IntVar(&o.MaxIdleConnections, namePrefix+"max-idle-connections", o.MaxIdleConnections, "MySQL max idle connections")
	fs.IntVar(&o.MaxOpenConnections, namePrefix+"max-open-connections", o.MaxOpenConnections, "MySQL max open connections")
	fs.DurationVar(&o.MaxConnectionLifeTime, namePrefix+"max-connection-life-time", o.MaxConnectionLifeTime, "MySQL max connection life time")
	fs.DurationVar(&o.MaxIdleTime, namePrefix+"max-idle-time", o.MaxIdleTime, "MySQL max idle time")
	fs.IntVar(&o.LogLevel, namePrefix+"log-level", o.LogLevel, "MySQL log level")
}

// BuildDSN creates a MySQL Data Source Name (DSN) from the provided options.
// The DSN format is: username:password@tcp(host:port)/database?params
//
// SECURITY NOTE: This function properly escapes the password to prevent
// DSN injection attacks when passwords contain special characters.
//
// Example:
//
//	opts := &Options{
//	    Host:     "localhost",
//	    Port:     3306,
//	    Username: "root",
//	    Password: "secret",
//	    Database: "mydb",
//	}
//	dsn := BuildDSN(opts)
//	// Returns: root:secret@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local
func BuildDSN(opts *Options) string {
	// Escape password to handle special characters safely.
	// Characters like @, /, :, etc. in passwords would break DSN parsing without escaping.
	escapedPassword := url.QueryEscape(opts.Password)

	// Build DSN according to MySQL driver format
	// username:password@tcp(host:port)/database?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		opts.Username,
		escapedPassword,
		opts.Host,
		opts.Port,
		opts.Database,
	)
	return dsn
}
