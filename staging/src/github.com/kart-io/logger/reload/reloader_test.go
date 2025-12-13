package reload

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kart-io/logger/factory"
	"github.com/kart-io/logger/option"
)

func TestNewConfigReloader(t *testing.T) {
	// Create test config
	cfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	// Create factory
	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	// Test with default config
	reloader, err := NewConfigReloader(nil, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader with default config: %v", err)
	}

	if reloader.config.Triggers != (TriggerSignal | TriggerFileWatch) {
		t.Errorf("Expected default triggers to be signal and file watch")
	}

	// Test with custom config
	reloadConfig := &ReloadConfig{
		Triggers:      TriggerSignal,
		ReloadTimeout: 10 * time.Second,
	}
	reloader2, err := NewConfigReloader(reloadConfig, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader with custom config: %v", err)
	}

	if reloader2.config.Triggers != TriggerSignal {
		t.Errorf("Expected custom triggers to be signal only")
	}
}

func TestConfigReloader_StartStop(t *testing.T) {
	cfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	reloadConfig := &ReloadConfig{
		Triggers: TriggerSignal,
	}

	reloader, err := NewConfigReloader(reloadConfig, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	// Test start
	if err := reloader.Start(); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}

	if !reloader.isRunning() {
		t.Error("Reloader should be running after start")
	}

	// Test double start (should fail)
	if err := reloader.Start(); err == nil {
		t.Error("Expected error when starting already running reloader")
	}

	// Test stop
	if err := reloader.Stop(); err != nil {
		t.Fatalf("Failed to stop reloader: %v", err)
	}

	if reloader.isRunning() {
		t.Error("Reloader should not be running after stop")
	}

	// Test double stop (should fail)
	if err := reloader.Stop(); err == nil {
		t.Error("Expected error when stopping already stopped reloader")
	}
}

func TestConfigReloader_GetCurrentConfig(t *testing.T) {
	cfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	reloader, err := NewConfigReloader(nil, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	currentCfg := reloader.GetCurrentConfig()
	if currentCfg.Engine != cfg.Engine {
		t.Errorf("Expected engine %s, got %s", cfg.Engine, currentCfg.Engine)
	}

	// Verify it's a copy by checking it's not the same pointer
	if currentCfg == reloader.currentConfig {
		t.Error("GetCurrentConfig should return a copy, not the original pointer")
	}

	// Verify the content is the same
	currentCfg2 := reloader.GetCurrentConfig()
	if currentCfg2.Engine != currentCfg.Engine {
		t.Error("Multiple calls to GetCurrentConfig should return the same content")
	}
}

func TestConfigReloader_TriggerReload(t *testing.T) {
	cfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	reloader, err := NewConfigReloader(nil, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	// Test trigger reload when not running (should fail)
	newCfg := &option.LogOption{
		Engine: "zap",
		Level:  "DEBUG",
		Format: "console",
	}

	if err := reloader.TriggerReload(newCfg); err == nil {
		t.Error("Expected error when triggering reload on stopped reloader")
	}

	// Start reloader and test successful reload
	if err := reloader.Start(); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}
	defer func() {
		_ = reloader.Stop()
	}()

	// Give some time for goroutines to start
	time.Sleep(100 * time.Millisecond)

	if err := reloader.TriggerReload(newCfg); err != nil {
		t.Errorf("Failed to trigger reload: %v", err)
	}

	// Give some time for reload to process
	time.Sleep(200 * time.Millisecond)

	// Check if config was updated
	currentCfg := reloader.GetCurrentConfig()
	if currentCfg.Engine != "zap" {
		t.Errorf("Expected engine to be updated to zap, got %s", currentCfg.Engine)
	}
}

func TestConfigReloader_FileWatch(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	initialCfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	// Write initial config
	data, err := yaml.Marshal(initialCfg)
	if err != nil {
		t.Fatalf("Failed to marshal initial config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0o644); err != nil {
		t.Fatalf("Failed to write initial config file: %v", err)
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	reloadConfig := &ReloadConfig{
		ConfigFile: configFile,
		Triggers:   TriggerFileWatch,
	}

	reloader, err := NewConfigReloader(reloadConfig, initialCfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	// Start reloader
	if err := reloader.Start(); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}
	defer func() {
		_ = reloader.Stop()
	}()

	// Give some time for file watcher to initialize
	time.Sleep(200 * time.Millisecond)

	// Modify config file
	updatedCfg := &option.LogOption{
		Engine: "zap",
		Level:  "DEBUG",
		Format: "console",
	}

	data, err = yaml.Marshal(updatedCfg)
	if err != nil {
		t.Fatalf("Failed to marshal updated config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0o644); err != nil {
		t.Fatalf("Failed to write updated config file: %v", err)
	}

	// Give some time for file change to be detected and processed
	time.Sleep(500 * time.Millisecond)

	// Check if config was reloaded
	currentCfg := reloader.GetCurrentConfig()
	if currentCfg.Engine != "zap" {
		t.Errorf("Expected engine to be updated to zap via file watch, got %s", currentCfg.Engine)
	}
	if currentCfg.Level != "DEBUG" {
		t.Errorf("Expected level to be updated to DEBUG via file watch, got %s", currentCfg.Level)
	}
}

func TestConfigReloader_SignalReload(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.json")

	initialCfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	// Write initial config
	data, err := json.Marshal(initialCfg)
	if err != nil {
		t.Fatalf("Failed to marshal initial config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0o644); err != nil {
		t.Fatalf("Failed to write initial config file: %v", err)
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	reloadConfig := &ReloadConfig{
		ConfigFile: configFile,
		Triggers:   TriggerSignal,
		Signals:    []os.Signal{syscall.SIGUSR1},
	}

	reloader, err := NewConfigReloader(reloadConfig, initialCfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	// Start reloader
	if err := reloader.Start(); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}
	defer func() {
		_ = reloader.Stop()
	}()

	// Give some time for signal handler to initialize
	time.Sleep(200 * time.Millisecond)

	// Update config file first
	updatedCfg := &option.LogOption{
		Engine: "zap",
		Level:  "WARN",
		Format: "console",
	}

	data, err = json.Marshal(updatedCfg)
	if err != nil {
		t.Fatalf("Failed to marshal updated config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0o644); err != nil {
		t.Fatalf("Failed to write updated config file: %v", err)
	}

	// Send SIGUSR1 signal to current process
	if err := syscall.Kill(os.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatalf("Failed to send SIGUSR1 signal: %v", err)
	}

	// Give some time for signal to be processed
	time.Sleep(500 * time.Millisecond)

	// Check if config was reloaded
	currentCfg := reloader.GetCurrentConfig()
	if currentCfg.Engine != "zap" {
		t.Errorf("Expected engine to be updated to zap via signal, got %s", currentCfg.Engine)
	}
	if currentCfg.Level != "WARN" {
		t.Errorf("Expected level to be updated to WARN via signal, got %s", currentCfg.Level)
	}
}

func TestConfigReloader_BackupAndRollback(t *testing.T) {
	cfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	reloadConfig := &ReloadConfig{
		BackupOnReload:  true,
		BackupRetention: 3,
	}

	reloader, err := NewConfigReloader(reloadConfig, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	// Start reloader
	if err := reloader.Start(); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}
	defer func() {
		_ = reloader.Stop()
	}()

	// Give some time for goroutines to start
	time.Sleep(100 * time.Millisecond)

	// Test rollback with no backups
	if err := reloader.RollbackToPrevious(); err == nil {
		t.Error("Expected error when rolling back with no backups")
	}

	// Trigger several reloads to create backups
	configs := []*option.LogOption{
		{Engine: "zap", Level: "DEBUG", Format: "console"},
		{Engine: "slog", Level: "WARN", Format: "json"},
		{Engine: "zap", Level: "ERROR", Format: "console"},
	}

	for _, newCfg := range configs {
		if err := reloader.TriggerReload(newCfg); err != nil {
			t.Fatalf("Failed to trigger reload: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Check backups
	backups := reloader.GetBackupConfigs()
	if len(backups) != 3 {
		t.Errorf("Expected 3 backup configurations, got %d", len(backups))
	}

	// Current config should be the last one
	currentCfg := reloader.GetCurrentConfig()
	if currentCfg.Engine != "zap" || currentCfg.Level != "ERROR" {
		t.Errorf("Current config doesn't match last reload")
	}

	// Test rollback
	if err := reloader.RollbackToPrevious(); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Check if rolled back to previous config
	rolledBackCfg := reloader.GetCurrentConfig()
	if rolledBackCfg.Engine != "slog" || rolledBackCfg.Level != "WARN" {
		t.Errorf("Rollback didn't restore previous config correctly")
	}

	// Check that backup was removed
	backups = reloader.GetBackupConfigs()
	if len(backups) != 2 {
		t.Errorf("Expected 2 backup configurations after rollback, got %d", len(backups))
	}
}

func TestConfigReloader_ValidationCallback(t *testing.T) {
	cfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	var mu sync.Mutex
	var callbackCalled bool
	var callbackOldConfig, callbackNewConfig *option.LogOption

	reloadConfig := &ReloadConfig{
		ValidateBeforeReload: true,
		ValidationFunc: func(cfg *option.LogOption) error {
			// Reject configs with ERROR level
			if cfg.Level == "ERROR" {
				return os.ErrInvalid
			}
			return nil
		},
		Callback: func(oldCfg, newCfg *option.LogOption) error {
			mu.Lock()
			defer mu.Unlock()
			callbackCalled = true
			callbackOldConfig = oldCfg
			callbackNewConfig = newCfg
			return nil
		},
	}

	reloader, err := NewConfigReloader(reloadConfig, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	// Start reloader
	if err := reloader.Start(); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}
	defer func() {
		_ = reloader.Stop()
	}()

	// Give some time for goroutines to start
	time.Sleep(100 * time.Millisecond)

	// Test valid config
	validCfg := &option.LogOption{
		Engine: "zap",
		Level:  "DEBUG",
		Format: "console",
	}

	if err := reloader.TriggerReload(validCfg); err != nil {
		t.Errorf("Valid config should not fail reload: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	called := callbackCalled
	oldCfg := callbackOldConfig
	newCfg := callbackNewConfig
	mu.Unlock()

	if !called {
		t.Error("Callback should be called for successful reload")
	}

	if oldCfg.Level != "INFO" || newCfg.Level != "DEBUG" {
		t.Error("Callback received incorrect config parameters")
	}

	// Reset callback state
	mu.Lock()
	callbackCalled = false
	mu.Unlock()

	// Test invalid config (should be rejected by validation)
	invalidCfg := &option.LogOption{
		Engine: "zap",
		Level:  "ERROR", // This should be rejected by validation
		Format: "console",
	}

	if err := reloader.TriggerReload(invalidCfg); err != nil {
		t.Errorf("TriggerReload should not fail even if validation fails: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Config should not have changed due to validation failure
	currentCfg := reloader.GetCurrentConfig()
	if currentCfg.Level == "ERROR" {
		t.Error("Invalid config should not have been applied")
	}

	// Callback should not have been called for failed validation
	mu.Lock()
	called = callbackCalled
	mu.Unlock()
	if called {
		t.Error("Callback should not be called for failed validation")
	}
}

func TestConfigReloader_ConcurrentAccess(t *testing.T) {
	cfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	reloader, err := NewConfigReloader(nil, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	// Start reloader
	if err := reloader.Start(); err != nil {
		t.Fatalf("Failed to start reloader: %v", err)
	}
	defer func() {
		_ = reloader.Stop()
	}()

	// Give some time for goroutines to start
	time.Sleep(100 * time.Millisecond)

	// Test concurrent access
	var wg sync.WaitGroup
	const numGoroutines = 10
	const numOperations = 50

	// Concurrent config reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = reloader.GetCurrentConfig()
				_ = reloader.GetBackupConfigs()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Concurrent reloads
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations/5; j++ {
				newCfg := &option.LogOption{
					Engine: "slog",
					Level:  "DEBUG",
					Format: "json",
				}
				_ = reloader.TriggerReload(newCfg)
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Should still be functional after concurrent access
	currentCfg := reloader.GetCurrentConfig()
	if currentCfg == nil {
		t.Error("Config should not be nil after concurrent access")
	}
}

func TestConfigReloader_LoadConfigFromFile(t *testing.T) {
	cfg := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}

	opt := &option.LogOption{
		Engine: "slog",
		Level:  "INFO",
		Format: "json",
	}
	factory := factory.NewLoggerFactory(opt)

	reloader, err := NewConfigReloader(nil, cfg, factory)
	if err != nil {
		t.Fatalf("Failed to create reloader: %v", err)
	}

	tmpDir := t.TempDir()

	// Test YAML file
	yamlFile := filepath.Join(tmpDir, "config.yaml")
	yamlData := `
engine: zap
level: DEBUG
format: console
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0o644); err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	yamlCfg, err := reloader.loadConfigFromFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	if yamlCfg.Engine != "zap" || yamlCfg.Level != "DEBUG" || yamlCfg.Format != "console" {
		t.Error("YAML config not loaded correctly")
	}

	// Test JSON file
	jsonFile := filepath.Join(tmpDir, "config.json")
	jsonData := `{
		"engine": "slog",
		"level": "WARN",
		"format": "json"
	}`
	if err := os.WriteFile(jsonFile, []byte(jsonData), 0o644); err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}

	jsonCfg, err := reloader.loadConfigFromFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	if jsonCfg.Engine != "slog" || jsonCfg.Level != "WARN" || jsonCfg.Format != "json" {
		t.Error("JSON config not loaded correctly")
	}

	// Test unsupported file format
	txtFile := filepath.Join(tmpDir, "config.txt")
	if err := os.WriteFile(txtFile, []byte("invalid"), 0o644); err != nil {
		t.Fatalf("Failed to write TXT file: %v", err)
	}

	_, err = reloader.loadConfigFromFile(txtFile)
	if err == nil {
		t.Error("Expected error for unsupported file format")
	}

	// Test non-existent file
	_, err = reloader.loadConfigFromFile("/non/existent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestReloadTrigger_String(t *testing.T) {
	tests := []struct {
		trigger  ReloadTrigger
		expected string
	}{
		{TriggerNone, "none"},
		{TriggerSignal, "signal"},
		{TriggerFileWatch, "file_watch"},
		{TriggerAPI, "api"},
		{TriggerAll, "multiple"},
		{TriggerSignal | TriggerAPI, "multiple"},
		{TriggerFileWatch | TriggerAPI, "multiple"},
	}

	for i, test := range tests {
		result := test.trigger.String()
		if result != test.expected {
			t.Errorf("Test %d: Expected %s for trigger %d, got %s", i, test.expected, test.trigger, result)
		}
	}
}

func TestDefaultReloadConfig(t *testing.T) {
	defaultCfg := DefaultReloadConfig()

	if defaultCfg.Triggers != (TriggerSignal | TriggerFileWatch) {
		t.Error("Default config should enable signal and file watch triggers")
	}

	if !defaultCfg.ValidateBeforeReload {
		t.Error("Default config should enable validation before reload")
	}

	if defaultCfg.ReloadTimeout != 30*time.Second {
		t.Error("Default config should have 30 second reload timeout")
	}

	if !defaultCfg.BackupOnReload {
		t.Error("Default config should enable backup on reload")
	}

	if defaultCfg.BackupRetention != 5 {
		t.Error("Default config should keep 5 backup configurations")
	}

	expectedSignals := []os.Signal{syscall.SIGUSR1, syscall.SIGHUP}
	if len(defaultCfg.Signals) != len(expectedSignals) {
		t.Errorf("Expected %d default signals, got %d", len(expectedSignals), len(defaultCfg.Signals))
	}
}
