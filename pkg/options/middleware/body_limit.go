package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareBodyLimit, func() MiddlewareConfig {
		return NewBodyLimitOptions()
	})
}

// 确保 BodyLimitOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*BodyLimitOptions)(nil)

// BodyLimitOptions 定义请求体大小限制中间件的配置选项。
// 用于防止恶意客户端发送超大请求体导致服务器资源耗尽（DoS 攻击）。
type BodyLimitOptions struct {
	// MaxSize 最大请求体大小（字节）。
	// 默认值：4MB（4 * 1024 * 1024）
	// 建议值：根据业务需求调整，常见值：
	//   - API 服务：1MB - 10MB
	//   - 文件上传服务：100MB - 1GB
	MaxSize int64 `json:"max-size" mapstructure:"max-size"`

	// SkipPaths 跳过检查的精确路径列表。
	// 示例：["/upload", "/webhook"]
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`

	// SkipPathPrefixes 跳过检查的路径前缀列表。
	// 示例：["/api/v1/files", "/internal"]
	SkipPathPrefixes []string `json:"skip-path-prefixes" mapstructure:"skip-path-prefixes"`
}

// NewBodyLimitOptions 创建默认的 BodyLimit 中间件配置。
// 默认最大请求体大小为 4MB，适用于大多数 API 场景。
func NewBodyLimitOptions() *BodyLimitOptions {
	return &BodyLimitOptions{
		MaxSize:          4 * 1024 * 1024, // 4MB
		SkipPaths:        []string{},
		SkipPathPrefixes: []string{},
	}
}

// AddFlags 添加 BodyLimit 配置的命令行标志。
func (o *BodyLimitOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.Int64Var(&o.MaxSize, options.Join(prefixes...)+"middleware.body-limit.max-size", o.MaxSize, "Maximum request body size in bytes.")
	fs.StringSliceVar(&o.SkipPaths, options.Join(prefixes...)+"middleware.body-limit.skip-paths", o.SkipPaths, "Skip paths for body limit middleware.")
	fs.StringSliceVar(&o.SkipPathPrefixes, options.Join(prefixes...)+"middleware.body-limit.skip-path-prefixes", o.SkipPathPrefixes, "Skip path prefixes for body limit middleware.")
}

// Validate 验证 BodyLimit 配置的有效性。
func (o *BodyLimitOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if o.MaxSize <= 0 {
		errs = append(errs, errors.New("body-limit: MaxSize must be greater than 0"))
	}
	// 建议：检查是否设置了过大的值（可能导致内存问题）
	const maxReasonableSize = 1 * 1024 * 1024 * 1024 // 1GB
	if o.MaxSize > maxReasonableSize {
		// 警告：不阻止，但记录在错误列表中
		// 实际使用时可以通过日志警告而不是返回错误
	}
	return errs
}

// Complete 完成 BodyLimit 配置的默认值填充。
func (o *BodyLimitOptions) Complete() error {
	return nil
}
