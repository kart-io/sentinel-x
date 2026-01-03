// Package middleware provides middleware configuration options.
package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/spf13/pflag"
)

// PathMatcher contains common path matching configuration.
type PathMatcher struct {
	SkipPaths        []string
	SkipPathPrefixes []string
}

// 中间件名称常量。
const (
	MiddlewareRecovery  = "recovery"
	MiddlewareRequestID = "request-id"
	MiddlewareLogger    = "logger"
	MiddlewareCORS      = "cors"
	MiddlewareTimeout   = "timeout"
	MiddlewareHealth    = "health"
	MiddlewareMetrics   = "metrics"
	MiddlewarePprof     = "pprof"
	MiddlewareAuth      = "auth"
	MiddlewareAuthz     = "authz"
)

// AllMiddlewares 所有支持的中间件名称。
var AllMiddlewares = []string{
	MiddlewareRecovery,
	MiddlewareRequestID,
	MiddlewareLogger,
	MiddlewareCORS,
	MiddlewareTimeout,
	MiddlewareHealth,
	MiddlewareMetrics,
	MiddlewarePprof,
	MiddlewareAuth,
	MiddlewareAuthz,
}

// DefaultEnabledMiddlewares 默认启用的中间件列表。
var DefaultEnabledMiddlewares = []string{
	MiddlewareRecovery,
	MiddlewareRequestID,
	MiddlewareLogger,
	MiddlewareHealth,
	MiddlewareMetrics,
}

// Options contains all middleware configuration.
type Options struct {
	// Enabled 指定启用的中间件列表。
	// 支持的值: recovery, request-id, logger, cors, timeout, health, metrics, pprof, auth, authz
	Enabled []string `json:"enabled" mapstructure:"enabled"`

	// Recovery 配置。
	Recovery *RecoveryOptions `json:"recovery" mapstructure:"recovery"`

	// RequestID 配置。
	RequestID *RequestIDOptions `json:"request-id" mapstructure:"request-id"`

	// Logger 配置。
	Logger *LoggerOptions `json:"logger" mapstructure:"logger"`

	// CORS 配置。
	CORS *CORSOptions `json:"cors" mapstructure:"cors"`

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

	// enabledSet 内部使用的启用中间件集合，用于快速查找。
	enabledSet map[string]bool
}

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions creates default middleware options.
func NewOptions() *Options {
	return &Options{
		Enabled:   append([]string{}, DefaultEnabledMiddlewares...),
		Recovery:  NewRecoveryOptions(),
		RequestID: NewRequestIDOptions(),
		Logger:    NewLoggerOptions(),
		CORS:      NewCORSOptions(),
		Timeout:   NewTimeoutOptions(),
		Health:    NewHealthOptions(),
		Metrics:   NewMetricsOptions(),
		Pprof:     NewPprofOptions(),
		Auth:      NewAuthOptions(),
		Authz:     NewAuthzOptions(),
	}
}

// Validate validates the middleware options.
func (o *Options) Validate() error {
	var errs []error

	// 确保所有子选项都已初始化
	o.ensureDefaults()

	// 构建启用集合
	o.buildEnabledSet()

	// 验证 Enabled 数组中的名称是否有效
	validNames := make(map[string]bool)
	for _, name := range AllMiddlewares {
		validNames[name] = true
	}
	for _, name := range o.Enabled {
		if !validNames[name] {
			errs = append(errs, fmt.Errorf("invalid middleware name: %q, valid names: %v", name, AllMiddlewares))
		}
	}

	// 验证启用的中间件配置
	if o.IsEnabled(MiddlewareTimeout) {
		if err := o.Timeout.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewareCORS) {
		if err := o.CORS.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewareAuth) {
		if err := o.Auth.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewareAuthz) {
		if err := o.Authz.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewareHealth) {
		if err := o.Health.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewareMetrics) {
		if err := o.Metrics.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewarePprof) {
		if err := o.Pprof.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewareRequestID) {
		if err := o.RequestID.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewareRecovery) {
		if err := o.Recovery.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if o.IsEnabled(MiddlewareLogger) {
		if err := o.Logger.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("middleware validation errors: %v", errs)
	}
	return nil
}

// Complete completes the middleware options with defaults.
func (o *Options) Complete() error {
	// 确保所有子选项都已初始化
	o.ensureDefaults()

	// 构建启用集合
	o.buildEnabledSet()

	// 设置 Logger 默认输出
	if o.Logger.Output == nil {
		o.Logger.Output = log.Printf
	}

	// 调用各子选项的 Complete 方法
	if err := o.Recovery.Complete(); err != nil {
		return err
	}
	if err := o.RequestID.Complete(); err != nil {
		return err
	}
	if err := o.Logger.Complete(); err != nil {
		return err
	}
	if err := o.CORS.Complete(); err != nil {
		return err
	}
	if err := o.Timeout.Complete(); err != nil {
		return err
	}
	if err := o.Health.Complete(); err != nil {
		return err
	}
	if err := o.Metrics.Complete(); err != nil {
		return err
	}
	if err := o.Pprof.Complete(); err != nil {
		return err
	}
	if err := o.Auth.Complete(); err != nil {
		return err
	}
	if err := o.Authz.Complete(); err != nil {
		return err
	}

	return nil
}

// buildEnabledSet 构建启用的中间件集合。
func (o *Options) buildEnabledSet() {
	o.enabledSet = make(map[string]bool)
	for _, name := range o.Enabled {
		o.enabledSet[name] = true
	}
}

// IsEnabled 检查指定的中间件是否启用。
func (o *Options) IsEnabled(name string) bool {
	if o.enabledSet == nil {
		o.buildEnabledSet()
	}
	return o.enabledSet[name]
}

// Enable 启用指定的中间件。
func (o *Options) Enable(names ...string) {
	for _, name := range names {
		if !o.IsEnabled(name) {
			o.Enabled = append(o.Enabled, name)
		}
	}
	o.buildEnabledSet()
}

// Disable 禁用指定的中间件。
func (o *Options) Disable(names ...string) {
	disableSet := make(map[string]bool)
	for _, name := range names {
		disableSet[name] = true
	}

	var newEnabled []string
	for _, name := range o.Enabled {
		if !disableSet[name] {
			newEnabled = append(newEnabled, name)
		}
	}
	o.Enabled = newEnabled
	o.buildEnabledSet()
}

// GetEnabledMiddlewares 返回所有启用的中间件名称列表。
func (o *Options) GetEnabledMiddlewares() []string {
	return append([]string{}, o.Enabled...)
}

// ensureDefaults ensures all sub-options are initialized.
func (o *Options) ensureDefaults() {
	if o.Recovery == nil {
		o.Recovery = NewRecoveryOptions()
	}
	if o.RequestID == nil {
		o.RequestID = NewRequestIDOptions()
	}
	if o.Logger == nil {
		o.Logger = NewLoggerOptions()
	}
	if o.CORS == nil {
		o.CORS = NewCORSOptions()
	}
	if o.Timeout == nil {
		o.Timeout = NewTimeoutOptions()
	}
	if o.Health == nil {
		o.Health = NewHealthOptions()
	}
	if o.Metrics == nil {
		o.Metrics = NewMetricsOptions()
	}
	if o.Pprof == nil {
		o.Pprof = NewPprofOptions()
	}
	if o.Auth == nil {
		o.Auth = NewAuthOptions()
	}
	if o.Authz == nil {
		o.Authz = NewAuthzOptions()
	}
}

// AddFlags adds flags for middleware options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	// 确保所有子选项都已初始化
	o.ensureDefaults()

	// Enabled list flag
	fs.StringSliceVar(&o.Enabled, "middleware.enabled", o.Enabled,
		"List of enabled middlewares. "+
			"Valid values: recovery, request-id, logger, cors, timeout, health, metrics, pprof, auth, authz")

	// 委托给各子选项的 AddFlags 方法
	o.Recovery.AddFlags(fs)
	o.RequestID.AddFlags(fs)
	o.Logger.AddFlags(fs)
	o.CORS.AddFlags(fs)
	o.Timeout.AddFlags(fs)
	o.Health.AddFlags(fs)
	o.Metrics.AddFlags(fs)
	o.Pprof.AddFlags(fs)
	o.Auth.AddFlags(fs)
	o.Authz.AddFlags(fs)
}

// WithRecovery configures and enables recovery middleware.
func WithRecovery(enableStackTrace bool, onPanic func(ctx transport.Context, err interface{}, stack []byte)) Option {
	return func(o *Options) {
		o.Enable(MiddlewareRecovery)
		o.Recovery.EnableStackTrace = enableStackTrace
		if onPanic != nil {
			o.Recovery.OnPanic = onPanic
		}
	}
}

// WithoutRecovery disables recovery middleware.
func WithoutRecovery() Option {
	return func(o *Options) { o.Disable(MiddlewareRecovery) }
}

// WithRequestID enables request ID middleware with custom header.
func WithRequestID(header string) Option {
	return func(o *Options) {
		o.Enable(MiddlewareRequestID)
		if header != "" {
			o.RequestID.Header = header
		}
	}
}

// WithoutRequestID disables request ID middleware.
func WithoutRequestID() Option {
	return func(o *Options) { o.Disable(MiddlewareRequestID) }
}

// WithLogger enables logger middleware.
func WithLogger(skipPaths ...string) Option {
	return func(o *Options) {
		o.Enable(MiddlewareLogger)
		if len(skipPaths) > 0 {
			o.Logger.SkipPaths = skipPaths
		}
	}
}

// WithoutLogger disables logger middleware.
func WithoutLogger() Option {
	return func(o *Options) { o.Disable(MiddlewareLogger) }
}

// WithCORS enables CORS middleware.
func WithCORS(origins ...string) Option {
	return func(o *Options) {
		o.Enable(MiddlewareCORS)
		if len(origins) > 0 {
			o.CORS.AllowOrigins = origins
		}
	}
}

// WithoutCORS disables CORS middleware.
func WithoutCORS() Option {
	return func(o *Options) { o.Disable(MiddlewareCORS) }
}

// WithTimeout enables timeout middleware.
func WithTimeout(timeout time.Duration, skipPaths ...string) Option {
	return func(o *Options) {
		o.Enable(MiddlewareTimeout)
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
	return func(o *Options) { o.Disable(MiddlewareTimeout) }
}

// WithHealth enables health check endpoints.
func WithHealth(path, livenessPath, readinessPath string) Option {
	return func(o *Options) {
		o.Enable(MiddlewareHealth)
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
	return func(o *Options) { o.Disable(MiddlewareHealth) }
}

// WithMetrics enables metrics endpoint.
func WithMetrics(path, namespace, subsystem string) Option {
	return func(o *Options) {
		o.Enable(MiddlewareMetrics)
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
	return func(o *Options) { o.Disable(MiddlewareMetrics) }
}

// WithPprof enables pprof endpoints.
func WithPprof(prefix string) Option {
	return func(o *Options) {
		o.Enable(MiddlewarePprof)
		if prefix != "" {
			o.Pprof.Prefix = prefix
		}
	}
}

// WithoutPprof disables pprof endpoints.
func WithoutPprof() Option {
	return func(o *Options) { o.Disable(MiddlewarePprof) }
}

// WithAuth enables authentication middleware.
func WithAuth(tokenLookup, authScheme string, skipPaths ...string) Option {
	return func(o *Options) {
		o.Enable(MiddlewareAuth)
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
	return func(o *Options) { o.Disable(MiddlewareAuth) }
}

// WithAuthz enables authorization middleware.
func WithAuthz() Option {
	return func(o *Options) { o.Enable(MiddlewareAuthz) }
}

// WithoutAuthz disables authorization middleware.
func WithoutAuthz() Option {
	return func(o *Options) { o.Disable(MiddlewareAuthz) }
}

// WithMiddlewares enables specific middlewares.
func WithMiddlewares(names ...string) Option {
	return func(o *Options) {
		o.Enabled = names
		o.buildEnabledSet()
	}
}
