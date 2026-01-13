package middleware

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/logger"
	configpkg "github.com/kart-io/sentinel-x/pkg/infra/config"
	options "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// ReloadableMiddleware 封装中间件配置并提供热重载能力。
// 维护线程安全的配置访问，支持运行时配置变更无需重启服务。
//
// 支持热重载的配置：
//   - CORS 设置（origins, methods, headers, credentials, max age）
//   - Timeout 持续时间和跳过路径
//   - Request ID 头部
//   - Logger 跳过路径
//   - Recovery 堆栈跟踪设置
//
// 注意：某些中间件配置无法热重载，需要服务重启或中间件链重建（如启用/禁用标志）。
type ReloadableMiddleware struct {
	opts *Options
	mu   sync.RWMutex
	// 需要配置变更通知的组件回调
	onTimeoutChange func(time.Duration, []string) error
	onCORSChange    func(*CORSOptions) error
}

// NewReloadableMiddleware 创建新的可重载中间件管理器。
func NewReloadableMiddleware(opts *Options) *ReloadableMiddleware {
	return &ReloadableMiddleware{
		opts: opts,
	}
}

// OnConfigChange 实现 config.Reloadable 接口。
// 原子性地验证并应用新的中间件配置。
func (rm *ReloadableMiddleware) OnConfigChange(newConfig interface{}) error {
	newOpts, ok := newConfig.(*Options)
	if !ok {
		return fmt.Errorf("invalid config type: expected *middleware.Options, got %T", newConfig)
	}

	// 验证新配置
	if errs := newOpts.Validate(); len(errs) > 0 {
		return fmt.Errorf("invalid middleware configuration: %w", errors.Join(errs...))
	}

	// 获取写锁
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 记录变更用于日志
	changes := []string{}

	// 更新 Timeout 配置
	changes = append(changes, rm.updateTimeoutConfig(newOpts)...)

	// 更新 CORS 配置
	changes = append(changes, rm.updateCORSConfig(newOpts)...)

	// 更新 RequestID 配置
	changes = append(changes, rm.updateRequestIDConfig(newOpts)...)

	// 更新 Logger 配置
	changes = append(changes, rm.updateLoggerConfig(newOpts)...)

	// 更新 Recovery 配置
	changes = append(changes, rm.updateRecoveryConfig(newOpts)...)

	// 更新 Health 配置
	changes = append(changes, rm.updateHealthConfig(newOpts)...)

	// 更新 Metrics 配置
	changes = append(changes, rm.updateMetricsConfig(newOpts)...)

	// 更新 Pprof 配置
	changes = append(changes, rm.updatePprofConfig(newOpts)...)

	if len(changes) > 0 {
		logger.Infof("Middleware configuration reloaded: %v", changes)
	} else {
		logger.Debug("Middleware configuration unchanged")
	}

	return nil
}

// updateTimeoutConfig 更新 Timeout 配置。
func (rm *ReloadableMiddleware) updateTimeoutConfig(newOpts *Options) []string {
	var changes []string

	if !rm.opts.IsEnabled(options.MiddlewareTimeout) {
		return changes
	}

	oldCfg, oldOk := options.GetConfigTyped[*options.TimeoutOptions](rm.opts, options.MiddlewareTimeout)
	newCfg, newOk := options.GetConfigTyped[*options.TimeoutOptions](newOpts, options.MiddlewareTimeout)

	if !oldOk || !newOk {
		return changes
	}

	if oldCfg.Timeout != newCfg.Timeout {
		changes = append(changes, fmt.Sprintf("timeout: %v -> %v", oldCfg.Timeout, newCfg.Timeout))

		// 调用回调（如果已注册）
		if rm.onTimeoutChange != nil {
			if err := rm.onTimeoutChange(newCfg.Timeout, newCfg.SkipPaths); err != nil {
				logger.Warnw("failed to apply timeout change", "error", err.Error())
			}
		}
	}

	// 更新配置
	rm.opts.SetConfig(options.MiddlewareTimeout, newCfg)

	return changes
}

// updateCORSConfig 更新 CORS 配置。
func (rm *ReloadableMiddleware) updateCORSConfig(newOpts *Options) []string {
	var changes []string

	if !rm.opts.IsEnabled(options.MiddlewareCORS) {
		return changes
	}

	oldCfg, oldOk := options.GetConfigTyped[*options.CORSOptions](rm.opts, options.MiddlewareCORS)
	newCfg, newOk := options.GetConfigTyped[*options.CORSOptions](newOpts, options.MiddlewareCORS)

	if !oldOk || !newOk {
		return changes
	}

	corsChanged := false

	if !stringSlicesEqual(oldCfg.AllowOrigins, newCfg.AllowOrigins) {
		changes = append(changes, "cors.allow-origins")
		corsChanged = true
	}
	if !stringSlicesEqual(oldCfg.AllowMethods, newCfg.AllowMethods) {
		changes = append(changes, "cors.allow-methods")
		corsChanged = true
	}
	if !stringSlicesEqual(oldCfg.AllowHeaders, newCfg.AllowHeaders) {
		changes = append(changes, "cors.allow-headers")
		corsChanged = true
	}
	if oldCfg.AllowCredentials != newCfg.AllowCredentials {
		changes = append(changes, "cors.allow-credentials")
		corsChanged = true
	}
	if oldCfg.MaxAge != newCfg.MaxAge {
		changes = append(changes, "cors.max-age")
		corsChanged = true
	}

	if corsChanged {
		// 调用回调（如果已注册）
		if rm.onCORSChange != nil {
			if err := rm.onCORSChange(newCfg); err != nil {
				logger.Warnw("failed to apply CORS change", "error", err.Error())
			}
		}

		rm.opts.SetConfig(options.MiddlewareCORS, newCfg)
	}

	return changes
}

// updateRequestIDConfig 更新 RequestID 配置。
func (rm *ReloadableMiddleware) updateRequestIDConfig(newOpts *Options) []string {
	var changes []string

	oldCfg, oldOk := options.GetConfigTyped[*options.RequestIDOptions](rm.opts, options.MiddlewareRequestID)
	newCfg, newOk := options.GetConfigTyped[*options.RequestIDOptions](newOpts, options.MiddlewareRequestID)

	if !oldOk || !newOk {
		return changes
	}

	if oldCfg.Header != newCfg.Header {
		changes = append(changes, fmt.Sprintf("request-id.header: %s -> %s", oldCfg.Header, newCfg.Header))
		rm.opts.SetConfig(options.MiddlewareRequestID, newCfg)
	}

	return changes
}

// updateLoggerConfig 更新 Logger 配置。
func (rm *ReloadableMiddleware) updateLoggerConfig(newOpts *Options) []string {
	var changes []string

	oldCfg, oldOk := options.GetConfigTyped[*options.LoggerOptions](rm.opts, options.MiddlewareLogger)
	newCfg, newOk := options.GetConfigTyped[*options.LoggerOptions](newOpts, options.MiddlewareLogger)

	if !oldOk || !newOk {
		return changes
	}

	if !stringSlicesEqual(oldCfg.SkipPaths, newCfg.SkipPaths) {
		changes = append(changes, "logger.skip-paths")
	}

	if oldCfg.UseStructuredLogger != newCfg.UseStructuredLogger {
		changes = append(changes, fmt.Sprintf("logger.use-structured-logger: %v -> %v",
			oldCfg.UseStructuredLogger, newCfg.UseStructuredLogger))
	}

	if len(changes) > 0 {
		rm.opts.SetConfig(options.MiddlewareLogger, newCfg)
	}

	return changes
}

// updateRecoveryConfig 更新 Recovery 配置。
func (rm *ReloadableMiddleware) updateRecoveryConfig(newOpts *Options) []string {
	var changes []string

	oldCfg, oldOk := options.GetConfigTyped[*options.RecoveryOptions](rm.opts, options.MiddlewareRecovery)
	newCfg, newOk := options.GetConfigTyped[*options.RecoveryOptions](newOpts, options.MiddlewareRecovery)

	if !oldOk || !newOk {
		return changes
	}

	if oldCfg.EnableStackTrace != newCfg.EnableStackTrace {
		changes = append(changes, fmt.Sprintf("recovery.enable-stack-trace: %v -> %v",
			oldCfg.EnableStackTrace, newCfg.EnableStackTrace))
		rm.opts.SetConfig(options.MiddlewareRecovery, newCfg)
	}

	return changes
}

// updateHealthConfig 更新 Health 配置。
func (rm *ReloadableMiddleware) updateHealthConfig(newOpts *Options) []string {
	var changes []string

	oldCfg, oldOk := options.GetConfigTyped[*options.HealthOptions](rm.opts, options.MiddlewareHealth)
	newCfg, newOk := options.GetConfigTyped[*options.HealthOptions](newOpts, options.MiddlewareHealth)

	if !oldOk || !newOk {
		return changes
	}

	if oldCfg.Path != newCfg.Path {
		changes = append(changes, fmt.Sprintf("health.path: %s -> %s", oldCfg.Path, newCfg.Path))
	}
	if oldCfg.LivenessPath != newCfg.LivenessPath {
		changes = append(changes, "health.liveness-path")
	}
	if oldCfg.ReadinessPath != newCfg.ReadinessPath {
		changes = append(changes, "health.readiness-path")
	}

	if len(changes) > 0 {
		rm.opts.SetConfig(options.MiddlewareHealth, newCfg)
	}

	return changes
}

// updateMetricsConfig 更新 Metrics 配置。
func (rm *ReloadableMiddleware) updateMetricsConfig(newOpts *Options) []string {
	var changes []string

	oldCfg, oldOk := options.GetConfigTyped[*options.MetricsOptions](rm.opts, options.MiddlewareMetrics)
	newCfg, newOk := options.GetConfigTyped[*options.MetricsOptions](newOpts, options.MiddlewareMetrics)

	if !oldOk || !newOk {
		return changes
	}

	if oldCfg.Path != newCfg.Path {
		changes = append(changes, "metrics.path")
	}
	if oldCfg.Namespace != newCfg.Namespace {
		changes = append(changes, "metrics.namespace")
	}
	if oldCfg.Subsystem != newCfg.Subsystem {
		changes = append(changes, "metrics.subsystem")
	}

	if len(changes) > 0 {
		rm.opts.SetConfig(options.MiddlewareMetrics, newCfg)
	}

	return changes
}

// updatePprofConfig 更新 Pprof 配置。
func (rm *ReloadableMiddleware) updatePprofConfig(newOpts *Options) []string {
	var changes []string

	oldCfg, oldOk := options.GetConfigTyped[*options.PprofOptions](rm.opts, options.MiddlewarePprof)
	newCfg, newOk := options.GetConfigTyped[*options.PprofOptions](newOpts, options.MiddlewarePprof)

	if !oldOk || !newOk {
		return changes
	}

	if oldCfg.BlockProfileRate != newCfg.BlockProfileRate {
		changes = append(changes, "pprof.block-profile-rate")
	}
	if oldCfg.MutexProfileFraction != newCfg.MutexProfileFraction {
		changes = append(changes, "pprof.mutex-profile-fraction")
	}

	if len(changes) > 0 {
		rm.opts.SetConfig(options.MiddlewarePprof, newCfg)
	}

	return changes
}

// GetOptions 返回当前中间件配置的副本。
// 这是线程安全的读取操作。
func (rm *ReloadableMiddleware) GetOptions() *Options {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// 创建新的 Options 并复制配置
	newOpts := &Options{}

	// 复制所有配置到新实例
	for _, name := range rm.opts.ListConfigs() {
		cfg := rm.opts.GetConfig(name)
		if cfg != nil {
			newOpts.SetConfig(name, cfg)
		}
	}

	return newOpts
}

// SetTimeoutChangeCallback 注册超时配置变更时调用的回调。
// 允许实际的中间件实现更新其行为。
func (rm *ReloadableMiddleware) SetTimeoutChangeCallback(fn func(time.Duration, []string) error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onTimeoutChange = fn
}

// SetCORSChangeCallback 注册 CORS 配置变更时调用的回调。
// 允许实际的中间件实现更新其行为。
func (rm *ReloadableMiddleware) SetCORSChangeCallback(fn func(*CORSOptions) error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onCORSChange = fn
}

// RegisterWithWatcher 将可重载中间件注册到配置监视器。
// handlerID 应在所有已注册处理器中唯一。
func (rm *ReloadableMiddleware) RegisterWithWatcher(watcher *configpkg.Watcher, handlerID, configKey string) {
	target := NewOptions()
	subscriber := configpkg.NewReloadableSubscriber(rm, configKey, target)
	watcher.Subscribe(handlerID, subscriber.Handler())
}

// stringSlicesEqual 比较两个字符串切片是否相等。
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
