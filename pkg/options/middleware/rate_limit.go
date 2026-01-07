package middleware

import (
	"errors"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareRateLimit, func() MiddlewareConfig {
		return NewRateLimitOptions()
	})
}

// 确保 RateLimitOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*RateLimitOptions)(nil)

// RateLimitOptions 定义限流中间件的配置选项（纯配置，可 JSON 序列化）。
type RateLimitOptions struct {
	// Limit 是时间窗口内允许的最大请求数。
	Limit int `json:"limit" mapstructure:"limit"`

	// Window 是限流时间窗口（秒）。
	Window int `json:"window" mapstructure:"window"`

	// SkipPaths 是跳过限流的路径列表。
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`

	// TrustedProxies 是受信任的代理 IP 地址或 CIDR 范围列表。
	// 为空时，不信任代理头（X-Forwarded-For, X-Real-IP）。
	// 示例：["127.0.0.1", "10.0.0.0/8", "172.16.0.0/12"]
	TrustedProxies []string `json:"trusted-proxies" mapstructure:"trusted-proxies"`

	// TrustProxyHeaders 控制是否信任代理头来提取 IP。
	// 即使为 true，也仅在请求来自 TrustedProxies 时才信任头。
	TrustProxyHeaders bool `json:"trust-proxy-headers" mapstructure:"trust-proxy-headers"`

	// UseRedis 是否使用 Redis 作为限流器后端。
	// false 时使用内存限流器。
	UseRedis bool `json:"use-redis" mapstructure:"use-redis"`

	// RedisAddr 是 Redis 服务器地址（仅在 UseRedis=true 时使用）。
	RedisAddr string `json:"redis-addr" mapstructure:"redis-addr"`

	// RedisPassword 是 Redis 密码（可选）。
	RedisPassword string `json:"redis-password" mapstructure:"redis-password"`

	// RedisDB 是 Redis 数据库编号（可选）。
	RedisDB int `json:"redis-db" mapstructure:"redis-db"`
}

// NewRateLimitOptions 创建默认的限流选项。
func NewRateLimitOptions() *RateLimitOptions {
	return &RateLimitOptions{
		Limit:             100,
		Window:            60, // 60 秒 = 1 分钟
		SkipPaths:         []string{},
		TrustedProxies:    []string{},
		TrustProxyHeaders: false,
		UseRedis:          false,
		RedisAddr:         "localhost:6379",
		RedisPassword:     "",
		RedisDB:           0,
	}
}

// AddFlags 为限流选项添加标志到指定的 FlagSet。
func (o *RateLimitOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	prefix := options.Join(prefixes...) + "middleware.rate-limit."

	fs.IntVar(&o.Limit, prefix+"limit", o.Limit, "Maximum number of requests allowed within the time window.")
	fs.IntVar(&o.Window, prefix+"window", o.Window, "Time window duration for rate limiting (seconds).")
	fs.StringSliceVar(&o.SkipPaths, prefix+"skip-paths", o.SkipPaths, "List of paths to skip rate limiting.")
	fs.StringSliceVar(&o.TrustedProxies, prefix+"trusted-proxies", o.TrustedProxies, "List of trusted proxy IP addresses or CIDR ranges.")
	fs.BoolVar(&o.TrustProxyHeaders, prefix+"trust-proxy-headers", o.TrustProxyHeaders, "Trust proxy headers for IP extraction.")
	fs.BoolVar(&o.UseRedis, prefix+"use-redis", o.UseRedis, "Use Redis as rate limiter backend.")
	fs.StringVar(&o.RedisAddr, prefix+"redis-addr", o.RedisAddr, "Redis server address.")
	fs.StringVar(&o.RedisPassword, prefix+"redis-password", o.RedisPassword, "Redis password.")
	fs.IntVar(&o.RedisDB, prefix+"redis-db", o.RedisDB, "Redis database number.")
}

// Validate 验证限流选项。
func (o *RateLimitOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if o.Limit <= 0 {
		errs = append(errs, errors.New("rate limit must be positive"))
	}
	if o.Window <= 0 {
		errs = append(errs, errors.New("rate limit window must be positive"))
	}
	if o.UseRedis && o.RedisAddr == "" {
		errs = append(errs, errors.New("redis address is required when UseRedis is true"))
	}
	return errs
}

// Complete 完成限流选项的默认值设置。
func (o *RateLimitOptions) Complete() error {
	return nil
}

// GetWindow 返回时间窗口的 time.Duration 表示。
func (o *RateLimitOptions) GetWindow() time.Duration {
	return time.Duration(o.Window) * time.Second
}
