// Package mongodb provides MongoDB options.
package mongodb

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

// redactedPassword is the placeholder used when serializing passwords.
const redactedPassword = "[REDACTED]"

// Options defines configuration options for MongoDB.
type Options struct {
	// Connection
	URI      string `json:"uri" mapstructure:"uri"`           // MongoDB URI (mongodb://...)
	Host     string `json:"host" mapstructure:"host"`         // Host (if not using URI)
	Port     int    `json:"port" mapstructure:"port"`         // Port (default 27017)
	Username string `json:"username" mapstructure:"username"` // Username
	Password string `json:"-" mapstructure:"password"`        // Password (use env var) - Excluded from JSON
	Database string `json:"database" mapstructure:"database"` // Database name

	// Connection Pool
	MaxPoolSize     uint64        `json:"max-pool-size" mapstructure:"max-pool-size"`
	MinPoolSize     uint64        `json:"min-pool-size" mapstructure:"min-pool-size"`
	MaxIdleTime     time.Duration `json:"max-idle-time" mapstructure:"max-idle-time"`
	MaxConnIdleTime time.Duration `json:"max-conn-idle-time" mapstructure:"max-conn-idle-time"`

	// Timeouts
	ConnectTimeout         time.Duration `json:"connect-timeout" mapstructure:"connect-timeout"`
	SocketTimeout          time.Duration `json:"socket-timeout" mapstructure:"socket-timeout"`
	ServerSelectionTimeout time.Duration `json:"server-selection-timeout" mapstructure:"server-selection-timeout"`

	// Other
	ReplicaSet string `json:"replica-set" mapstructure:"replica-set"`
	AuthSource string `json:"auth-source" mapstructure:"auth-source"`
	Direct     bool   `json:"direct" mapstructure:"direct"`
}

// optionsForJSON is used for JSON marshaling with password redacted.
type optionsForJSON struct {
	URI                    string        `json:"uri"`
	Host                   string        `json:"host"`
	Port                   int           `json:"port"`
	Username               string        `json:"username"`
	Password               string        `json:"password"`
	Database               string        `json:"database"`
	MaxPoolSize            uint64        `json:"max-pool-size"`
	MinPoolSize            uint64        `json:"min-pool-size"`
	MaxIdleTime            time.Duration `json:"max-idle-time"`
	MaxConnIdleTime        time.Duration `json:"max-conn-idle-time"`
	ConnectTimeout         time.Duration `json:"connect-timeout"`
	SocketTimeout          time.Duration `json:"socket-timeout"`
	ServerSelectionTimeout time.Duration `json:"server-selection-timeout"`
	ReplicaSet             string        `json:"replica-set"`
	AuthSource             string        `json:"auth-source"`
	Direct                 bool          `json:"direct"`
}

// MarshalJSON implements json.Marshaler with password redaction.
// This prevents accidental password exposure in logs or debug output.
func (o *Options) MarshalJSON() ([]byte, error) {
	password := redactedPassword
	if o.Password == "" {
		password = ""
	}

	return json.Marshal(optionsForJSON{
		URI:                    o.URI,
		Host:                   o.Host,
		Port:                   o.Port,
		Username:               o.Username,
		Password:               password,
		Database:               o.Database,
		MaxPoolSize:            o.MaxPoolSize,
		MinPoolSize:            o.MinPoolSize,
		MaxIdleTime:            o.MaxIdleTime,
		MaxConnIdleTime:        o.MaxConnIdleTime,
		ConnectTimeout:         o.ConnectTimeout,
		SocketTimeout:          o.SocketTimeout,
		ServerSelectionTimeout: o.ServerSelectionTimeout,
		ReplicaSet:             o.ReplicaSet,
		AuthSource:             o.AuthSource,
		Direct:                 o.Direct,
	})
}

// String returns a string representation with password redacted.
// Safe for logging and debugging.
func (o *Options) String() string {
	password := redactedPassword
	if o.Password == "" {
		password = ""
	}
	return fmt.Sprintf("MongoDB{host=%s, port=%d, user=%s, password=%s, database=%s}",
		o.Host, o.Port, o.Username, password, o.Database)
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Host:                   "127.0.0.1",
		Port:                   27017,
		Username:               "",
		Password:               "",
		Database:               "",
		MaxPoolSize:            100,
		MinPoolSize:            10,
		MaxIdleTime:            10 * time.Minute,
		MaxConnIdleTime:        5 * time.Minute,
		ConnectTimeout:         10 * time.Second,
		SocketTimeout:          30 * time.Second,
		ServerSelectionTimeout: 30 * time.Second,
		AuthSource:             "admin",
		Direct:                 false,
	}
}

// Complete fills in any fields not set that are required to have valid data.
// This includes reading sensitive information from environment variables and
// setting default values for connection pool parameters.
func (o *Options) Complete() error {
	// Read password from environment variable if not set
	if o.Password == "" {
		o.Password = os.Getenv("MONGODB_PASSWORD")
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

// AddFlags adds flags for MongoDB options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.URI, options.Join(prefixes...)+"mongodb.uri", o.URI, "MongoDB URI (mongodb://...).")
	fs.StringVar(&o.Host, options.Join(prefixes...)+"mongodb.host", o.Host, "MongoDB service host address.")
	fs.IntVar(&o.Port, options.Join(prefixes...)+"mongodb.port", o.Port, "MongoDB service port.")
	fs.StringVar(&o.Username, options.Join(prefixes...)+"mongodb.username", o.Username, "Username for access to mongodb service.")
	fs.StringVar(&o.Password, options.Join(prefixes...)+"mongodb.password", o.Password, "Password for access to mongodb (DEPRECATED: use MONGODB_PASSWORD env var instead).")
	fs.StringVar(&o.Database, options.Join(prefixes...)+"mongodb.database", o.Database, "Database name for the server to use.")
	fs.Uint64Var(&o.MaxPoolSize, options.Join(prefixes...)+"mongodb.max-pool-size", o.MaxPoolSize, "Maximum number of connections in the pool.")
	fs.Uint64Var(&o.MinPoolSize, options.Join(prefixes...)+"mongodb.min-pool-size", o.MinPoolSize, "Minimum number of connections in the pool.")
	fs.DurationVar(&o.MaxIdleTime, options.Join(prefixes...)+"mongodb.max-idle-time", o.MaxIdleTime, "Maximum idle time for a connection.")
	fs.DurationVar(&o.MaxConnIdleTime, options.Join(prefixes...)+"mongodb.max-conn-idle-time", o.MaxConnIdleTime, "Maximum connection idle time.")
	fs.DurationVar(&o.ConnectTimeout, options.Join(prefixes...)+"mongodb.connect-timeout", o.ConnectTimeout, "Timeout for connection.")
	fs.DurationVar(&o.SocketTimeout, options.Join(prefixes...)+"mongodb.socket-timeout", o.SocketTimeout, "Timeout for socket operations.")
	fs.DurationVar(&o.ServerSelectionTimeout, options.Join(prefixes...)+"mongodb.server-selection-timeout", o.ServerSelectionTimeout, "Timeout for server selection.")
	fs.StringVar(&o.ReplicaSet, options.Join(prefixes...)+"mongodb.replica-set", o.ReplicaSet, "MongoDB replica set name.")
	fs.StringVar(&o.AuthSource, options.Join(prefixes...)+"mongodb.auth-source", o.AuthSource, "MongoDB authentication source.")
	fs.BoolVar(&o.Direct, options.Join(prefixes...)+"mongodb.direct", o.Direct, "MongoDB direct connection.")
}

// BuildURI builds a MongoDB URI from options.
// If URI is already set in options, it returns that.
// Otherwise, it constructs a URI from host, port, username, password, etc.
func BuildURI(opts *Options) string {
	// If URI is already provided, use it
	if opts.URI != "" {
		return opts.URI
	}

	// Build URI from components
	var uri strings.Builder

	uri.WriteString("mongodb://")

	// Add credentials if provided
	if opts.Username != "" {
		uri.WriteString(url.QueryEscape(opts.Username))
		if opts.Password != "" {
			uri.WriteString(":")
			uri.WriteString(url.QueryEscape(opts.Password))
		}
		uri.WriteString("@")
	}

	// Add host and port
	uri.WriteString(opts.Host)
	if opts.Port != 0 {
		uri.WriteString(fmt.Sprintf(":%d", opts.Port))
	}

	// Add database if provided
	if opts.Database != "" {
		uri.WriteString("/")
		uri.WriteString(opts.Database)
	} else {
		uri.WriteString("/")
	}

	// Add query parameters
	params := url.Values{}

	if opts.AuthSource != "" && opts.AuthSource != "admin" {
		params.Add("authSource", opts.AuthSource)
	}

	if opts.ReplicaSet != "" {
		params.Add("replicaSet", opts.ReplicaSet)
	}

	if opts.Direct {
		params.Add("directConnection", "true")
	}

	if len(params) > 0 {
		uri.WriteString("?")
		uri.WriteString(params.Encode())
	}

	return uri.String()
}
