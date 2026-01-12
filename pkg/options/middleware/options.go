// Package middleware provides middleware configuration options.
package middleware

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
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

// AllMiddlewares 所有支持的中间件名称。
var AllMiddlewares = []string{
	MiddlewareRecovery,
	MiddlewareRequestID,
	MiddlewareLogger,
	MiddlewareCORS,
	MiddlewareBodyLimit,
	MiddlewareTimeout,
	MiddlewareHealth,
	MiddlewareMetrics,
	MiddlewarePprof,
	MiddlewareAuth,
	MiddlewareAuthz,
	MiddlewareVersion,
	MiddlewareCompression,
	MiddlewareSecurityHeaders,
	MiddlewareRateLimit,
	MiddlewareCircuitBreaker,
}

// Options contains all middleware configuration.
type Options struct {
	// Middleware 指定中间件的应用顺序。
	// 如果为空，则使用默认顺序。
	// 示例: ["recovery", "request-id", "logger", "cors", "timeout"]
	Middleware []string `json:"middleware" mapstructure:"middleware"`

	// Recovery 配置。
	Recovery *RecoveryOptions `json:"recovery" mapstructure:"recovery"`

	// RequestID 配置。
	RequestID *RequestIDOptions `json:"request-id" mapstructure:"request-id"`

	// Logger 配置。
	Logger *LoggerOptions `json:"logger" mapstructure:"logger"`

	// CORS 配置。
	CORS *CORSOptions `json:"cors" mapstructure:"cors"`

	// BodyLimit 配置。
	BodyLimit *BodyLimitOptions `json:"body-limit" mapstructure:"body-limit"`

	// Timeout 配置。
	Timeout *TimeoutOptions `json:"timeout" mapstructure:"timeout"`

	// Health 配置。
	Health *HealthOptions `json:"health" mapstructure:"health"`

	// Metrics 配置。
	Metrics *MetricsOptions `json:"metrics" mapstructure:"metrics"`

	// Pprof 配置。
	Pprof *PprofOptions `json:"pprof" mapstructure:"pprof"`

	// Auth 配置（JWT 认证）。
	Auth *AuthOptions `json:"auth" mapstructure:"auth"`

	// Authz 配置（RBAC 授权）。
	Authz *AuthzOptions `json:"authz" mapstructure:"authz"`

	// Version 配置。
	Version *VersionOptions `json:"version" mapstructure:"version"`

	// Compression 配置。
	Compression *CompressionOptions `json:"compression" mapstructure:"compression"`

	// SecurityHeaders 配置。
	SecurityHeaders *SecurityHeadersOptions `json:"security-headers" mapstructure:"security-headers"`

	// RateLimit 配置。
	RateLimit *RateLimitOptions `json:"rate-limit" mapstructure:"rate-limit"`

	// CircuitBreaker 配置。
	CircuitBreaker *CircuitBreakerOptions `json:"circuit-breaker" mapstructure:"circuit-breaker"`
}

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions creates default middleware options.
// 默认启用 Recovery, RequestID, Logger, Health, Metrics, Version 中间件。
// 其他中间件（CORS, BodyLimit, Timeout, Pprof, Auth, Authz, Compression）默认禁用（nil）。
func NewOptions() *Options {
	return &Options{
		Recovery:  NewRecoveryOptions(),
		RequestID: NewRequestIDOptions(),
		Logger:    NewLoggerOptions(),
		Health:    NewHealthOptions(),
		Metrics:   NewMetricsOptions(),
		Version:   NewVersionOptions(),
		// CORS, BodyLimit, Timeout, Pprof, Auth, Authz, Compression 默认禁用（nil）
	}
}

// Validate validates the middleware options.
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error

	// 验证 Middleware 配置
	errs = append(errs, o.ValidateMiddleware()...)

	// 验证启用的中间件配置
	if o.Recovery != nil {
		errs = append(errs, o.Recovery.Validate()...)
	}

	if o.RequestID != nil {
		errs = append(errs, o.RequestID.Validate()...)
	}

	if o.Logger != nil {
		errs = append(errs, o.Logger.Validate()...)
	}

	if o.CORS != nil {
		errs = append(errs, o.CORS.Validate()...)
	}

	if o.BodyLimit != nil {
		errs = append(errs, o.BodyLimit.Validate()...)
	}

	if o.Timeout != nil {
		errs = append(errs, o.Timeout.Validate()...)
	}

	if o.Health != nil {
		errs = append(errs, o.Health.Validate()...)
	}

	if o.Metrics != nil {
		errs = append(errs, o.Metrics.Validate()...)
	}

	if o.Pprof != nil {
		errs = append(errs, o.Pprof.Validate()...)
	}

	if o.Auth != nil {
		errs = append(errs, o.Auth.Validate()...)
	}

	if o.Authz != nil {
		errs = append(errs, o.Authz.Validate()...)
	}

	if o.Version != nil {
		errs = append(errs, o.Version.Validate()...)
	}

	if o.Compression != nil {
		errs = append(errs, o.Compression.Validate()...)
	}

	if o.SecurityHeaders != nil {
		errs = append(errs, o.SecurityHeaders.Validate()...)
	}

	if o.RateLimit != nil {
		errs = append(errs, o.RateLimit.Validate()...)
	}

	if o.CircuitBreaker != nil {
		errs = append(errs, o.CircuitBreaker.Validate()...)
	}

	return errs
}

// Complete completes the middleware options with defaults.
func (o *Options) Complete() error {
	// 调用各子选项的 Complete 方法
	if o.Recovery != nil {
		if err := o.Recovery.Complete(); err != nil {
			return err
		}
	}
	if o.RequestID != nil {
		if err := o.RequestID.Complete(); err != nil {
			return err
		}
	}
	if o.Logger != nil {
		if err := o.Logger.Complete(); err != nil {
			return err
		}
	}
	if o.CORS != nil {
		if err := o.CORS.Complete(); err != nil {
			return err
		}
	}
	if o.BodyLimit != nil {
		if err := o.BodyLimit.Complete(); err != nil {
			return err
		}
	}
	if o.Timeout != nil {
		if err := o.Timeout.Complete(); err != nil {
			return err
		}
	}
	if o.Health != nil {
		if err := o.Health.Complete(); err != nil {
			return err
		}
	}
	if o.Metrics != nil {
		if err := o.Metrics.Complete(); err != nil {
			return err
		}
	}
	if o.Pprof != nil {
		if err := o.Pprof.Complete(); err != nil {
			return err
		}
	}
	if o.Auth != nil {
		if err := o.Auth.Complete(); err != nil {
			return err
		}
	}
	if o.Authz != nil {
		if err := o.Authz.Complete(); err != nil {
			return err
		}
	}
	if o.Version != nil {
		if err := o.Version.Complete(); err != nil {
			return err
		}
	}
	if o.Compression != nil {
		if err := o.Compression.Complete(); err != nil {
			return err
		}
	}
	if o.SecurityHeaders != nil {
		if err := o.SecurityHeaders.Complete(); err != nil {
			return err
		}
	}
	if o.RateLimit != nil {
		if err := o.RateLimit.Complete(); err != nil {
			return err
		}
	}
	if o.CircuitBreaker != nil {
		if err := o.CircuitBreaker.Complete(); err != nil {
			return err
		}
	}

	return nil
}

// IsEnabled checks if the specified middleware is enabled.
// 通过检查配置是否为 nil 来判断中间件是否启用。
func (o *Options) IsEnabled(name string) bool {
	switch name {
	case MiddlewareRecovery:
		return o.Recovery != nil
	case MiddlewareRequestID:
		return o.RequestID != nil
	case MiddlewareLogger:
		return o.Logger != nil
	case MiddlewareCORS:
		return o.CORS != nil
	case MiddlewareBodyLimit:
		return o.BodyLimit != nil
	case MiddlewareTimeout:
		return o.Timeout != nil
	case MiddlewareHealth:
		return o.Health != nil
	case MiddlewareMetrics:
		return o.Metrics != nil
	case MiddlewarePprof:
		return o.Pprof != nil
	case MiddlewareAuth:
		return o.Auth != nil
	case MiddlewareAuthz:
		return o.Authz != nil
	case MiddlewareVersion:
		return o.Version != nil && o.Version.Enabled
	case MiddlewareCompression:
		return o.Compression != nil
	case MiddlewareSecurityHeaders:
		return o.SecurityHeaders != nil
	case MiddlewareRateLimit:
		return o.RateLimit != nil
	case MiddlewareCircuitBreaker:
		return o.CircuitBreaker != nil && o.CircuitBreaker.Enabled
	default:
		return false
	}
}

// GetEnabledMiddlewares 返回所有启用的中间件名称列表。
func (o *Options) GetEnabledMiddlewares() []string {
	var enabled []string
	for _, name := range AllMiddlewares {
		if o.IsEnabled(name) {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// DefaultMiddlewareOrder 返回默认的中间件应用顺序。
// 这个顺序保持与原有硬编码顺序一致，确保向后兼容。
func DefaultMiddlewareOrder() []string {
	return []string{
		MiddlewareRecovery,  // 最高优先级，捕获 panic
		MiddlewareRequestID, // 为其他中间件提供 RequestID
		MiddlewareLogger,    // 依赖 RequestID
		MiddlewareMetrics,   // 监控指标收集
		MiddlewareCORS,      // 跨域支持
		MiddlewareTimeout,   // 超时控制
		// 注意：Auth, BodyLimit, RateLimit, CircuitBreaker, SecurityHeaders, Compression 等
		// 中间件需要根据具体业务需求手动添加到 middleware 配置中
	}
}

// GetMiddlewareOrder 返回中间件应用顺序。
// 如果 Middleware 字段为空，返回默认顺序。
func (o *Options) GetMiddlewareOrder() []string {
	if len(o.Middleware) > 0 {
		return o.Middleware
	}
	return DefaultMiddlewareOrder()
}

// ValidateMiddleware 验证 Middleware 配置的有效性。
// 检查：
// 1. Middleware 中的中间件名称是否都是已知的中间件
// 2. 是否有重复的中间件名称
func (o *Options) ValidateMiddleware() []error {
	if len(o.Middleware) == 0 {
		return nil
	}

	var errs []error
	seen := make(map[string]bool)

	for _, name := range o.Middleware {
		// 检查中间件名称是否有效
		valid := false
		for _, known := range AllMiddlewares {
			if name == known {
				valid = true
				break
			}
		}
		if !valid {
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

// GetConfig 获取指定中间件的配置（通用方法）。
// 返回 MiddlewareConfig 接口，调用者需要类型断言获取具体类型。
func (o *Options) GetConfig(name string) MiddlewareConfig {
	switch name {
	case MiddlewareRecovery:
		return o.Recovery
	case MiddlewareRequestID:
		return o.RequestID
	case MiddlewareLogger:
		return o.Logger
	case MiddlewareCORS:
		return o.CORS
	case MiddlewareBodyLimit:
		return o.BodyLimit
	case MiddlewareTimeout:
		return o.Timeout
	case MiddlewareHealth:
		return o.Health
	case MiddlewareMetrics:
		return o.Metrics
	case MiddlewarePprof:
		return o.Pprof
	case MiddlewareAuth:
		return o.Auth
	case MiddlewareAuthz:
		return o.Authz
	case MiddlewareVersion:
		return o.Version
	case MiddlewareCompression:
		return o.Compression
	case MiddlewareSecurityHeaders:
		return o.SecurityHeaders
	case MiddlewareRateLimit:
		return o.RateLimit
	case MiddlewareCircuitBreaker:
		return o.CircuitBreaker
	default:
		return nil
	}
}

// AddFlags adds flags for middleware options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	// 委托给各子选项的 AddFlags 方法
	if o.Recovery != nil {
		o.Recovery.AddFlags(fs, prefixes...)
	}
	if o.RequestID != nil {
		o.RequestID.AddFlags(fs, prefixes...)
	}
	if o.Logger != nil {
		o.Logger.AddFlags(fs, prefixes...)
	}
	if o.CORS != nil {
		o.CORS.AddFlags(fs, prefixes...)
	}
	if o.BodyLimit != nil {
		o.BodyLimit.AddFlags(fs, prefixes...)
	}
	if o.Timeout != nil {
		o.Timeout.AddFlags(fs, prefixes...)
	}
	if o.Health != nil {
		o.Health.AddFlags(fs, prefixes...)
	}
	if o.Metrics != nil {
		o.Metrics.AddFlags(fs, prefixes...)
	}
	if o.Pprof != nil {
		o.Pprof.AddFlags(fs, prefixes...)
	}
	if o.Auth != nil {
		o.Auth.AddFlags(fs, prefixes...)
	}
	if o.Authz != nil {
		o.Authz.AddFlags(fs, prefixes...)
	}
	if o.Version != nil {
		o.Version.AddFlags(fs, prefixes...)
	}
	if o.Compression != nil {
		o.Compression.AddFlags(fs, prefixes...)
	}
	if o.SecurityHeaders != nil {
		o.SecurityHeaders.AddFlags(fs, prefixes...)
	}
	if o.RateLimit != nil {
		o.RateLimit.AddFlags(fs, prefixes...)
	}
	if o.CircuitBreaker != nil {
		o.CircuitBreaker.AddFlags(fs, prefixes...)
	}
}

// WithRecovery configures and enables recovery middleware.
// 注意：onPanic 参数已废弃，应通过 middleware.RecoveryWithOptions() 传入。
func WithRecovery(enableStackTrace bool) Option {
	return func(o *Options) {
		if o.Recovery == nil {
			o.Recovery = NewRecoveryOptions()
		}
		o.Recovery.EnableStackTrace = enableStackTrace
	}
}

// WithoutRecovery disables recovery middleware.
func WithoutRecovery() Option {
	return func(o *Options) { o.Recovery = nil }
}

// WithRequestID enables request ID middleware with custom header.
func WithRequestID(header string) Option {
	return func(o *Options) {
		if o.RequestID == nil {
			o.RequestID = NewRequestIDOptions()
		}
		if header != "" {
			o.RequestID.Header = header
		}
	}
}

// WithoutRequestID disables request ID middleware.
func WithoutRequestID() Option {
	return func(o *Options) { o.RequestID = nil }
}

// WithLogger enables logger middleware.
func WithLogger(skipPaths ...string) Option {
	return func(o *Options) {
		if o.Logger == nil {
			o.Logger = NewLoggerOptions()
		}
		if len(skipPaths) > 0 {
			o.Logger.SkipPaths = skipPaths
		}
	}
}

// WithoutLogger disables logger middleware.
func WithoutLogger() Option {
	return func(o *Options) { o.Logger = nil }
}

// WithCORS enables CORS middleware.
func WithCORS(origins ...string) Option {
	return func(o *Options) {
		if o.CORS == nil {
			o.CORS = NewCORSOptions()
		}
		if len(origins) > 0 {
			o.CORS.AllowOrigins = origins
		}
	}
}

// WithoutCORS disables CORS middleware.
func WithoutCORS() Option {
	return func(o *Options) { o.CORS = nil }
}

// WithTimeout enables timeout middleware.
func WithTimeout(timeout time.Duration, skipPaths ...string) Option {
	return func(o *Options) {
		if o.Timeout == nil {
			o.Timeout = NewTimeoutOptions()
		}
		if timeout > 0 {
			o.Timeout.Timeout = timeout
		}
		if len(skipPaths) > 0 {
			o.Timeout.SkipPaths = skipPaths
		}
	}
}

// WithoutTimeout disables timeout middleware.
func WithoutTimeout() Option {
	return func(o *Options) { o.Timeout = nil }
}

// WithHealth enables health check endpoints.
func WithHealth(path, livenessPath, readinessPath string) Option {
	return func(o *Options) {
		if o.Health == nil {
			o.Health = NewHealthOptions()
		}
		if path != "" {
			o.Health.Path = path
		}
		if livenessPath != "" {
			o.Health.LivenessPath = livenessPath
		}
		if readinessPath != "" {
			o.Health.ReadinessPath = readinessPath
		}
	}
}

// WithoutHealth disables health check endpoints.
func WithoutHealth() Option {
	return func(o *Options) { o.Health = nil }
}

// WithMetrics enables metrics endpoint.
func WithMetrics(path, namespace, subsystem string) Option {
	return func(o *Options) {
		if o.Metrics == nil {
			o.Metrics = NewMetricsOptions()
		}
		if path != "" {
			o.Metrics.Path = path
		}
		if namespace != "" {
			o.Metrics.Namespace = namespace
		}
		if subsystem != "" {
			o.Metrics.Subsystem = subsystem
		}
	}
}

// WithoutMetrics disables metrics endpoint.
func WithoutMetrics() Option {
	return func(o *Options) { o.Metrics = nil }
}

// WithPprof enables pprof endpoints.
func WithPprof(prefix string) Option {
	return func(o *Options) {
		if o.Pprof == nil {
			o.Pprof = NewPprofOptions()
		}
		if prefix != "" {
			o.Pprof.Prefix = prefix
		}
	}
}

// WithoutPprof disables pprof endpoints.
func WithoutPprof() Option {
	return func(o *Options) { o.Pprof = nil }
}

// WithAuth enables authentication middleware.
func WithAuth(tokenLookup, authScheme string, skipPaths ...string) Option {
	return func(o *Options) {
		if o.Auth == nil {
			o.Auth = NewAuthOptions()
		}
		if tokenLookup != "" {
			o.Auth.TokenLookup = tokenLookup
		}
		if authScheme != "" {
			o.Auth.AuthScheme = authScheme
		}
		if len(skipPaths) > 0 {
			o.Auth.SkipPaths = skipPaths
		}
	}
}

// WithoutAuth disables authentication middleware.
func WithoutAuth() Option {
	return func(o *Options) { o.Auth = nil }
}

// WithAuthz enables authorization middleware.
func WithAuthz() Option {
	return func(o *Options) {
		if o.Authz == nil {
			o.Authz = NewAuthzOptions()
		}
	}
}

// WithoutAuthz disables authorization middleware.
func WithoutAuthz() Option {
	return func(o *Options) { o.Authz = nil }
}

// WithVersion enables version endpoint.
func WithVersion(path string, hideDetails bool) Option {
	return func(o *Options) {
		if o.Version == nil {
			o.Version = NewVersionOptions()
		}
		if path != "" {
			o.Version.Path = path
		}
		o.Version.HideDetails = hideDetails
	}
}

// WithoutVersion disables version endpoint.
func WithoutVersion() Option {
	return func(o *Options) { o.Version = nil }
}

// WithBodyLimit enables body limit middleware.
func WithBodyLimit(maxSize int64, skipPaths ...string) Option {
	return func(o *Options) {
		if o.BodyLimit == nil {
			o.BodyLimit = NewBodyLimitOptions()
		}
		if maxSize > 0 {
			o.BodyLimit.MaxSize = maxSize
		}
		if len(skipPaths) > 0 {
			o.BodyLimit.SkipPaths = skipPaths
		}
	}
}

// WithoutBodyLimit disables body limit middleware.
func WithoutBodyLimit() Option {
	return func(o *Options) { o.BodyLimit = nil }
}

// WithCompression enables compression middleware.
func WithCompression(level int, minSize int, types ...string) Option {
	return func(o *Options) {
		if o.Compression == nil {
			o.Compression = NewCompressionOptions()
		}
		if level >= -1 && level <= 9 {
			o.Compression.Level = level
		}
		if minSize >= 0 {
			o.Compression.MinSize = minSize
		}
		if len(types) > 0 {
			o.Compression.Types = types
		}
	}
}

// WithoutCompression disables compression middleware.
func WithoutCompression() Option {
	return func(o *Options) { o.Compression = nil }
}
