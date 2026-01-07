package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareCompression, func() MiddlewareConfig {
		return NewCompressionOptions()
	})
}

// 确保 CompressionOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*CompressionOptions)(nil)

// CompressionOptions 定义响应压缩中间件的配置选项。
// 自动压缩 HTTP 响应以减少带宽消耗，提升传输性能。
type CompressionOptions struct {
	// Level 压缩级别（1-9）。
	// 1 = 最快速度，最低压缩率
	// 9 = 最慢速度，最高压缩率
	// 6 = 推荐默认值（平衡性能和压缩率）
	// -1 = 使用默认压缩级别（等同于 6）
	Level int `json:"level" mapstructure:"level"`

	// MinSize 最小压缩大小（字节）。
	// 小于此值的响应体不进行压缩，避免压缩小内容反而增加开销。
	// 默认值：1024 字节（1KB）
	MinSize int `json:"min-size" mapstructure:"min-size"`

	// Types 需要压缩的 Content-Type 列表。
	// 仅匹配的 Content-Type 会被压缩，避免压缩已压缩内容（如图片、视频）。
	// 默认包含常见的文本类型（JSON、HTML、CSS、JS 等）。
	Types []string `json:"types" mapstructure:"types"`

	// SkipPaths 跳过压缩的精确路径列表。
	// 某些路径可能不适合压缩（如已压缩文件下载、流式传输）。
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`

	// SkipPathPrefixes 跳过压缩的路径前缀列表。
	// 示例：["/api/v1/download", "/streaming"]
	SkipPathPrefixes []string `json:"skip-path-prefixes" mapstructure:"skip-path-prefixes"`
}

// NewCompressionOptions 创建默认的 Compression 中间件配置。
// 默认使用中等压缩级别（6），最小压缩大小 1KB，支持常见文本类型。
func NewCompressionOptions() *CompressionOptions {
	return &CompressionOptions{
		Level:   6,    // 默认中等压缩
		MinSize: 1024, // 默认 1KB
		Types: []string{
			"application/json",
			"application/javascript",
			"application/xml",
			"text/html",
			"text/css",
			"text/plain",
			"text/xml",
			"text/javascript",
		},
		SkipPaths:        []string{},
		SkipPathPrefixes: []string{},
	}
}

// AddFlags 添加 Compression 配置的命令行标志。
func (o *CompressionOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.IntVar(&o.Level, options.Join(prefixes...)+"middleware.compression.level", o.Level, "Compression level (1-9, 6 is recommended).")
	fs.IntVar(&o.MinSize, options.Join(prefixes...)+"middleware.compression.min-size", o.MinSize, "Minimum size in bytes to compress.")
	fs.StringSliceVar(&o.Types, options.Join(prefixes...)+"middleware.compression.types", o.Types, "Content-Type list to compress.")
	fs.StringSliceVar(&o.SkipPaths, options.Join(prefixes...)+"middleware.compression.skip-paths", o.SkipPaths, "Skip paths for compression middleware.")
	fs.StringSliceVar(&o.SkipPathPrefixes, options.Join(prefixes...)+"middleware.compression.skip-path-prefixes", o.SkipPathPrefixes, "Skip path prefixes for compression middleware.")
}

// Validate 验证 Compression 配置的有效性。
func (o *CompressionOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error

	// 验证压缩级别范围
	if o.Level < -1 || o.Level > 9 {
		errs = append(errs, errors.New("compression: Level must be between -1 and 9"))
	}

	// 验证最小压缩大小
	if o.MinSize < 0 {
		errs = append(errs, errors.New("compression: MinSize must be non-negative"))
	}

	// 验证至少有一个 Content-Type
	if len(o.Types) == 0 {
		errs = append(errs, errors.New("compression: Types must not be empty, at least one Content-Type should be specified"))
	}

	return errs
}

// Complete 完成 Compression 配置的默认值填充。
func (o *CompressionOptions) Complete() error {
	// 如果压缩级别为 -1，使用默认级别 6
	if o.Level == -1 {
		o.Level = 6
	}
	return nil
}
