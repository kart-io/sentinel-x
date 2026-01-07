// Package middleware provides middleware configuration options.
package middleware

import "github.com/spf13/pflag"

// Config 定义中间件配置的统一接口。
// 所有中间件配置必须实现此接口以支持注册器模式。
type Config interface {
	// Validate 验证配置的有效性。
	Validate() []error

	// Complete 完成配置的默认值填充。
	Complete() error

	// AddFlags 添加命令行标志。
	AddFlags(fs *pflag.FlagSet, prefixes ...string)
}

// MiddlewareConfig 是 Config 的别名,保持向后兼容。
//
//nolint:revive // MiddlewareConfig 保持向后兼容性
type MiddlewareConfig = Config
