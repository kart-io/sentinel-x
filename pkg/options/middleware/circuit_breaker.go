package middleware

import (
	"errors"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareCircuitBreaker, func() MiddlewareConfig {
		return NewCircuitBreakerOptions()
	})
}

// 确保 CircuitBreakerOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*CircuitBreakerOptions)(nil)

// CircuitBreakerOptions 定义熔断器中间件的配置选项（纯配置，可 JSON 序列化）。
// 是否启用由 middleware 数组配置控制，而非 Enabled 字段。
type CircuitBreakerOptions struct {
	// MaxFailures 触发熔断的最大失败次数。
	MaxFailures int `json:"max-failures" mapstructure:"max-failures"`

	// Timeout 熔断器打开后的超时时间（秒）。
	Timeout int `json:"timeout" mapstructure:"timeout"`

	// HalfOpenMaxCalls 半开状态允许的最大调用次数。
	HalfOpenMaxCalls int `json:"half-open-max-calls" mapstructure:"half-open-max-calls"`

	// SkipPaths 是跳过熔断的路径列表（如健康检查、监控端点）。
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`

	// SkipPathPrefixes 是跳过熔断的路径前缀列表（如 /static/）。
	SkipPathPrefixes []string `json:"skip-path-prefixes" mapstructure:"skip-path-prefixes"`

	// ErrorThreshold HTTP 状态码阈值，>= 该值视为失败。
	// 例如：500 表示 5xx 错误触发熔断，400 表示 4xx 和 5xx 都触发。
	ErrorThreshold int `json:"error-threshold" mapstructure:"error-threshold"`
}

// NewCircuitBreakerOptions 创建默认的熔断器选项。
func NewCircuitBreakerOptions() *CircuitBreakerOptions {
	return &CircuitBreakerOptions{
		MaxFailures:      5,
		Timeout:          60, // 60 秒
		HalfOpenMaxCalls: 1,
		SkipPaths:        []string{"/health", "/metrics"},
		SkipPathPrefixes: []string{},
		ErrorThreshold:   500, // 默认 5xx 错误触发熔断
	}
}

// AddFlags 为熔断器选项添加标志到指定的 FlagSet。
func (o *CircuitBreakerOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	prefix := options.Join(prefixes...) + "middleware.circuit-breaker."

	fs.IntVar(&o.MaxFailures, prefix+"max-failures", o.MaxFailures, "Maximum number of failures before opening circuit breaker.")
	fs.IntVar(&o.Timeout, prefix+"timeout", o.Timeout, "Circuit breaker timeout duration (seconds).")
	fs.IntVar(&o.HalfOpenMaxCalls, prefix+"half-open-max-calls", o.HalfOpenMaxCalls, "Maximum calls allowed in half-open state.")
	fs.StringSliceVar(&o.SkipPaths, prefix+"skip-paths", o.SkipPaths, "List of paths to skip circuit breaker.")
	fs.StringSliceVar(&o.SkipPathPrefixes, prefix+"skip-path-prefixes", o.SkipPathPrefixes, "List of path prefixes to skip circuit breaker.")
	fs.IntVar(&o.ErrorThreshold, prefix+"error-threshold", o.ErrorThreshold, "HTTP status code threshold for errors (>= this value triggers circuit breaker).")
}

// Validate 验证熔断器选项。
func (o *CircuitBreakerOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if o.MaxFailures <= 0 {
		errs = append(errs, errors.New("max failures must be positive"))
	}
	if o.Timeout <= 0 {
		errs = append(errs, errors.New("timeout must be positive"))
	}
	if o.HalfOpenMaxCalls <= 0 {
		errs = append(errs, errors.New("half-open max calls must be positive"))
	}
	if o.ErrorThreshold < 400 || o.ErrorThreshold > 599 {
		errs = append(errs, errors.New("error threshold must be between 400 and 599"))
	}
	return errs
}

// Complete 完成熔断器选项的默认值设置。
func (o *CircuitBreakerOptions) Complete() error {
	return nil
}

// GetTimeout 返回超时时间的 time.Duration 表示。
func (o *CircuitBreakerOptions) GetTimeout() time.Duration {
	return time.Duration(o.Timeout) * time.Second
}
