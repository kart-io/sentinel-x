// Package middleware provides middleware configuration options.
package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigError 表示配置错误。
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config error in field %q: %s", e.Field, e.Message)
	}
	return e.Message
}

// PathMatcher contains common path matching configuration.
type PathMatcher struct {
	SkipPaths        []string
	SkipPathPrefixes []string
}

// 中间件名称常量。
const (
	MiddlewareRecovery        = "recovery"
	MiddlewareRequestID       = "request-id"
	MiddlewareLogger          = "logger"
	MiddlewareCORS            = "cors"
	MiddlewareBodyLimit       = "body-limit"
	MiddlewareTimeout         = "timeout"
	MiddlewareHealth          = "health"
	MiddlewareMetrics         = "metrics"
	MiddlewarePprof           = "pprof"
	MiddlewareAuth            = "auth"
	MiddlewareAuthz           = "authz"
	MiddlewareVersion         = "version"
	MiddlewareCompression     = "compression"
	MiddlewareSecurityHeaders = "security-headers"
	MiddlewareRateLimit       = "rate-limit"
	MiddlewareCircuitBreaker  = "circuit-breaker"
)

// Options 纯动态中间件配置。
// 所有中间件配置统一存储在 configs map 中，支持完全的插拔式扩展。
// 是否启用中间件由 Middleware 数组配置控制，而非各配置的 Enabled 字段。
type Options struct {
	// Middleware 指定中间件的应用顺序。
	// 如果为空，则使用默认顺序。
	// 示例: ["recovery", "request-id", "logger", "cors", "timeout"]
	Middleware []string `json:"middleware" mapstructure:"middleware"`

	// mu 保护 configs map 的并发访问。
	mu sync.RWMutex

	// configs 动态存储所有中间件配置。
	// 键为中间件名称（如 "recovery"），值为具体配置实例。
	configs map[string]MiddlewareConfig
}

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions 创建默认中间件选项。
// 默认启用 Recovery, RequestID, Logger, Health, Metrics, Version 中间件。
func NewOptions() *Options {
	o := &Options{
		configs: make(map[string]MiddlewareConfig),
	}

	// 设置默认启用的中间件（通过注册机制创建）
	defaultEnabled := []string{
		MiddlewareRecovery,
		MiddlewareRequestID,
		MiddlewareLogger,
		MiddlewareHealth,
		MiddlewareMetrics,
		MiddlewareVersion,
	}

	for _, name := range defaultEnabled {
		if cfg, err := Create(name); err == nil {
			o.configs[name] = cfg
		}
	}

	return o
}

// LoadFromViper 从 viper 加载中间件配置。
// 这是纯动态架构的核心方法，根据注册表动态解析配置。
func (o *Options) LoadFromViper(v *viper.Viper) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.configs == nil {
		o.configs = make(map[string]MiddlewareConfig)
	}

	// 阶段1：加载 middleware 顺序数组
	if v.IsSet("middleware") {
		if err := v.UnmarshalKey("middleware", &o.Middleware); err != nil {
			return fmt.Errorf("unmarshal middleware order: %w", err)
		}
	}

	// 阶段2：遍历所有已注册的中间件名称
	for _, name := range ListRegistered() {
		if !v.IsSet(name) {
			continue // 该中间件未在配置文件中配置
		}

		// 从注册表创建配置实例（确保类型正确）
		cfg, err := Create(name)
		if err != nil {
			return fmt.Errorf("create config for %s: %w", name, err)
		}

		// 使用 viper 解析该中间件的配置
		if err := v.UnmarshalKey(name, cfg); err != nil {
			return fmt.Errorf("unmarshal config for %s: %w", name, err)
		}

		o.configs[name] = cfg
	}

	return nil
}

// SetConfig 设置指定中间件的配置。
// 这是插拔式扩展的入口，允许动态添加新中间件配置。
func (o *Options) SetConfig(name string, cfg MiddlewareConfig) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.configs == nil {
		o.configs = make(map[string]MiddlewareConfig)
	}
	o.configs[name] = cfg
}

// GetConfig 获取指定中间件的配置。
func (o *Options) GetConfig(name string) MiddlewareConfig {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.configs == nil {
		return nil
	}
	return o.configs[name]
}

// GetOrCreate 获取或创建配置实例。
// 如果配置不存在，则从注册表创建新实例。
func (o *Options) GetOrCreate(name string) MiddlewareConfig {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.configs == nil {
		o.configs = make(map[string]MiddlewareConfig)
	}

	if cfg, ok := o.configs[name]; ok {
		return cfg
	}

	cfg, err := Create(name)
	if err != nil {
		return nil
	}
	o.configs[name] = cfg
	return cfg
}

// DeleteConfig 删除指定中间件的配置（禁用该中间件）。
func (o *Options) DeleteConfig(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.configs != nil {
		delete(o.configs, name)
	}
}

// GetConfigTyped 获取指定中间件的配置并进行类型断言（泛型）。
// 用于需要具体类型的场景，避免调用方手动断言。
func GetConfigTyped[T MiddlewareConfig](o *Options, name string) (T, bool) {
	cfg := o.GetConfig(name)
	if cfg == nil {
		var zero T
		return zero, false
	}
	typed, ok := cfg.(T)
	return typed, ok
}

// Validate 验证所有中间件配置。
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	var errs []error

	// 验证 Middleware 顺序配置
	errs = append(errs, o.validateMiddlewareLocked()...)

	// 遍历 configs 验证所有配置
	for name, cfg := range o.configs {
		if cfg != nil {
			for _, err := range cfg.Validate() {
				errs = append(errs, &ConfigError{
					Field:   name,
					Message: err.Error(),
				})
			}
		}
	}

	return errs
}

// Complete 完成所有中间件配置的默认值填充。
func (o *Options) Complete() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.configs == nil {
		o.configs = make(map[string]MiddlewareConfig)
	}

	for name, cfg := range o.configs {
		if cfg != nil {
			if err := cfg.Complete(); err != nil {
				return &ConfigError{
					Field:   name,
					Message: err.Error(),
				}
			}
		}
	}

	return nil
}

// IsEnabled 检查指定中间件是否启用。
// 通过检查 configs map 中是否存在且非 nil 来判断。
// 是否启用完全由 configs map 控制，无需检查各配置的 Enabled 字段。
func (o *Options) IsEnabled(name string) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.configs == nil {
		return false
	}

	cfg, ok := o.configs[name]
	return ok && cfg != nil
}

// GetEnabledMiddlewares 返回所有启用的中间件名称列表。
func (o *Options) GetEnabledMiddlewares() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.configs == nil {
		return nil
	}

	var enabled []string
	for name, cfg := range o.configs {
		if cfg != nil {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// DefaultMiddlewareOrder 返回默认的中间件应用顺序。
func DefaultMiddlewareOrder() []string {
	return []string{
		MiddlewareRecovery,  // 最高优先级，捕获 panic
		MiddlewareRequestID, // 为其他中间件提供 RequestID
		MiddlewareLogger,    // 依赖 RequestID
		MiddlewareMetrics,   // 监控指标收集
		MiddlewareCORS,      // 跨域支持
		MiddlewareTimeout,   // 超时控制
	}
}

// GetMiddlewareOrder 返回中间件应用顺序。
// 如果 Middleware 字段为空，返回默认顺序。
func (o *Options) GetMiddlewareOrder() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if len(o.Middleware) > 0 {
		return o.Middleware
	}
	return DefaultMiddlewareOrder()
}

// ValidateMiddleware 验证 Middleware 配置的有效性。
func (o *Options) ValidateMiddleware() []error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.validateMiddlewareLocked()
}

// validateMiddlewareLocked 内部方法，调用前需持有锁。
func (o *Options) validateMiddlewareLocked() []error {
	if len(o.Middleware) == 0 {
		return nil
	}

	var errs []error
	seen := make(map[string]bool)

	// 获取所有已注册的中间件名称
	registered := ListRegistered()
	registeredMap := make(map[string]bool, len(registered))
	for _, name := range registered {
		registeredMap[name] = true
	}

	for _, name := range o.Middleware {
		// 检查中间件名称是否有效（已注册）
		if !registeredMap[name] {
			errs = append(errs, &ConfigError{
				Field:   "middleware",
				Message: "unknown middleware: " + name,
			})
		}

		// 检查重复
		if seen[name] {
			errs = append(errs, &ConfigError{
				Field:   "middleware",
				Message: "duplicate middleware in list: " + name,
			})
		}
		seen[name] = true
	}

	return errs
}

// AddFlags 添加所有中间件配置的命令行标志。
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for _, cfg := range o.configs {
		if cfg != nil {
			cfg.AddFlags(fs, prefixes...)
		}
	}
}

// ListConfigs 返回所有已配置的中间件名称。
func (o *Options) ListConfigs() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.configs == nil {
		return nil
	}

	names := make([]string, 0, len(o.configs))
	for name := range o.configs {
		names = append(names, name)
	}
	return names
}

// ===== 泛型配置修改器 =====

// Configure 通用配置修改器（泛型）。
// T 必须是实现 MiddlewareConfig 的指针类型。
func Configure[T MiddlewareConfig](name string, modifier func(T)) Option {
	return func(o *Options) {
		cfg := o.GetOrCreate(name)
		if cfg == nil {
			return
		}
		if typed, ok := cfg.(T); ok {
			modifier(typed)
		}
	}
}

// Without 通用禁用函数。
func Without(name string) Option {
	return func(o *Options) {
		o.DeleteConfig(name)
	}
}

// ===== 便捷配置函数 =====

// WithRecovery 配置并启用 recovery 中间件。
func WithRecovery(enableStackTrace bool) Option {
	return Configure(MiddlewareRecovery, func(cfg *RecoveryOptions) {
		cfg.EnableStackTrace = enableStackTrace
	})
}

// WithoutRecovery 禁用 recovery 中间件。
func WithoutRecovery() Option { return Without(MiddlewareRecovery) }

// WithRequestID 配置并启用 request-id 中间件。
func WithRequestID(header string) Option {
	return Configure(MiddlewareRequestID, func(cfg *RequestIDOptions) {
		if header != "" {
			cfg.Header = header
		}
	})
}

// WithoutRequestID 禁用 request-id 中间件。
func WithoutRequestID() Option { return Without(MiddlewareRequestID) }

// WithLogger 配置并启用 logger 中间件。
func WithLogger(skipPaths ...string) Option {
	return Configure(MiddlewareLogger, func(cfg *LoggerOptions) {
		if len(skipPaths) > 0 {
			cfg.SkipPaths = skipPaths
		}
	})
}

// WithoutLogger 禁用 logger 中间件。
func WithoutLogger() Option { return Without(MiddlewareLogger) }

// WithCORS 配置并启用 CORS 中间件。
func WithCORS(origins ...string) Option {
	return Configure(MiddlewareCORS, func(cfg *CORSOptions) {
		if len(origins) > 0 {
			cfg.AllowOrigins = origins
		}
	})
}

// WithoutCORS 禁用 CORS 中间件。
func WithoutCORS() Option { return Without(MiddlewareCORS) }

// WithTimeout 配置并启用 timeout 中间件。
func WithTimeout(timeout time.Duration, skipPaths ...string) Option {
	return Configure(MiddlewareTimeout, func(cfg *TimeoutOptions) {
		if timeout > 0 {
			cfg.Timeout = timeout
		}
		if len(skipPaths) > 0 {
			cfg.SkipPaths = skipPaths
		}
	})
}

// WithoutTimeout 禁用 timeout 中间件。
func WithoutTimeout() Option { return Without(MiddlewareTimeout) }

// WithHealth 配置并启用 health 中间件。
func WithHealth(path, livenessPath, readinessPath string) Option {
	return Configure(MiddlewareHealth, func(cfg *HealthOptions) {
		if path != "" {
			cfg.Path = path
		}
		if livenessPath != "" {
			cfg.LivenessPath = livenessPath
		}
		if readinessPath != "" {
			cfg.ReadinessPath = readinessPath
		}
	})
}

// WithoutHealth 禁用 health 中间件。
func WithoutHealth() Option { return Without(MiddlewareHealth) }

// WithMetrics 配置并启用 metrics 中间件。
func WithMetrics(path, namespace, subsystem string) Option {
	return Configure(MiddlewareMetrics, func(cfg *MetricsOptions) {
		if path != "" {
			cfg.Path = path
		}
		if namespace != "" {
			cfg.Namespace = namespace
		}
		if subsystem != "" {
			cfg.Subsystem = subsystem
		}
	})
}

// WithoutMetrics 禁用 metrics 中间件。
func WithoutMetrics() Option { return Without(MiddlewareMetrics) }

// WithPprof 配置并启用 pprof 中间件。
func WithPprof(prefix string) Option {
	return Configure(MiddlewarePprof, func(cfg *PprofOptions) {
		if prefix != "" {
			cfg.Prefix = prefix
		}
	})
}

// WithoutPprof 禁用 pprof 中间件。
func WithoutPprof() Option { return Without(MiddlewarePprof) }

// WithAuth 配置并启用 auth 中间件。
func WithAuth(tokenLookup, authScheme string, skipPaths ...string) Option {
	return Configure(MiddlewareAuth, func(cfg *AuthOptions) {
		if tokenLookup != "" {
			cfg.TokenLookup = tokenLookup
		}
		if authScheme != "" {
			cfg.AuthScheme = authScheme
		}
		if len(skipPaths) > 0 {
			cfg.SkipPaths = skipPaths
		}
	})
}

// WithoutAuth 禁用 auth 中间件。
func WithoutAuth() Option { return Without(MiddlewareAuth) }

// WithAuthz 配置并启用 authz 中间件。
func WithAuthz() Option {
	return func(o *Options) {
		_ = o.GetOrCreate(MiddlewareAuthz)
	}
}

// WithoutAuthz 禁用 authz 中间件。
func WithoutAuthz() Option { return Without(MiddlewareAuthz) }

// WithVersion 配置并启用 version 中间件。
func WithVersion(path string, hideDetails bool) Option {
	return Configure(MiddlewareVersion, func(cfg *VersionOptions) {
		if path != "" {
			cfg.Path = path
		}
		cfg.HideDetails = hideDetails
	})
}

// WithoutVersion 禁用 version 中间件。
func WithoutVersion() Option { return Without(MiddlewareVersion) }

// WithBodyLimit 配置并启用 body-limit 中间件。
func WithBodyLimit(maxSize int64, skipPaths ...string) Option {
	return Configure(MiddlewareBodyLimit, func(cfg *BodyLimitOptions) {
		if maxSize > 0 {
			cfg.MaxSize = maxSize
		}
		if len(skipPaths) > 0 {
			cfg.SkipPaths = skipPaths
		}
	})
}

// WithoutBodyLimit 禁用 body-limit 中间件。
func WithoutBodyLimit() Option { return Without(MiddlewareBodyLimit) }

// WithCompression 配置并启用 compression 中间件。
func WithCompression(level int, minSize int, types ...string) Option {
	return Configure(MiddlewareCompression, func(cfg *CompressionOptions) {
		if level >= -1 && level <= 9 {
			cfg.Level = level
		}
		if minSize >= 0 {
			cfg.MinSize = minSize
		}
		if len(types) > 0 {
			cfg.Types = types
		}
	})
}

// WithoutCompression 禁用 compression 中间件。
func WithoutCompression() Option { return Without(MiddlewareCompression) }

// WithSecurityHeaders 配置并启用 security-headers 中间件。
func WithSecurityHeaders() Option {
	return func(o *Options) {
		_ = o.GetOrCreate(MiddlewareSecurityHeaders)
	}
}

// WithoutSecurityHeaders 禁用 security-headers 中间件。
func WithoutSecurityHeaders() Option { return Without(MiddlewareSecurityHeaders) }

// WithRateLimit 配置并启用 rate-limit 中间件。
func WithRateLimit() Option {
	return func(o *Options) {
		_ = o.GetOrCreate(MiddlewareRateLimit)
	}
}

// WithoutRateLimit 禁用 rate-limit 中间件。
func WithoutRateLimit() Option { return Without(MiddlewareRateLimit) }

// WithCircuitBreaker 配置并启用 circuit-breaker 中间件。
func WithCircuitBreaker() Option {
	return func(o *Options) {
		_ = o.GetOrCreate(MiddlewareCircuitBreaker)
	}
}

// WithoutCircuitBreaker 禁用 circuit-breaker 中间件。
func WithoutCircuitBreaker() Option { return Without(MiddlewareCircuitBreaker) }
