package mongodb

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
)

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
// For MongoDB options, this method currently has no completion logic as all
// defaults are set in NewOptions(). This method is provided to satisfy the
// component.ConfigOptions interface.
func (o *Options) Complete() error {
	return nil
}

// Validate checks if the options are valid.
func (o *Options) Validate() error {
	// 如果 CLI 参数为空，从环境变量读取
	if o.Password == "" {
		o.Password = os.Getenv("MONGODB_PASSWORD")
	}

	// 警告使用 CLI 参数传递密码
	// 如果密码非空但环境变量为空，说明密码是通过 CLI 传递的
	if o.Password != "" && os.Getenv("MONGODB_PASSWORD") == "" {
		fmt.Fprintf(os.Stderr, "WARNING: Passing MongoDB password via CLI is insecure. Use MONGODB_PASSWORD environment variable instead.\n")
	}

	return nil
}

// AddFlags adds flags for MongoDB options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, namePrefix string) {
	fs.StringVar(&o.URI, namePrefix+"uri", o.URI, "MongoDB URI (mongodb://...)")
	fs.StringVar(&o.Host, namePrefix+"host", o.Host, "MongoDB host")
	fs.IntVar(&o.Port, namePrefix+"port", o.Port, "MongoDB port")
	fs.StringVar(&o.Username, namePrefix+"username", o.Username, "MongoDB username")
	fs.StringVar(&o.Password, namePrefix+"password", o.Password, "MongoDB password (DEPRECATED: use MONGODB_PASSWORD env var instead)")
	fs.StringVar(&o.Database, namePrefix+"database", o.Database, "MongoDB database")
	fs.Uint64Var(&o.MaxPoolSize, namePrefix+"max-pool-size", o.MaxPoolSize, "MongoDB max pool size")
	fs.Uint64Var(&o.MinPoolSize, namePrefix+"min-pool-size", o.MinPoolSize, "MongoDB min pool size")
	fs.DurationVar(&o.MaxIdleTime, namePrefix+"max-idle-time", o.MaxIdleTime, "MongoDB max idle time")
	fs.DurationVar(&o.MaxConnIdleTime, namePrefix+"max-conn-idle-time", o.MaxConnIdleTime, "MongoDB max connection idle time")
	fs.DurationVar(&o.ConnectTimeout, namePrefix+"connect-timeout", o.ConnectTimeout, "MongoDB connect timeout")
	fs.DurationVar(&o.SocketTimeout, namePrefix+"socket-timeout", o.SocketTimeout, "MongoDB socket timeout")
	fs.DurationVar(&o.ServerSelectionTimeout, namePrefix+"server-selection-timeout", o.ServerSelectionTimeout, "MongoDB server selection timeout")
	fs.StringVar(&o.ReplicaSet, namePrefix+"replica-set", o.ReplicaSet, "MongoDB replica set")
	fs.StringVar(&o.AuthSource, namePrefix+"auth-source", o.AuthSource, "MongoDB auth source")
	fs.BoolVar(&o.Direct, namePrefix+"direct", o.Direct, "MongoDB direct connection")
}
