package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareRequestID, func() MiddlewareConfig {
		return NewRequestIDOptions()
	})
}

// 确保 RequestIDOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*RequestIDOptions)(nil)

// RequestIDOptions defines request ID middleware options.
// 此结构体必须保持可 JSON 序列化，运行时依赖（如 Generator）应通过函数参数注入。
type RequestIDOptions struct {
	Header string `json:"header" mapstructure:"header"`
	// GeneratorType 指定 ID 生成器类型
	// 支持的值:
	//   - "random" 或 "hex": 使用加密随机十六进制生成器(默认,32字符)
	//   - "ulid": 使用 ULID 生成器(推荐,26字符,时间可排序,性能提升3x)
	GeneratorType string `json:"generator_type" mapstructure:"generator_type"`
}

// NewRequestIDOptions creates default request ID middleware options.
func NewRequestIDOptions() *RequestIDOptions {
	return &RequestIDOptions{
		Header:        "X-Request-ID",
		GeneratorType: "random", // 默认使用随机十六进制生成器(向后兼容)
	}
}

// AddFlags adds flags for request ID options to the specified FlagSet.
func (o *RequestIDOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Header, options.Join(prefixes...)+"middleware.request-id.header", o.Header, "Request ID header name.")
	fs.StringVar(&o.GeneratorType, options.Join(prefixes...)+"middleware.request-id.generator", o.GeneratorType, "ID generator type: random/hex (default, 32 chars) or ulid (recommended, 26 chars, sortable, 3x faster).")
}

// Validate validates the request ID options.
func (o *RequestIDOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if o.Header == "" {
		errs = append(errs, errors.New("request ID header name is required"))
	}
	// 验证生成器类型
	validTypes := map[string]bool{
		"random": true,
		"hex":    true,
		"ulid":   true,
		"":       true, // 空值将使用默认值
	}
	if !validTypes[o.GeneratorType] {
		errs = append(errs, errors.New("invalid generator type: must be 'random', 'hex', or 'ulid'"))
	}
	return errs
}

// Complete completes the request ID options with defaults.
func (o *RequestIDOptions) Complete() error {
	return nil
}
