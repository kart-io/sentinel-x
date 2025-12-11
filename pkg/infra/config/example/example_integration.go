// Package example demonstrates how to integrate config hot reload
// with the Sentinel-X application bootstrap process.
//
// This is an example implementation showing where to add the config
// watcher in the application initialization flow.
package example

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/config"
	infralog "github.com/kart-io/sentinel-x/pkg/infra/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/spf13/viper"
)

// ExampleBootstrapWithConfigReload demonstrates how to integrate
// the config watcher into the application bootstrap process.
//
// Add this to your bootstrap.go or main application initialization:
func ExampleBootstrapWithConfigReload(
	ctx context.Context,
	v *viper.Viper,
	logOpts *logopts.Options,
	mwOpts *mwopts.Options,
) (*config.Watcher, error) {
	// 1. Create reloadable wrappers for components
	reloadableLogger := infralog.NewReloadableLogger(logOpts)
	reloadableMiddleware := middleware.NewReloadableMiddleware(mwOpts)

	// 2. Create config watcher
	watcher := config.NewWatcher(v)

	// 3. Register reloadable components with the watcher
	//    The handler IDs should be unique across your application
	//    The config keys match your YAML structure
	reloadableLogger.RegisterWithWatcher(watcher, "app.logger", "log")
	reloadableMiddleware.RegisterWithWatcher(watcher, "app.middleware", "server.http.middleware")

	// 4. Optionally: Register custom handlers for other components
	watcher.Subscribe("app.custom", func(v *viper.Viper) error {
		// Handle custom configuration changes
		logger.Info("Custom configuration changed")
		return nil
	})

	// 5. Start watching for configuration changes
	watcher.Start()

	logger.Info("Configuration hot reload enabled")

	return watcher, nil
}

// ExampleCustomReloadableComponent shows how to create a custom
// component that implements the Reloadable interface.
type ExampleCustomReloadableComponent struct {
	config ExampleConfig
	// Add sync.RWMutex for thread-safe access
	// mu sync.RWMutex
}

type ExampleConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	MaxRetries int    `mapstructure:"max_retries"`
	Timeout    string `mapstructure:"timeout"`
}

// OnConfigChange implements the config.Reloadable interface.
func (c *ExampleCustomReloadableComponent) OnConfigChange(newConfig interface{}) error {
	cfg, ok := newConfig.(*ExampleConfig)
	if !ok {
		return fmt.Errorf("invalid config type: expected *ExampleConfig, got %T", newConfig)
	}

	// Validate new configuration
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}

	// Apply changes atomically
	// c.mu.Lock()
	// defer c.mu.Unlock()

	oldRetries := c.config.MaxRetries
	c.config = *cfg

	logger.Infof("Component config reloaded: max_retries %d -> %d", oldRetries, cfg.MaxRetries)

	return nil
}

// RegisterCustomComponent shows how to register a custom component with the watcher.
func RegisterCustomComponent(watcher *config.Watcher) {
	component := &ExampleCustomReloadableComponent{}
	target := &ExampleConfig{}

	subscriber := config.NewReloadableSubscriber(component, "custom.component", target)
	watcher.Subscribe("custom-component", subscriber.Handler())
}

// ExampleIntegrationWithApp shows the complete integration in an application.
func ExampleIntegrationWithApp() {
	// This example shows the typical flow in your main() or app.Run()

	// 1. Load configuration with viper
	v := viper.New()
	v.SetConfigFile("configs/sentinel-api.yaml")
	if err := v.ReadInConfig(); err != nil {
		logger.Fatalf("Failed to read config: %v", err)
	}

	// 2. Parse initial configuration
	logOpts := logopts.NewOptions()
	if err := v.UnmarshalKey("log", logOpts); err != nil {
		logger.Fatalf("Failed to unmarshal log config: %v", err)
	}

	mwOpts := mwopts.NewOptions()
	if err := v.UnmarshalKey("server.http.middleware", mwOpts); err != nil {
		logger.Fatalf("Failed to unmarshal middleware config: %v", err)
	}

	// 3. Initialize logger with initial config
	if err := logOpts.Init(); err != nil {
		logger.Fatalf("Failed to initialize logger: %v", err)
	}

	// 4. Set up config hot reload
	ctx := context.Background()
	watcher, err := ExampleBootstrapWithConfigReload(ctx, v, logOpts, mwOpts)
	if err != nil {
		logger.Fatalf("Failed to setup config reload: %v", err)
	}

	// 5. Continue with normal application initialization
	//    The watcher is now monitoring config changes in the background

	// 6. On application shutdown, stop the watcher
	defer watcher.Stop()

	// Your application runs here...
	logger.Info("Application running with config hot reload")
}

// ExampleMiddlewareCallbacks demonstrates how to set up callbacks
// for middleware configuration changes.
func ExampleMiddlewareCallbacks(rm *middleware.ReloadableMiddleware) {
	// Set timeout change callback
	rm.SetTimeoutChangeCallback(func(newTimeout time.Duration, skipPaths []string) error {
		logger.Infof("Timeout changed to: %v", newTimeout)
		// Update your middleware implementation here
		return nil
	})

	// Set CORS change callback
	rm.SetCORSChangeCallback(func(corsOpts *mwopts.CORSOptions) error {
		logger.Infof("CORS configuration updated: %d origins", len(corsOpts.AllowOrigins))
		// Update your CORS middleware here
		return nil
	})
}
