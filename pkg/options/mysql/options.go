package mysql

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
)

// Options defines configuration options for MySQL.
type Options struct {
	Host                  string        `json:"host" mapstructure:"host"`
	Port                  int           `json:"port" mapstructure:"port"`
	Username              string        `json:"username" mapstructure:"username"`
	Password              string        `json:"password" mapstructure:"password"`
	Database              string        `json:"database" mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections" mapstructure:"max-idle-connections"`
	MaxOpenConnections    int           `json:"max-open-connections" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Host:                  "127.0.0.1",
		Port:                  3306,
		Username:              "root",
		Password:              "",
		Database:              "",
		MaxIdleConnections:    10,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: 10 * time.Second,
		LogLevel:              1, // Silent
	}
}

// Validate checks if the options are valid.
func (o *Options) Validate() error {
	// 如果 CLI 参数为空，从环境变量读取
	if o.Password == "" {
		o.Password = os.Getenv("MYSQL_PASSWORD")
	}

	// 警告使用 CLI 参数传递密码
	// 如果密码非空但环境变量为空，说明密码是通过 CLI 传递的
	if o.Password != "" && os.Getenv("MYSQL_PASSWORD") == "" {
		fmt.Fprintf(os.Stderr, "WARNING: Passing MySQL password via CLI is insecure. Use MYSQL_PASSWORD environment variable instead.\n")
	}

	return nil
}

// AddFlags adds flags for MySQL options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Host, "mysql.host", o.Host, "MySQL host")
	fs.IntVar(&o.Port, "mysql.port", o.Port, "MySQL port")
	fs.StringVar(&o.Username, "mysql.username", o.Username, "MySQL username")
	fs.StringVar(&o.Password, "mysql.password", o.Password, "MySQL password (DEPRECATED: use MYSQL_PASSWORD env var instead)")
	fs.StringVar(&o.Database, "mysql.database", o.Database, "MySQL database")
	fs.IntVar(&o.MaxIdleConnections, "mysql.max-idle-connections", o.MaxIdleConnections, "MySQL max idle connections")
	fs.IntVar(&o.MaxOpenConnections, "mysql.max-open-connections", o.MaxOpenConnections, "MySQL max open connections")
	fs.DurationVar(&o.MaxConnectionLifeTime, "mysql.max-connection-life-time", o.MaxConnectionLifeTime, "MySQL max connection life time")
	fs.IntVar(&o.LogLevel, "mysql.log-level", o.LogLevel, "MySQL log level")
}
