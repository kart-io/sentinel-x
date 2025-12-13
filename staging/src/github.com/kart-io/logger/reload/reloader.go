package reload

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/fsnotify/fsnotify"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/errors"
	"github.com/kart-io/logger/factory"
	"github.com/kart-io/logger/option"
)

// ReloadTrigger represents different ways to trigger configuration reload
type ReloadTrigger int

const (
	TriggerNone   ReloadTrigger = 0
	TriggerSignal ReloadTrigger = 1 << iota
	TriggerFileWatch
	TriggerAPI
	TriggerAll = TriggerSignal | TriggerFileWatch | TriggerAPI
)

func (t ReloadTrigger) String() string {
	switch t {
	case TriggerNone:
		return "none"
	case TriggerSignal:
		return "signal"
	case TriggerFileWatch:
		return "file_watch"
	case TriggerAPI:
		return "api"
	default:
		return "multiple"
	}
}

// ReloadCallback is called when configuration is successfully reloaded
type ReloadCallback func(oldConfig, newConfig *option.LogOption) error

// ValidationFunc validates a configuration before applying it
type ValidationFunc func(*option.LogOption) error

// ReloadConfig holds configuration for the reloader
type ReloadConfig struct {
	// ConfigFile path to watch for changes
	ConfigFile string

	// Triggers specifies which reload triggers to enable
	Triggers ReloadTrigger

	// Signals to listen for (default: SIGUSR1, SIGHUP)
	Signals []os.Signal

	// ValidateBeforeReload validates config before applying
	ValidateBeforeReload bool

	// ValidationFunc custom validation function
	ValidationFunc ValidationFunc

	// ReloadTimeout maximum time to wait for reload to complete
	ReloadTimeout time.Duration

	// BackupOnReload whether to backup the current config before reload
	BackupOnReload bool

	// BackupRetention number of backup configurations to keep
	BackupRetention int

	// Callback function called after successful reload
	Callback ReloadCallback

	// Logger for internal logging
	Logger core.Logger
}

// DefaultReloadConfig returns default reload configuration
func DefaultReloadConfig() *ReloadConfig {
	return &ReloadConfig{
		Triggers:             TriggerSignal | TriggerFileWatch,
		Signals:              []os.Signal{syscall.SIGUSR1, syscall.SIGHUP},
		ValidateBeforeReload: true,
		ReloadTimeout:        30 * time.Second,
		BackupOnReload:       true,
		BackupRetention:      5,
	}
}

// ConfigReloader manages dynamic configuration reloading
type ConfigReloader struct {
	mu            sync.RWMutex
	config        *ReloadConfig
	currentConfig *option.LogOption
	factory       *factory.LoggerFactory
	watcher       *fsnotify.Watcher
	signalChan    chan os.Signal
	reloadChan    chan *option.LogOption
	errorHandler  *errors.ErrorHandler
	backupConfigs []*option.LogOption
	ctx           context.Context
	cancel        context.CancelFunc
	running       bool
}

// NewConfigReloader creates a new configuration reloader
func NewConfigReloader(reloadConfig *ReloadConfig, initialConfig *option.LogOption, factory *factory.LoggerFactory) (*ConfigReloader, error) {
	if reloadConfig == nil {
		reloadConfig = DefaultReloadConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	reloader := &ConfigReloader{
		config:        reloadConfig,
		currentConfig: initialConfig,
		factory:       factory,
		reloadChan:    make(chan *option.LogOption, 10),
		errorHandler:  errors.NewErrorHandler(nil),
		backupConfigs: make([]*option.LogOption, 0, reloadConfig.BackupRetention),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Initialize file watcher if needed
	if reloadConfig.Triggers&TriggerFileWatch != 0 && reloadConfig.ConfigFile != "" {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil, fmt.Errorf("failed to create file watcher: %w", err)
		}
		reloader.watcher = watcher

		// Add config file to watch list
		if err := reloader.watcher.Add(reloadConfig.ConfigFile); err != nil {
			reloader.log("warn", fmt.Sprintf("Failed to watch config file %s: %v", reloadConfig.ConfigFile, err))
		}
	}

	// Initialize signal handler if needed
	if reloadConfig.Triggers&TriggerSignal != 0 {
		reloader.signalChan = make(chan os.Signal, 1)
		signal.Notify(reloader.signalChan, reloadConfig.Signals...)
	}

	return reloader, nil
}

// Start begins the configuration reloader
func (r *ConfigReloader) Start() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("reloader is already running")
	}
	r.running = true
	r.mu.Unlock()

	r.log("info", "Configuration reloader started")

	// Start file watcher goroutine
	if r.watcher != nil {
		go r.watchFiles()
	}

	// Start signal handler goroutine
	if r.signalChan != nil {
		go r.handleSignals()
	}

	// Start main reload processing goroutine
	go r.processReloads()

	return nil
}

// Stop stops the configuration reloader
func (r *ConfigReloader) Stop() error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return fmt.Errorf("reloader is not running")
	}
	r.running = false
	r.mu.Unlock()

	r.log("info", "Stopping configuration reloader")

	// Cancel context to stop all goroutines
	r.cancel()

	// Close watchers and channels
	if r.watcher != nil {
		_ = r.watcher.Close()
	}

	if r.signalChan != nil {
		signal.Stop(r.signalChan)
		close(r.signalChan)
	}

	close(r.reloadChan)

	r.log("info", "Configuration reloader stopped")
	return nil
}

// TriggerReload manually triggers a configuration reload
func (r *ConfigReloader) TriggerReload(newConfig *option.LogOption) error {
	if !r.isRunning() {
		return fmt.Errorf("reloader is not running")
	}

	select {
	case r.reloadChan <- newConfig:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout triggering reload")
	}
}

// GetCurrentConfig returns the current configuration
func (r *ConfigReloader) GetCurrentConfig() *option.LogOption {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.currentConfig == nil {
		return nil
	}

	// Return a copy to prevent external modifications
	configCopy := *r.currentConfig
	return &configCopy
}

// GetBackupConfigs returns the backup configurations
func (r *ConfigReloader) GetBackupConfigs() []*option.LogOption {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return copies to prevent external modifications
	backups := make([]*option.LogOption, len(r.backupConfigs))
	for i, backup := range r.backupConfigs {
		configCopy := *backup
		backups[i] = &configCopy
	}
	return backups
}

// RollbackToPrevious rolls back to the previous configuration
func (r *ConfigReloader) RollbackToPrevious() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.backupConfigs) == 0 {
		return fmt.Errorf("no backup configuration available for rollback")
	}

	// Get the most recent backup
	prevConfig := r.backupConfigs[len(r.backupConfigs)-1]

	// Apply the backup configuration
	if err := r.applyConfig(prevConfig); err != nil {
		return fmt.Errorf("failed to rollback to previous configuration: %w", err)
	}

	// Remove the backup we just used
	r.backupConfigs = r.backupConfigs[:len(r.backupConfigs)-1]

	r.log("info", "Successfully rolled back to previous configuration")
	return nil
}

// Private methods

func (r *ConfigReloader) isRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

func (r *ConfigReloader) watchFiles() {
	r.log("debug", "File watcher started")

	for {
		select {
		case <-r.ctx.Done():
			r.log("debug", "File watcher stopped")
			return
		case event, ok := <-r.watcher.Events:
			if !ok {
				r.log("warn", "File watcher events channel closed")
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				r.log("debug", fmt.Sprintf("Config file modified: %s", event.Name))

				// Load the updated configuration
				newConfig, err := r.loadConfigFromFile(event.Name)
				if err != nil {
					r.log("error", fmt.Sprintf("Failed to load config from file %s: %v", event.Name, err))
					continue
				}

				// Trigger reload
				select {
				case r.reloadChan <- newConfig:
				case <-time.After(5 * time.Second):
					r.log("warn", "Timeout sending file-triggered reload")
				}
			}
		case err, ok := <-r.watcher.Errors:
			if !ok {
				r.log("warn", "File watcher errors channel closed")
				return
			}
			r.log("error", fmt.Sprintf("File watcher error: %v", err))
		}
	}
}

func (r *ConfigReloader) handleSignals() {
	r.log("debug", "Signal handler started")

	for {
		select {
		case <-r.ctx.Done():
			r.log("debug", "Signal handler stopped")
			return
		case sig := <-r.signalChan:
			r.log("info", fmt.Sprintf("Received reload signal: %s", sig))

			// For signal-triggered reload, reload from the original config file
			if r.config.ConfigFile != "" {
				newConfig, err := r.loadConfigFromFile(r.config.ConfigFile)
				if err != nil {
					r.log("error", fmt.Sprintf("Failed to load config for signal reload: %v", err))
					continue
				}

				select {
				case r.reloadChan <- newConfig:
				case <-time.After(5 * time.Second):
					r.log("warn", "Timeout sending signal-triggered reload")
				}
			} else {
				r.log("warn", "Signal received but no config file specified")
			}
		}
	}
}

func (r *ConfigReloader) processReloads() {
	r.log("debug", "Reload processor started")

	for {
		select {
		case <-r.ctx.Done():
			r.log("debug", "Reload processor stopped")
			return
		case newConfig := <-r.reloadChan:
			if err := r.handleReload(newConfig); err != nil {
				r.log("error", fmt.Sprintf("Failed to handle configuration reload: %v", err))
			}
		}
	}
}

func (r *ConfigReloader) handleReload(newConfig *option.LogOption) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.log("info", "Processing configuration reload")

	// Validate the new configuration
	if r.config.ValidateBeforeReload {
		if err := r.validateConfig(newConfig); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Backup current configuration if enabled
	if r.config.BackupOnReload {
		r.backupCurrentConfig()
	}

	oldConfig := r.currentConfig

	// Apply the new configuration
	if err := r.applyConfig(newConfig); err != nil {
		return fmt.Errorf("failed to apply new configuration: %w", err)
	}

	// Call the callback if provided
	if r.config.Callback != nil {
		if err := r.config.Callback(oldConfig, newConfig); err != nil {
			r.log("warn", fmt.Sprintf("Reload callback failed: %v", err))
		}
	}

	r.log("info", "Configuration reload completed successfully")
	return nil
}

func (r *ConfigReloader) validateConfig(cfg *option.LogOption) error {
	// Basic validation
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Custom validation if provided
	if r.config.ValidationFunc != nil {
		if err := r.config.ValidationFunc(cfg); err != nil {
			return err
		}
	}

	return nil
}

func (r *ConfigReloader) applyConfig(newConfig *option.LogOption) error {
	// Update the factory with the new configuration
	if err := r.factory.UpdateOption(newConfig); err != nil {
		return fmt.Errorf("failed to update factory configuration: %w", err)
	}

	// Update current configuration
	r.currentConfig = newConfig

	return nil
}

func (r *ConfigReloader) backupCurrentConfig() {
	// Add current config to backup list
	configCopy := *r.currentConfig
	r.backupConfigs = append(r.backupConfigs, &configCopy)

	// Maintain backup retention limit
	if len(r.backupConfigs) > r.config.BackupRetention {
		r.backupConfigs = r.backupConfigs[len(r.backupConfigs)-r.config.BackupRetention:]
	}
}

func (r *ConfigReloader) loadConfigFromFile(filename string) (*option.LogOption, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &option.LogOption{}

	// Determine file format by extension
	ext := filepath.Ext(filename)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	return cfg, nil
}

func (r *ConfigReloader) log(level, message string) {
	if r.config.Logger == nil {
		return
	}

	switch level {
	case "debug":
		r.config.Logger.Debugw("ConfigReloader", "component", "reloader", "message", message)
	case "info":
		r.config.Logger.Infow("ConfigReloader", "component", "reloader", "message", message)
	case "warn":
		r.config.Logger.Warnw("ConfigReloader", "component", "reloader", "message", message)
	case "error":
		r.config.Logger.Errorw("ConfigReloader", "component", "reloader", "message", message)
	}
}
