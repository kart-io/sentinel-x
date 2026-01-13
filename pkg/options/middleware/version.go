// Package middleware provides version middleware options.
package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareVersion, func() MiddlewareConfig {
		return NewVersionOptions()
	})
}

// 确保 VersionOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*VersionOptions)(nil)

// VersionOptions 包含版本端点配置。
// 是否启用由 middleware 数组配置控制，而非 Enabled 字段。
type VersionOptions struct {
	// Path 指定版本端点路径。
	Path string `json:"path" mapstructure:"path"`
	// HideDetails 隐藏敏感构建详情（commit hash、构建日期）。
	HideDetails bool `json:"hide-details" mapstructure:"hide-details"`
}

// NewVersionOptions 创建默认版本选项。
func NewVersionOptions() *VersionOptions {
	return &VersionOptions{
		Path:        "/version",
		HideDetails: false, // 默认显示完整信息
	}
}

// Validate 验证版本选项。
func (o *VersionOptions) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	// 路径必须以 / 开头
	if o.Path != "" && o.Path[0] != '/' {
		errs = append(errs, errors.New("middleware.version.path must start with '/'"))
	}

	return errs
}

// AddFlags 将版本选项的标志添加到指定的 FlagSet。
func (o *VersionOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	prefix := options.Join(prefixes...) + "middleware.version."

	fs.StringVar(&o.Path, prefix+"path", o.Path,
		"Version endpoint path.")
	fs.BoolVar(&o.HideDetails, prefix+"hide-details", o.HideDetails,
		"Hide sensitive build details in version response.")
}

// Complete 使用默认值完成版本选项。
func (o *VersionOptions) Complete() error {
	if o.Path == "" {
		o.Path = "/version"
	}
	return nil
}
