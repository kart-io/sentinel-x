// Package config provides configuration management and hot reload capabilities.
package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/kart-io/logger"
	"github.com/spf13/viper"
)

// ChangeHandler is a callback function invoked when configuration changes.
// It receives the updated viper instance and should return an error if it cannot
// handle the configuration change.
type ChangeHandler func(v *viper.Viper) error

// Watcher manages configuration file watching and change notifications.
// It uses fsnotify via viper to monitor configuration file changes and
// provides a thread-safe subscription mechanism for components to react to changes.
type Watcher struct {
	viper    *viper.Viper
	handlers map[string]ChangeHandler
	mu       sync.RWMutex
	watching bool
}

// NewWatcher creates a new configuration watcher.
// The provided viper instance should already be initialized with a configuration file.
func NewWatcher(v *viper.Viper) *Watcher {
	return &Watcher{
		viper:    v,
		handlers: make(map[string]ChangeHandler),
		watching: false,
	}
}

// Subscribe registers a change handler with the given identifier.
// The handler will be called when the configuration file changes.
// If a handler with the same ID already exists, it will be replaced.
// This operation is thread-safe.
func (w *Watcher) Subscribe(id string, handler ChangeHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[id] = handler
	logger.Infof("Config watcher: subscribed handler '%s'", id)
}

// Unsubscribe removes a change handler by its identifier.
// This operation is thread-safe and is safe to call even if the handler doesn't exist.
func (w *Watcher) Unsubscribe(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.handlers[id]; exists {
		delete(w.handlers, id)
		logger.Infof("Config watcher: unsubscribed handler '%s'", id)
	}
}

// Start begins watching the configuration file for changes.
// When a change is detected, all registered handlers are notified sequentially.
// If any handler returns an error, it will be logged but will not stop other handlers.
// This method is idempotent - calling it multiple times has no additional effect.
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.watching {
		w.mu.Unlock()
		return
	}
	w.watching = true
	w.mu.Unlock()

	w.viper.WatchConfig()
	w.viper.OnConfigChange(func(e fsnotify.Event) {
		logger.Infof("Config file changed: %s", e.Name)

		// Acquire read lock to iterate handlers
		w.mu.RLock()
		handlers := make(map[string]ChangeHandler, len(w.handlers))
		for id, handler := range w.handlers {
			handlers[id] = handler
		}
		w.mu.RUnlock()

		// Call handlers without holding the lock
		for id, handler := range handlers {
			if err := handler(w.viper); err != nil {
				logger.Errorf("Config watcher: handler '%s' failed: %v", id, err)
			} else {
				logger.Infof("Config watcher: handler '%s' processed change successfully", id)
			}
		}
	})

	logger.Info("Config watcher: started watching for configuration changes")
}

// Stop stops watching the configuration file.
// This is a no-op in the current implementation as viper doesn't provide
// a mechanism to stop watching. The method is provided for API consistency
// and future extensibility.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.watching {
		return
	}
	// Note: viper doesn't provide a way to stop watching
	// We just mark it as not watching
	w.watching = false
	logger.Info("Config watcher: stopped")
}

// IsWatching returns whether the watcher is currently active.
func (w *Watcher) IsWatching() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.watching
}

// HandlerCount returns the number of registered handlers.
// This is primarily useful for testing and diagnostics.
func (w *Watcher) HandlerCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.handlers)
}

// ReloadableSubscriber is a helper that wraps a Reloadable component
// and creates a ChangeHandler that unmarshals config into the component.
type ReloadableSubscriber struct {
	component Reloadable
	configKey string
	target    interface{}
}

// NewReloadableSubscriber creates a new subscriber for a Reloadable component.
// configKey is the viper key path to unmarshal (e.g., "server", "log").
// target should be a pointer to the configuration structure.
func NewReloadableSubscriber(component Reloadable, configKey string, target interface{}) *ReloadableSubscriber {
	return &ReloadableSubscriber{
		component: component,
		configKey: configKey,
		target:    target,
	}
}

// Handler returns a ChangeHandler that can be registered with the Watcher.
// When invoked, it unmarshals the configuration at the configured key
// and passes it to the component's OnConfigChange method.
func (rs *ReloadableSubscriber) Handler() ChangeHandler {
	return func(v *viper.Viper) error {
		// Unmarshal the specific configuration section
		if err := v.UnmarshalKey(rs.configKey, rs.target); err != nil {
			return fmt.Errorf("failed to unmarshal config key '%s': %w", rs.configKey, err)
		}

		// Notify the component
		if err := rs.component.OnConfigChange(rs.target); err != nil {
			return fmt.Errorf("component rejected config change: %w", err)
		}

		return nil
	}
}
