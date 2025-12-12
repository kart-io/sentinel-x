package logger

import (
	"fmt"
	"sync"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
	configpkg "github.com/kart-io/sentinel-x/pkg/infra/config"
	options "github.com/kart-io/sentinel-x/pkg/options/logger"
)

// Options is re-exported from pkg/options/logger for convenience.
type Options = options.Options

// NewOptions is re-exported from pkg/options/logger for convenience.
var NewOptions = options.NewOptions

// ReloadableLogger wraps logger options with hot reload capability.
// It maintains thread-safe access to logger configuration and can apply
// configuration changes at runtime without service restart.
type ReloadableLogger struct {
	opts *Options
	mu   sync.RWMutex
}

// NewReloadableLogger creates a new reloadable logger manager.
func NewReloadableLogger(opts *Options) *ReloadableLogger {
	return &ReloadableLogger{
		opts: opts,
	}
}

// OnConfigChange implements the config.Reloadable interface.
// It validates and applies new logger configuration atomically.
// Supported hot-reloadable configuration:
//   - Log level (debug, info, warn, error, fatal)
//   - Log format (json, console)
//   - Output paths
//   - Development mode
//   - Caller and stacktrace settings
func (rl *ReloadableLogger) OnConfigChange(newConfig interface{}) error {
	newOpts, ok := newConfig.(*Options)
	if !ok {
		return fmt.Errorf("invalid config type: expected *logger.Options, got %T", newConfig)
	}

	// Validate new configuration
	if err := newOpts.Validate(); err != nil {
		return fmt.Errorf("invalid logger configuration: %w", err)
	}

	// Acquire write lock
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Store old config for rollback on error
	oldLevel := rl.opts.Level
	oldFormat := rl.opts.Format
	oldDevelopment := rl.opts.Development
	oldDisableCaller := rl.opts.DisableCaller
	oldDisableStacktrace := rl.opts.DisableStacktrace
	oldOutputPaths := rl.opts.OutputPaths

	// Apply new configuration
	rl.opts.Level = newOpts.Level
	rl.opts.Format = newOpts.Format
	rl.opts.Development = newOpts.Development
	rl.opts.DisableCaller = newOpts.DisableCaller
	rl.opts.DisableStacktrace = newOpts.DisableStacktrace
	rl.opts.OutputPaths = newOpts.OutputPaths

	// Note: We update the underlying LogOption as well
	if rl.opts.LogOption == nil {
		rl.opts.LogOption = option.DefaultLogOption()
	}
	rl.opts.Level = newOpts.Level
	rl.opts.Format = newOpts.Format
	rl.opts.Development = newOpts.Development
	rl.opts.DisableCaller = newOpts.DisableCaller
	rl.opts.DisableStacktrace = newOpts.DisableStacktrace
	rl.opts.OutputPaths = newOpts.OutputPaths

	// Try to reinitialize the logger with new settings
	if err := rl.opts.Init(); err != nil {
		// Rollback on error
		rl.opts.Level = oldLevel
		rl.opts.Format = oldFormat
		rl.opts.Development = oldDevelopment
		rl.opts.DisableCaller = oldDisableCaller
		rl.opts.DisableStacktrace = oldDisableStacktrace
		rl.opts.OutputPaths = oldOutputPaths

		// 由于 Options 嵌入了 LogOption，回滚时也要更新嵌入字段
		// 移除冗余的 LogOption. 选择器
		if rl.opts.LogOption != nil {
			rl.opts.Level = oldLevel
			rl.opts.Format = oldFormat
			rl.opts.Development = oldDevelopment
			rl.opts.DisableCaller = oldDisableCaller
			rl.opts.DisableStacktrace = oldDisableStacktrace
			rl.opts.OutputPaths = oldOutputPaths
		}

		return fmt.Errorf("failed to apply logger config: %w", err)
	}

	logger.Infof("Logger configuration reloaded: level=%s, format=%s, development=%v",
		rl.opts.Level, rl.opts.Format, rl.opts.Development)

	return nil
}

// GetOptions returns a copy of the current logger options.
// This is thread-safe for reading.
func (rl *ReloadableLogger) GetOptions() *Options {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	// Return a shallow copy
	opts := &Options{
		LogOption: &option.LogOption{
			Engine:            rl.opts.Engine,
			Level:             rl.opts.Level,
			Format:            rl.opts.Format,
			OutputPaths:       append([]string(nil), rl.opts.OutputPaths...),
			Development:       rl.opts.Development,
			DisableCaller:     rl.opts.DisableCaller,
			DisableStacktrace: rl.opts.DisableStacktrace,
		},
	}

	return opts
}

// RegisterWithWatcher registers this reloadable logger with a configuration watcher.
// The handlerID should be unique across all registered handlers.
func (rl *ReloadableLogger) RegisterWithWatcher(watcher *configpkg.Watcher, handlerID, configKey string) {
	target := NewOptions()
	subscriber := configpkg.NewReloadableSubscriber(rl, configKey, target)
	watcher.Subscribe(handlerID, subscriber.Handler())
}
