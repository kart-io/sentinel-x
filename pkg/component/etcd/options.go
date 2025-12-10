package etcd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
)

// redactedPassword is the placeholder used when serializing passwords.
const redactedPassword = "[REDACTED]"

// Options defines configuration options for Etcd.
type Options struct {
	Endpoints      []string      `json:"endpoints" mapstructure:"endpoints"`
	Username       string        `json:"username" mapstructure:"username"`
	Password       string        `json:"-" mapstructure:"password"` // Excluded from JSON serialization
	DialTimeout    time.Duration `json:"dial-timeout" mapstructure:"dial-timeout"`
	RequestTimeout time.Duration `json:"request-timeout" mapstructure:"request-timeout"`
	LeaseTTL       int64         `json:"lease-ttl" mapstructure:"lease-ttl"`
}

// optionsForJSON is used for JSON marshaling with password redacted.
type optionsForJSON struct {
	Endpoints      []string      `json:"endpoints"`
	Username       string        `json:"username"`
	Password       string        `json:"password"`
	DialTimeout    time.Duration `json:"dial-timeout"`
	RequestTimeout time.Duration `json:"request-timeout"`
	LeaseTTL       int64         `json:"lease-ttl"`
}

// MarshalJSON implements json.Marshaler with password redaction.
// This prevents accidental password exposure in logs or debug output.
func (o *Options) MarshalJSON() ([]byte, error) {
	password := redactedPassword
	if o.Password == "" {
		password = ""
	}

	return json.Marshal(optionsForJSON{
		Endpoints:      o.Endpoints,
		Username:       o.Username,
		Password:       password,
		DialTimeout:    o.DialTimeout,
		RequestTimeout: o.RequestTimeout,
		LeaseTTL:       o.LeaseTTL,
	})
}

// String returns a string representation with password redacted.
// Safe for logging and debugging.
func (o *Options) String() string {
	password := redactedPassword
	if o.Password == "" {
		password = ""
	}
	return fmt.Sprintf("Etcd{endpoints=%v, user=%s, password=%s}",
		o.Endpoints, o.Username, password)
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Endpoints:      []string{"127.0.0.1:2379"},
		Username:       "",
		Password:       "",
		DialTimeout:    5 * time.Second,
		RequestTimeout: 2 * time.Second,
		LeaseTTL:       60,
	}
}

// Complete fills in any fields not set that are required to have valid data.
// For Etcd options, this method currently has no completion logic as all
// defaults are set in NewOptions(). This method is provided to satisfy the
// component.ConfigOptions interface.
func (o *Options) Complete() error {
	return nil
}

// Validate checks if the options are valid.
func (o *Options) Validate() error {
	// 如果 CLI 参数为空，从环境变量读取
	if o.Password == "" {
		o.Password = os.Getenv("ETCD_PASSWORD")
	}

	// 警告使用 CLI 参数传递密码
	// 如果密码非空但环境变量为空，说明密码是通过 CLI 传递的
	if o.Password != "" && os.Getenv("ETCD_PASSWORD") == "" {
		fmt.Fprintf(os.Stderr, "WARNING: Passing Etcd password via CLI is insecure. Use ETCD_PASSWORD environment variable instead.\n")
	}

	return nil
}

// AddFlags adds flags for Etcd options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, namePrefix string) {
	fs.StringSliceVar(&o.Endpoints, namePrefix+"endpoints", o.Endpoints, "Etcd endpoints")
	fs.StringVar(&o.Username, namePrefix+"username", o.Username, "Etcd username")
	fs.StringVar(&o.Password, namePrefix+"password", o.Password, "Etcd password (DEPRECATED: use ETCD_PASSWORD env var instead)")
	fs.DurationVar(&o.DialTimeout, namePrefix+"dial-timeout", o.DialTimeout, "Etcd dial timeout")
	fs.DurationVar(&o.RequestTimeout, namePrefix+"request-timeout", o.RequestTimeout, "Etcd request timeout")
	fs.Int64Var(&o.LeaseTTL, namePrefix+"lease-ttl", o.LeaseTTL, "Etcd lease TTL")
}
