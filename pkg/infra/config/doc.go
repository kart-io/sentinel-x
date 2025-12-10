// Package config provides configuration management and hot reload capabilities.
//
// Example Usage:
//
// This example demonstrates how to set up configuration hot reload for the Sentinel-X application.
//
//	package main
//
//	import (
//	    "github.com/kart-io/sentinel-x/cmd/api/app"
//	    "github.com/kart-io/sentinel-x/pkg/infra/config"
//	    "github.com/kart-io/sentinel-x/pkg/infra/logger"
//	    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
//	    "github.com/spf13/viper"
//	)
//
//	func main() {
//	    // 1. Load initial configuration
//	    opts := app.NewOptions()
//	    v := viper.New()
//	    v.SetConfigFile("configs/sentinel-api.yaml")
//	    if err := v.ReadInConfig(); err != nil {
//	        panic(err)
//	    }
//	    if err := v.Unmarshal(opts); err != nil {
//	        panic(err)
//	    }
//
//	    // 2. Initialize logger with options
//	    if err := opts.Log.Init(); err != nil {
//	        panic(err)
//	    }
//
//	    // 3. Create reloadable components
//	    reloadableLogger := logger.NewReloadableLogger(opts.Log)
//	    reloadableMiddleware := middleware.NewReloadableMiddleware(opts.Server.HTTP.Middleware)
//
//	    // 4. Create and configure the config watcher
//	    watcher := config.NewWatcher(v)
//
//	    // 5. Register reloadable components with the watcher
//	    reloadableLogger.RegisterWithWatcher(watcher, "logger", "log")
//	    reloadableMiddleware.RegisterWithWatcher(watcher, "middleware", "server.http.middleware")
//
//	    // 6. Start watching for configuration changes
//	    watcher.Start()
//
//	    // 7. Run your application
//	    // When config file changes, registered components will be notified automatically
//	    app.Run(opts)
//	}
//
// Custom Reloadable Component:
//
// To create a custom component that reacts to configuration changes:
//
//	type MyService struct {
//	    config MyConfig
//	    mu     sync.RWMutex
//	}
//
//	func (s *MyService) OnConfigChange(newConfig interface{}) error {
//	    cfg, ok := newConfig.(*MyConfig)
//	    if !ok {
//	        return fmt.Errorf("invalid config type")
//	    }
//
//	    // Validate new configuration
//	    if err := cfg.Validate(); err != nil {
//	        return err
//	    }
//
//	    // Apply changes atomically
//	    s.mu.Lock()
//	    defer s.mu.Unlock()
//	    s.config = *cfg
//
//	    logger.Info("MyService configuration reloaded")
//	    return nil
//	}
//
//	// Register with watcher
//	service := &MyService{}
//	target := &MyConfig{}
//	subscriber := config.NewReloadableSubscriber(service, "myservice", target)
//	watcher.Subscribe("myservice", subscriber.Handler())
//
// Thread Safety:
//
// All config watcher operations are thread-safe. You can subscribe/unsubscribe
// handlers from multiple goroutines concurrently. When a config change is detected,
// all handlers are called sequentially (not concurrently) to ensure predictable
// behavior and easier error handling.
//
// Error Handling:
//
// If a handler returns an error when processing a config change, the error is logged
// but does not stop other handlers from being called. Each component is responsible
// for maintaining its previous valid state if a config change fails.
package config
