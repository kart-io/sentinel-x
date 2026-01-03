// Package cache provides cache configuration options.
package cache

import (
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

// Options 查询缓存配置。
type Options struct {
	// Enabled 是否启用缓存。
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// TTL 缓存过期时间。
	TTL time.Duration `json:"ttl" mapstructure:"ttl"`

	// KeyPrefix 缓存键前缀。
	KeyPrefix string `json:"key-prefix" mapstructure:"key-prefix"`

	// Redis Redis 连接配置。
	Redis *redisopts.Options `json:"redis" mapstructure:"redis"`
}

// NewOptions 创建默认缓存配置。
func NewOptions() *Options {
	return &Options{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "cache:",
		Redis:     redisopts.NewOptions(),
	}
}

// AddFlags adds flags for cache options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.BoolVar(&o.Enabled, options.Join(prefixes...)+"cache.enabled", o.Enabled, "Enable cache.")
	fs.DurationVar(&o.TTL, options.Join(prefixes...)+"cache.ttl", o.TTL, "Cache TTL duration.")
	fs.StringVar(&o.KeyPrefix, options.Join(prefixes...)+"cache.key-prefix", o.KeyPrefix, "Cache key prefix.")

	if o.Redis == nil {
		o.Redis = redisopts.NewOptions()
	}
	o.Redis.AddFlags(fs, prefixes...)
}

// Validate validates the cache options.
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	if o.Enabled && o.Redis != nil {
		errs = append(errs, o.Redis.Validate()...)
	}
	return errs
}

// Complete completes the cache options with defaults.
func (o *Options) Complete() error {
	if o.Redis == nil {
		o.Redis = redisopts.NewOptions()
	}
	return o.Redis.Complete()
}
