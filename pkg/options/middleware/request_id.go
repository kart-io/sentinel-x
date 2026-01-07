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
}

// NewRequestIDOptions creates default request ID middleware options.
func NewRequestIDOptions() *RequestIDOptions {
	return &RequestIDOptions{
		Header: "X-Request-ID",
	}
}

// AddFlags adds flags for request ID options to the specified FlagSet.
func (o *RequestIDOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Header, options.Join(prefixes...)+"middleware.request-id.header", o.Header, "Request ID header name.")
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
	return errs
}

// Complete completes the request ID options with defaults.
func (o *RequestIDOptions) Complete() error {
	return nil
}
