// Package middleware provides version middleware options.
package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

// VersionOptions contains version endpoint configuration.
type VersionOptions struct {
	// Enabled enables the version endpoint.
	Enabled bool `json:"enabled" mapstructure:"enabled"`
	// Path specifies the version endpoint path.
	Path string `json:"path" mapstructure:"path"`
	// HideDetails hides sensitive build details (commit hash, build date).
	HideDetails bool `json:"hide-details" mapstructure:"hide-details"`
}

// NewVersionOptions creates default version options.
func NewVersionOptions() *VersionOptions {
	return &VersionOptions{
		Enabled:     true,  // 默认启用
		Path:        "/version",
		HideDetails: false, // 默认显示完整信息
	}
}

// Validate validates version options.
func (o *VersionOptions) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	// 路径必须以 / 开头
	if o.Enabled && o.Path != "" && o.Path[0] != '/' {
		errs = append(errs, errors.New("middleware.version.path must start with '/'"))
	}

	return errs
}

// AddFlags adds flags for version options to the specified FlagSet.
func (o *VersionOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	prefix := options.Join(prefixes...) + "middleware.version."

	fs.BoolVar(&o.Enabled, prefix+"enabled", o.Enabled,
		"Enable version endpoint.")
	fs.StringVar(&o.Path, prefix+"path", o.Path,
		"Version endpoint path.")
	fs.BoolVar(&o.HideDetails, prefix+"hide-details", o.HideDetails,
		"Hide sensitive build details in version response.")
}

// Complete completes version options with defaults.
func (o *VersionOptions) Complete() error {
	if o.Path == "" {
		o.Path = "/version"
	}
	return nil
}
