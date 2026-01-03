// Package mysql provides MySQL options.
package mysql

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

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
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	return nil
}

// AddFlags adds flags for MySQL options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Host, options.Join(prefixes...)+"mysql.host", o.Host, "MySQL service host address.")
	fs.IntVar(&o.Port, options.Join(prefixes...)+"mysql.port", o.Port, "MySQL service port.")
	fs.StringVar(&o.Username, options.Join(prefixes...)+"mysql.username", o.Username, "Username for access to mysql service.")
	fs.StringVar(&o.Password, options.Join(prefixes...)+"mysql.password", o.Password, "Password for access to mysql, should be used pair with password (DEPRECATED: use MYSQL_PASSWORD env var instead).")
	fs.StringVar(&o.Database, options.Join(prefixes...)+"mysql.database", o.Database, "Database name for the server to use.")
	fs.IntVar(&o.MaxIdleConnections, options.Join(prefixes...)+"mysql.max-idle-connections", o.MaxIdleConnections, "Maximum idle connections allowed to connect to mysql.")
	fs.IntVar(&o.MaxOpenConnections, options.Join(prefixes...)+"mysql.max-open-connections", o.MaxOpenConnections, "Maximum open connections allowed to connect to mysql.")
	fs.DurationVar(&o.MaxConnectionLifeTime, options.Join(prefixes...)+"mysql.max-connection-life-time", o.MaxConnectionLifeTime, "Maximum connection life time allowed to connect to mysql.")
	fs.DurationVar(&o.MaxIdleTime, options.Join(prefixes...)+"mysql.max-idle-time", o.MaxIdleTime, "Maximum idle time allowed for mysql connection.")
	fs.IntVar(&o.LogLevel, options.Join(prefixes...)+"mysql.log-level", o.LogLevel, "Specify gorm log level.")
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
