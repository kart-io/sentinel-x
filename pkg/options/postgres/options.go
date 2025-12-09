package postgres

import (
	"time"

	"github.com/spf13/pflag"
)

// Options defines configuration options for PostgreSQL.
type Options struct {
	Host                  string        `json:"host" mapstructure:"host"`
	Port                  int           `json:"port" mapstructure:"port"`
	Username              string        `json:"username" mapstructure:"username"`
	Password              string        `json:"password" mapstructure:"password"`
	Database              string        `json:"database" mapstructure:"database"`
	SSLMode               string        `json:"ssl-mode" mapstructure:"ssl-mode"`
	MaxIdleConnections    int           `json:"max-idle-connections" mapstructure:"max-idle-connections"`
	MaxOpenConnections    int           `json:"max-open-connections" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
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

// Validate checks if the options are valid.
func (o *Options) Validate() error {
	return nil
}

// AddFlags adds flags for PostgreSQL options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Host, "postgres.host", o.Host, "PostgreSQL host")
	fs.IntVar(&o.Port, "postgres.port", o.Port, "PostgreSQL port")
	fs.StringVar(&o.Username, "postgres.username", o.Username, "PostgreSQL username")
	fs.StringVar(&o.Password, "postgres.password", o.Password, "PostgreSQL password")
	fs.StringVar(&o.Database, "postgres.database", o.Database, "PostgreSQL database")
	fs.StringVar(&o.SSLMode, "postgres.ssl-mode", o.SSLMode, "PostgreSQL SSL mode")
	fs.IntVar(&o.MaxIdleConnections, "postgres.max-idle-connections", o.MaxIdleConnections, "PostgreSQL max idle connections")
	fs.IntVar(&o.MaxOpenConnections, "postgres.max-open-connections", o.MaxOpenConnections, "PostgreSQL max open connections")
	fs.DurationVar(&o.MaxConnectionLifeTime, "postgres.max-connection-life-time", o.MaxConnectionLifeTime, "PostgreSQL max connection life time")
	fs.IntVar(&o.LogLevel, "postgres.log-level", o.LogLevel, "PostgreSQL log level")
}
