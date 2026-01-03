package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/logger"
	configpkg "github.com/kart-io/sentinel-x/pkg/infra/config"
)

// ReloadableMiddleware wraps middleware options with hot reload capability.
// It maintains thread-safe access to middleware configuration and can apply
// configuration changes at runtime without service restart.
//
// Supported hot-reloadable configurations:
//   - CORS settings (origins, methods, headers, credentials, max age)
//   - Timeout duration and skip paths
//   - Request ID header
//   - Logger skip paths
//   - Recovery stack trace settings
//
// Note: Some middleware configurations cannot be hot-reloaded as they require
// server restart or middleware chain reconstruction (e.g., enable/disable flags).
type ReloadableMiddleware struct {
	opts *Options
	mu   sync.RWMutex
	// Callbacks for components that need notification of config changes
	onTimeoutChange func(time.Duration, []string) error
	onCORSChange    func(*CORSOptions) error
}

// NewReloadableMiddleware creates a new reloadable middleware manager.
func NewReloadableMiddleware(opts *Options) *ReloadableMiddleware {
	return &ReloadableMiddleware{
		opts: opts,
	}
}

// OnConfigChange implements the config.Reloadable interface.
// It validates and applies new middleware configuration atomically.
func (rm *ReloadableMiddleware) OnConfigChange(newConfig interface{}) error {
	newOpts, ok := newConfig.(*Options)
	if !ok {
		return fmt.Errorf("invalid config type: expected *middleware.Options, got %T", newConfig)
	}

	// Validate new configuration
	if err := newOpts.Validate(); err != nil {
		return fmt.Errorf("invalid middleware configuration: %w", err)
	}

	// Acquire write lock
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Track what changed for logging
	changes := []string{}

	// Update timeout configuration if changed
	if rm.opts.IsEnabled(MiddlewareTimeout) {
		if rm.opts.Timeout.Timeout != newOpts.Timeout.Timeout {
			changes = append(changes, fmt.Sprintf("timeout: %v -> %v",
				rm.opts.Timeout.Timeout, newOpts.Timeout.Timeout))

			// Call callback if registered
			if rm.onTimeoutChange != nil {
				if err := rm.onTimeoutChange(newOpts.Timeout.Timeout, newOpts.Timeout.SkipPaths); err != nil {
					return fmt.Errorf("failed to apply timeout change: %w", err)
				}
			}
		}

		rm.opts.Timeout.Timeout = newOpts.Timeout.Timeout
		rm.opts.Timeout.SkipPaths = append([]string(nil), newOpts.Timeout.SkipPaths...)
	}

	// Update CORS configuration if changed
	if rm.opts.IsEnabled(MiddlewareCORS) {
		corsChanged := false

		if !stringSlicesEqual(rm.opts.CORS.AllowOrigins, newOpts.CORS.AllowOrigins) {
			changes = append(changes, "cors.allow-origins")
			corsChanged = true
		}
		if !stringSlicesEqual(rm.opts.CORS.AllowMethods, newOpts.CORS.AllowMethods) {
			changes = append(changes, "cors.allow-methods")
			corsChanged = true
		}
		if !stringSlicesEqual(rm.opts.CORS.AllowHeaders, newOpts.CORS.AllowHeaders) {
			changes = append(changes, "cors.allow-headers")
			corsChanged = true
		}
		if rm.opts.CORS.AllowCredentials != newOpts.CORS.AllowCredentials {
			changes = append(changes, "cors.allow-credentials")
			corsChanged = true
		}
		if rm.opts.CORS.MaxAge != newOpts.CORS.MaxAge {
			changes = append(changes, "cors.max-age")
			corsChanged = true
		}

		if corsChanged {
			// Call callback if registered
			if rm.onCORSChange != nil {
				if err := rm.onCORSChange(newOpts.CORS); err != nil {
					return fmt.Errorf("failed to apply CORS change: %w", err)
				}
			}

			rm.opts.CORS = newOpts.CORS
		}
	}

	// Update Request ID header
	if rm.opts.RequestID.Header != newOpts.RequestID.Header {
		changes = append(changes, fmt.Sprintf("request-id.header: %s -> %s",
			rm.opts.RequestID.Header, newOpts.RequestID.Header))
		rm.opts.RequestID.Header = newOpts.RequestID.Header
	}

	// Update Logger skip paths
	if !stringSlicesEqual(rm.opts.Logger.SkipPaths, newOpts.Logger.SkipPaths) {
		changes = append(changes, "logger.skip-paths")
		rm.opts.Logger.SkipPaths = append([]string(nil), newOpts.Logger.SkipPaths...)
	}

	// Update Logger structured setting
	if rm.opts.Logger.UseStructuredLogger != newOpts.Logger.UseStructuredLogger {
		changes = append(changes, fmt.Sprintf("logger.use-structured-logger: %v -> %v",
			rm.opts.Logger.UseStructuredLogger, newOpts.Logger.UseStructuredLogger))
		rm.opts.Logger.UseStructuredLogger = newOpts.Logger.UseStructuredLogger
	}

	// Update Recovery stack trace setting
	if rm.opts.Recovery.EnableStackTrace != newOpts.Recovery.EnableStackTrace {
		changes = append(changes, fmt.Sprintf("recovery.enable-stack-trace: %v -> %v",
			rm.opts.Recovery.EnableStackTrace, newOpts.Recovery.EnableStackTrace))
		rm.opts.Recovery.EnableStackTrace = newOpts.Recovery.EnableStackTrace
	}

	// Update Health paths
	if rm.opts.Health.Path != newOpts.Health.Path {
		changes = append(changes, fmt.Sprintf("health.path: %s -> %s",
			rm.opts.Health.Path, newOpts.Health.Path))
		rm.opts.Health.Path = newOpts.Health.Path
	}
	if rm.opts.Health.LivenessPath != newOpts.Health.LivenessPath {
		changes = append(changes, "health.liveness-path")
		rm.opts.Health.LivenessPath = newOpts.Health.LivenessPath
	}
	if rm.opts.Health.ReadinessPath != newOpts.Health.ReadinessPath {
		changes = append(changes, "health.readiness-path")
		rm.opts.Health.ReadinessPath = newOpts.Health.ReadinessPath
	}

	// Update Metrics configuration
	if rm.opts.Metrics.Path != newOpts.Metrics.Path {
		changes = append(changes, "metrics.path")
		rm.opts.Metrics.Path = newOpts.Metrics.Path
	}
	if rm.opts.Metrics.Namespace != newOpts.Metrics.Namespace {
		changes = append(changes, "metrics.namespace")
		rm.opts.Metrics.Namespace = newOpts.Metrics.Namespace
	}
	if rm.opts.Metrics.Subsystem != newOpts.Metrics.Subsystem {
		changes = append(changes, "metrics.subsystem")
		rm.opts.Metrics.Subsystem = newOpts.Metrics.Subsystem
	}

	// Update Pprof configuration
	if rm.opts.Pprof.BlockProfileRate != newOpts.Pprof.BlockProfileRate {
		changes = append(changes, "pprof.block-profile-rate")
		rm.opts.Pprof.BlockProfileRate = newOpts.Pprof.BlockProfileRate
	}
	if rm.opts.Pprof.MutexProfileFraction != newOpts.Pprof.MutexProfileFraction {
		changes = append(changes, "pprof.mutex-profile-fraction")
		rm.opts.Pprof.MutexProfileFraction = newOpts.Pprof.MutexProfileFraction
	}

	if len(changes) > 0 {
		logger.Infof("Middleware configuration reloaded: %v", changes)
	} else {
		logger.Debug("Middleware configuration unchanged")
	}

	return nil
}

// GetOptions returns a copy of the current middleware options.
// This is thread-safe for reading.
func (rm *ReloadableMiddleware) GetOptions() *Options {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return a deep copy to prevent external modifications
	opts := &Options{
		Enabled: append([]string(nil), rm.opts.Enabled...),
		Recovery: &RecoveryOptions{
			EnableStackTrace: rm.opts.Recovery.EnableStackTrace,
			OnPanic:          rm.opts.Recovery.OnPanic,
		},
		RequestID: &RequestIDOptions{
			Header:    rm.opts.RequestID.Header,
			Generator: rm.opts.RequestID.Generator,
		},
		Logger: &LoggerOptions{
			SkipPaths:           append([]string(nil), rm.opts.Logger.SkipPaths...),
			UseStructuredLogger: rm.opts.Logger.UseStructuredLogger,
			Output:              rm.opts.Logger.Output,
		},
		CORS: &CORSOptions{
			AllowOrigins:     append([]string(nil), rm.opts.CORS.AllowOrigins...),
			AllowMethods:     append([]string(nil), rm.opts.CORS.AllowMethods...),
			AllowHeaders:     append([]string(nil), rm.opts.CORS.AllowHeaders...),
			ExposeHeaders:    append([]string(nil), rm.opts.CORS.ExposeHeaders...),
			AllowCredentials: rm.opts.CORS.AllowCredentials,
			MaxAge:           rm.opts.CORS.MaxAge,
		},
		Timeout: &TimeoutOptions{
			Timeout:   rm.opts.Timeout.Timeout,
			SkipPaths: append([]string(nil), rm.opts.Timeout.SkipPaths...),
		},
		Health: &HealthOptions{
			Path:          rm.opts.Health.Path,
			LivenessPath:  rm.opts.Health.LivenessPath,
			ReadinessPath: rm.opts.Health.ReadinessPath,
			Checker:       rm.opts.Health.Checker,
		},
		Metrics: &MetricsOptions{
			Path:      rm.opts.Metrics.Path,
			Namespace: rm.opts.Metrics.Namespace,
			Subsystem: rm.opts.Metrics.Subsystem,
		},
		Pprof: &PprofOptions{
			Prefix:               rm.opts.Pprof.Prefix,
			EnableCmdline:        rm.opts.Pprof.EnableCmdline,
			EnableProfile:        rm.opts.Pprof.EnableProfile,
			EnableSymbol:         rm.opts.Pprof.EnableSymbol,
			EnableTrace:          rm.opts.Pprof.EnableTrace,
			BlockProfileRate:     rm.opts.Pprof.BlockProfileRate,
			MutexProfileFraction: rm.opts.Pprof.MutexProfileFraction,
		},
		Auth:  rm.opts.Auth,
		Authz: rm.opts.Authz,
	}

	return opts
}

// SetTimeoutChangeCallback registers a callback to be invoked when timeout configuration changes.
// This allows the actual middleware implementation to update its behavior.
func (rm *ReloadableMiddleware) SetTimeoutChangeCallback(fn func(time.Duration, []string) error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onTimeoutChange = fn
}

// SetCORSChangeCallback registers a callback to be invoked when CORS configuration changes.
// This allows the actual middleware implementation to update its behavior.
func (rm *ReloadableMiddleware) SetCORSChangeCallback(fn func(*CORSOptions) error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onCORSChange = fn
}

// RegisterWithWatcher registers this reloadable middleware with a configuration watcher.
// The handlerID should be unique across all registered handlers.
func (rm *ReloadableMiddleware) RegisterWithWatcher(watcher *configpkg.Watcher, handlerID, configKey string) {
	target := NewOptions()
	subscriber := configpkg.NewReloadableSubscriber(rm, configKey, target)
	watcher.Subscribe(handlerID, subscriber.Handler())
}

// stringSlicesEqual compares two string slices for equality.
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
