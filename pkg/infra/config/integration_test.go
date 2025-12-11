//go:build integration
// +build integration

package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/config"
	"github.com/kart-io/sentinel-x/pkg/infra/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/spf13/viper"
)

// TestIntegrationFullReload demonstrates a complete configuration reload scenario
// with multiple components.
func TestIntegrationFullReload(t *testing.T) {
	// Create temporary directory for config file
	tmpDir, err := os.MkdirTemp("", "integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "sentinel-api.yaml")
	initialConfig := []byte(`
log:
  level: info
  format: json
  development: false
  disable-caller: false
  disable-stacktrace: true
  output-paths:
    - stdout

server:
  http:
    middleware:
      disable-cors: false
      cors:
        allow-origins:
          - "*"
        allow-methods:
          - GET
          - POST
        allow-credentials: false
        max-age: 86400

      disable-timeout: false
      timeout:
        timeout: 30s
        skip-paths:
          - /health
          - /metrics

      logger:
        skip-paths:
          - /health
        use-structured-logger: true

      recovery:
        enable-stack-trace: false

      request-id:
        header: X-Request-ID
`)

	if err := os.WriteFile(configFile, initialConfig, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize viper
	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Parse logger options
	logOpts := logopts.NewOptions()
	if err := v.UnmarshalKey("log", logOpts); err != nil {
		t.Fatalf("Failed to unmarshal log config: %v", err)
	}

	// Parse middleware options
	mwOpts := mwopts.NewOptions()
	if err := v.UnmarshalKey("server.http.middleware", mwOpts); err != nil {
		t.Fatalf("Failed to unmarshal middleware config: %v", err)
	}

	// Verify initial configuration
	if logOpts.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", logOpts.Level)
	}
	if mwOpts.Timeout.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", mwOpts.Timeout.Timeout)
	}

	// Create reloadable components
	reloadableLogger := logger.NewReloadableLogger(logOpts)
	reloadableMiddleware := middleware.NewReloadableMiddleware(mwOpts)

	// Create watcher and register components
	watcher := config.NewWatcher(v)
	reloadableLogger.RegisterWithWatcher(watcher, "logger", "log")
	reloadableMiddleware.RegisterWithWatcher(watcher, "middleware", "server.http.middleware")

	// Start watching
	watcher.Start()

	if !watcher.IsWatching() {
		t.Error("Watcher should be watching")
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Update configuration file
	updatedConfig := []byte(`
log:
  level: debug
  format: text
  development: true
  disable-caller: false
  disable-stacktrace: false
  output-paths:
    - stdout
    - stderr

server:
  http:
    middleware:
      disable-cors: false
      cors:
        allow-origins:
          - "https://example.com"
          - "https://api.example.com"
        allow-methods:
          - GET
          - POST
          - PUT
          - DELETE
        allow-credentials: true
        max-age: 3600

      disable-timeout: false
      timeout:
        timeout: 60s
        skip-paths:
          - /health
          - /metrics
          - /debug

      logger:
        skip-paths:
          - /health
          - /metrics
        use-structured-logger: false

      recovery:
        enable-stack-trace: true

      request-id:
        header: X-Trace-ID
`)

	if err := os.WriteFile(configFile, updatedConfig, 0o644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	// Wait for config reload (fsnotify needs time to detect and process)
	time.Sleep(1 * time.Second)

	// Verify logger configuration was updated
	currentLogOpts := reloadableLogger.GetOptions()
	if currentLogOpts.Level != "debug" {
		t.Errorf("Expected log level 'debug' after reload, got '%s'", currentLogOpts.Level)
	}
	if currentLogOpts.Format != "text" {
		t.Errorf("Expected log format 'text' after reload, got '%s'", currentLogOpts.Format)
	}
	if !currentLogOpts.Development {
		t.Error("Expected development mode to be true after reload")
	}

	// Verify middleware configuration was updated
	currentMwOpts := reloadableMiddleware.GetOptions()
	if currentMwOpts.Timeout.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s after reload, got %v", currentMwOpts.Timeout.Timeout)
	}
	if currentMwOpts.RequestID.Header != "X-Trace-ID" {
		t.Errorf("Expected request ID header 'X-Trace-ID' after reload, got '%s'",
			currentMwOpts.RequestID.Header)
	}
	if currentMwOpts.CORS.MaxAge != 3600 {
		t.Errorf("Expected CORS max age 3600 after reload, got %d", currentMwOpts.CORS.MaxAge)
	}
	if len(currentMwOpts.CORS.AllowOrigins) != 2 {
		t.Errorf("Expected 2 CORS origins after reload, got %d", len(currentMwOpts.CORS.AllowOrigins))
	}
	if !currentMwOpts.Recovery.EnableStackTrace {
		t.Error("Expected recovery stack trace to be enabled after reload")
	}

	// Cleanup
	watcher.Stop()
}

// TestIntegrationLoggerReload focuses on logger configuration reload.
func TestIntegrationLoggerReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-reload-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.yaml")
	initialConfig := []byte(`
log:
  level: warn
  format: json
`)

	if err := os.WriteFile(configFile, initialConfig, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	logOpts := logopts.NewOptions()
	if err := v.UnmarshalKey("log", logOpts); err != nil {
		t.Fatalf("Failed to unmarshal log config: %v", err)
	}

	reloadableLogger := logger.NewReloadableLogger(logOpts)
	watcher := config.NewWatcher(v)
	reloadableLogger.RegisterWithWatcher(watcher, "logger", "log")
	watcher.Start()

	time.Sleep(100 * time.Millisecond)

	// Change log level
	updatedConfig := []byte(`
log:
  level: error
  format: json
`)

	if err := os.WriteFile(configFile, updatedConfig, 0o644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	time.Sleep(1 * time.Second)

	currentOpts := reloadableLogger.GetOptions()
	if currentOpts.Level != "error" {
		t.Errorf("Expected log level 'error' after reload, got '%s'", currentOpts.Level)
	}

	watcher.Stop()
}

// TestIntegrationMiddlewareReload focuses on middleware configuration reload.
func TestIntegrationMiddlewareReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "middleware-reload-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.yaml")
	initialConfig := []byte(`
middleware:
  timeout:
    timeout: 15s
  cors:
    max-age: 7200
`)

	if err := os.WriteFile(configFile, initialConfig, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	mwOpts := mwopts.NewOptions()
	if err := v.UnmarshalKey("middleware", mwOpts); err != nil {
		t.Fatalf("Failed to unmarshal middleware config: %v", err)
	}

	reloadableMiddleware := middleware.NewReloadableMiddleware(mwOpts)
	watcher := config.NewWatcher(v)
	reloadableMiddleware.RegisterWithWatcher(watcher, "middleware", "middleware")
	watcher.Start()

	time.Sleep(100 * time.Millisecond)

	// Change timeout and CORS settings
	updatedConfig := []byte(`
middleware:
  timeout:
    timeout: 45s
  cors:
    max-age: 10800
`)

	if err := os.WriteFile(configFile, updatedConfig, 0o644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	time.Sleep(1 * time.Second)

	currentOpts := reloadableMiddleware.GetOptions()
	if currentOpts.Timeout.Timeout != 45*time.Second {
		t.Errorf("Expected timeout 45s after reload, got %v", currentOpts.Timeout.Timeout)
	}
	if currentOpts.CORS.MaxAge != 10800 {
		t.Errorf("Expected CORS max age 10800 after reload, got %d", currentOpts.CORS.MaxAge)
	}

	watcher.Stop()
}

// TestIntegrationUnsubscribe verifies that unsubscribing stops config updates.
func TestIntegrationUnsubscribe(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unsubscribe-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.yaml")
	initialConfig := []byte(`
log:
  level: info
`)

	if err := os.WriteFile(configFile, initialConfig, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	logOpts := logopts.NewOptions()
	if err := v.UnmarshalKey("log", logOpts); err != nil {
		t.Fatalf("Failed to unmarshal log config: %v", err)
	}

	reloadableLogger := logger.NewReloadableLogger(logOpts)
	watcher := config.NewWatcher(v)
	reloadableLogger.RegisterWithWatcher(watcher, "logger", "log")
	watcher.Start()

	time.Sleep(100 * time.Millisecond)

	// Unsubscribe the logger
	watcher.Unsubscribe("logger")

	// Change config
	updatedConfig := []byte(`
log:
  level: debug
`)

	if err := os.WriteFile(configFile, updatedConfig, 0o644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Logger should still have old config since it was unsubscribed
	currentOpts := reloadableLogger.GetOptions()
	if currentOpts.Level != "info" {
		t.Errorf("Expected log level to remain 'info' after unsubscribe, got '%s'", currentOpts.Level)
	}

	watcher.Stop()
}
