// Package middleware provides middleware configuration options.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
)

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

// Factory 定义中间件工厂接口。
// 每个中间件必须实现此接口以支持插拔式架构。
type Factory interface {
	// Name 返回中间件名称，必须与配置注册名称一致。
	Name() string

	// Create 根据配置创建 Gin 中间件处理函数。
	// cfg 是对应的中间件配置，调用者需确保类型正确。
	Create(cfg MiddlewareConfig) (gin.HandlerFunc, error)

	// NeedsRuntime 返回是否需要运行时依赖。
	// 如果返回 true，该中间件不会从配置文件自动加载，
	// 需要在业务代码中手动创建并注入依赖。
	// 典型场景：auth（需要 JWT 验证器）、rate-limit（需要 Redis 客户端）
	NeedsRuntime() bool
}

// RouteRegistrar 定义路由注册接口。
// 某些中间件需要注册独立路由（如 health、metrics、pprof、version）。
type RouteRegistrar interface {
	// RegisterRoutes 注册中间件的路由。
	RegisterRoutes(engine *gin.Engine, cfg MiddlewareConfig) error
}
