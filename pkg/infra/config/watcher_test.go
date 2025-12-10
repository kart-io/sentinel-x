package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// mockReloadable is a test implementation of the Reloadable interface.
type mockReloadable struct {
	mu            sync.Mutex
	callCount     int
	lastConfig    interface{}
	shouldError   bool
	errorMessage  string
	changeHandler func(interface{}) error
}

func newMockReloadable() *mockReloadable {
	return &mockReloadable{
		callCount: 0,
	}
}

func (m *mockReloadable) OnConfigChange(newConfig interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	m.lastConfig = newConfig

	if m.changeHandler != nil {
		return m.changeHandler(newConfig)
	}

	if m.shouldError {
		return &ConfigError{Message: m.errorMessage}
	}

	return nil
}

func (m *mockReloadable) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func (m *mockReloadable) getLastConfig() interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastConfig
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

// TestNewWatcher verifies that a new watcher is created with correct initial state.
func TestNewWatcher(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	if watcher == nil {
		t.Fatal("NewWatcher returned nil")
	}

	if watcher.viper != v {
		t.Error("Watcher viper instance does not match provided instance")
	}

	if watcher.handlers == nil {
		t.Error("Watcher handlers map is nil")
	}

	if watcher.watching {
		t.Error("Watcher should not be watching initially")
	}

	if count := watcher.HandlerCount(); count != 0 {
		t.Errorf("Expected 0 handlers, got %d", count)
	}
}

// TestSubscribeUnsubscribe tests handler registration and removal.
func TestSubscribeUnsubscribe(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	handler := func(v *viper.Viper) error {
		return nil
	}

	// Test subscribe
	watcher.Subscribe("test-handler", handler)

	if count := watcher.HandlerCount(); count != 1 {
		t.Errorf("Expected 1 handler after subscribe, got %d", count)
	}

	// Test that handler exists
	watcher.mu.RLock()
	if _, exists := watcher.handlers["test-handler"]; !exists {
		t.Error("Handler was not registered")
	}
	watcher.mu.RUnlock()

	// Test unsubscribe
	watcher.Unsubscribe("test-handler")

	if count := watcher.HandlerCount(); count != 0 {
		t.Errorf("Expected 0 handlers after unsubscribe, got %d", count)
	}

	// Test unsubscribe non-existent handler (should not panic)
	watcher.Unsubscribe("non-existent")
}

// TestSubscribeReplacement verifies that subscribing with the same ID replaces the handler.
func TestSubscribeReplacement(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	firstCalled := false
	firstHandler := func(v *viper.Viper) error {
		firstCalled = true
		return nil
	}

	secondCalled := false
	secondHandler := func(v *viper.Viper) error {
		secondCalled = true
		return nil
	}

	watcher.Subscribe("handler", firstHandler)
	watcher.Subscribe("handler", secondHandler)

	if count := watcher.HandlerCount(); count != 1 {
		t.Errorf("Expected 1 handler after replacement, got %d", count)
	}

	// Verify second handler replaced the first
	watcher.mu.RLock()
	handler := watcher.handlers["handler"]
	watcher.mu.RUnlock()

	// Call the handler
	_ = handler(v)

	if firstCalled {
		t.Error("First handler should not be called after replacement")
	}

	if !secondCalled {
		t.Error("Second handler should be called")
	}
}

// TestMultipleSubscribers tests multiple handlers working together.
func TestMultipleSubscribers(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	var callOrder []string
	var mu sync.Mutex

	handler1 := func(v *viper.Viper) error {
		mu.Lock()
		defer mu.Unlock()
		callOrder = append(callOrder, "handler1")
		return nil
	}

	handler2 := func(v *viper.Viper) error {
		mu.Lock()
		defer mu.Unlock()
		callOrder = append(callOrder, "handler2")
		return nil
	}

	handler3 := func(v *viper.Viper) error {
		mu.Lock()
		defer mu.Unlock()
		callOrder = append(callOrder, "handler3")
		return nil
	}

	watcher.Subscribe("h1", handler1)
	watcher.Subscribe("h2", handler2)
	watcher.Subscribe("h3", handler3)

	if count := watcher.HandlerCount(); count != 3 {
		t.Errorf("Expected 3 handlers, got %d", count)
	}
}

// TestIsWatching verifies the watching state.
func TestIsWatching(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	if watcher.IsWatching() {
		t.Error("Watcher should not be watching initially")
	}

	// Note: We can't easily test Start() in unit tests because it requires
	// a valid config file. The IsWatching state is tested separately.
	watcher.mu.Lock()
	watcher.watching = true
	watcher.mu.Unlock()

	if !watcher.IsWatching() {
		t.Error("Watcher should be watching after manual state change")
	}

	watcher.Stop()

	if watcher.IsWatching() {
		t.Error("Watcher should not be watching after Stop()")
	}
}

// TestStartIdempotent verifies that calling Start multiple times is safe.
func TestStartIdempotent(t *testing.T) {
	// Create a temporary config file
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("test: value\n"), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	watcher := NewWatcher(v)

	// Call Start multiple times
	watcher.Start()
	watcher.Start()
	watcher.Start()

	if !watcher.IsWatching() {
		t.Error("Watcher should be watching after Start()")
	}
}

// TestReloadableSubscriber tests the ReloadableSubscriber helper.
func TestReloadableSubscriber(t *testing.T) {
	type TestConfig struct {
		Value  string `mapstructure:"value"`
		Number int    `mapstructure:"number"`
	}

	mock := newMockReloadable()
	target := &TestConfig{}

	subscriber := NewReloadableSubscriber(mock, "test", target)

	if subscriber == nil {
		t.Fatal("NewReloadableSubscriber returned nil")
	}

	if subscriber.component != mock {
		t.Error("Subscriber component does not match")
	}

	if subscriber.configKey != "test" {
		t.Errorf("Expected configKey 'test', got '%s'", subscriber.configKey)
	}

	if subscriber.target != target {
		t.Error("Subscriber target does not match")
	}

	// Test the handler
	v := viper.New()
	v.Set("test.value", "hello")
	v.Set("test.number", 42)

	handler := subscriber.Handler()
	if err := handler(v); err != nil {
		t.Errorf("Handler failed: %v", err)
	}

	if mock.getCallCount() != 1 {
		t.Errorf("Expected 1 call to OnConfigChange, got %d", mock.getCallCount())
	}

	// Verify the target was updated
	if target.Value != "hello" {
		t.Errorf("Expected value 'hello', got '%s'", target.Value)
	}

	if target.Number != 42 {
		t.Errorf("Expected number 42, got %d", target.Number)
	}
}

// TestReloadableSubscriberError verifies error handling.
func TestReloadableSubscriberError(t *testing.T) {
	type TestConfig struct {
		Value string `mapstructure:"value"`
	}

	mock := newMockReloadable()
	mock.shouldError = true
	mock.errorMessage = "config validation failed"

	target := &TestConfig{}
	subscriber := NewReloadableSubscriber(mock, "test", target)

	v := viper.New()
	v.Set("test.value", "hello")

	handler := subscriber.Handler()
	err := handler(v)

	if err == nil {
		t.Error("Expected error from handler, got nil")
	}

	if err.Error() != "component rejected config change: config validation failed" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestReloadableSubscriberUnmarshalError tests unmarshal error handling.
func TestReloadableSubscriberUnmarshalError(t *testing.T) {
	type TestConfig struct {
		Number int `mapstructure:"number"`
	}

	mock := newMockReloadable()
	target := &TestConfig{}
	subscriber := NewReloadableSubscriber(mock, "test", target)

	v := viper.New()
	// Set an invalid type that cannot be unmarshaled to int
	v.Set("test.number", "not-a-number")

	handler := subscriber.Handler()
	err := handler(v)

	if err == nil {
		t.Error("Expected unmarshal error, got nil")
	}

	// The component's OnConfigChange should not be called on unmarshal error
	if mock.getCallCount() != 0 {
		t.Errorf("OnConfigChange should not be called on unmarshal error, got %d calls", mock.getCallCount())
	}
}

// TestConcurrentSubscribe tests thread safety of Subscribe.
func TestConcurrentSubscribe(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			handler := func(v *viper.Viper) error { return nil }
			watcher.Subscribe(string(rune('A'+id%26)), handler)
		}(i)
	}

	wg.Wait()

	// All handlers should be registered (though some may have replaced others due to same ID)
	if count := watcher.HandlerCount(); count == 0 {
		t.Error("No handlers registered after concurrent subscribes")
	}
}

// TestConcurrentUnsubscribe tests thread safety of Unsubscribe.
func TestConcurrentUnsubscribe(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	// Register handlers first
	for i := 0; i < 26; i++ {
		handler := func(v *viper.Viper) error { return nil }
		watcher.Subscribe(string(rune('A'+i)), handler)
	}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			watcher.Unsubscribe(string(rune('A' + id%26)))
		}(i)
	}

	wg.Wait()

	// All handlers should be removed
	if count := watcher.HandlerCount(); count != 0 {
		t.Errorf("Expected 0 handlers after concurrent unsubscribes, got %d", count)
	}
}

// TestConfigFileChange is an integration test that verifies file watching.
// This test creates a real config file and modifies it to trigger the watch callback.
func TestConfigFileChange(t *testing.T) {
	// Create a temporary directory for config file
	tmpDir, err := os.MkdirTemp("", "config-watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.yaml")
	initialConfig := []byte(`
log:
  level: info
  format: json
server:
  timeout: 30s
`)

	if err := os.WriteFile(configFile, initialConfig, 0o644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Setup viper
	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Create watcher
	watcher := NewWatcher(v)

	// Track handler calls
	var handlerCalls int
	var mu sync.Mutex
	done := make(chan struct{})

	handler := func(v *viper.Viper) error {
		mu.Lock()
		defer mu.Unlock()
		handlerCalls++
		if handlerCalls >= 1 {
			close(done)
		}
		return nil
	}

	watcher.Subscribe("test", handler)
	watcher.Start()

	// Give the watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Modify the config file
	updatedConfig := []byte(`
log:
  level: debug
  format: text
server:
  timeout: 60s
`)

	if err := os.WriteFile(configFile, updatedConfig, 0o644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	// Wait for handler to be called (with timeout)
	select {
	case <-done:
		// Handler was called
	case <-time.After(5 * time.Second):
		t.Fatal("Handler was not called within timeout")
	}

	mu.Lock()
	calls := handlerCalls
	mu.Unlock()

	if calls == 0 {
		t.Error("Handler was not called after config file change")
	}

	// Verify the config was actually updated in viper
	if v.GetString("log.level") != "debug" {
		t.Errorf("Expected log.level to be 'debug', got '%s'", v.GetString("log.level"))
	}
}

// BenchmarkSubscribe benchmarks the Subscribe operation.
func BenchmarkSubscribe(b *testing.B) {
	v := viper.New()
	watcher := NewWatcher(v)
	handler := func(v *viper.Viper) error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.Subscribe("bench-handler", handler)
	}
}

// BenchmarkUnsubscribe benchmarks the Unsubscribe operation.
func BenchmarkUnsubscribe(b *testing.B) {
	v := viper.New()
	watcher := NewWatcher(v)
	handler := func(v *viper.Viper) error { return nil }

	// Pre-populate with handlers
	for i := 0; i < 1000; i++ {
		watcher.Subscribe(string(rune(i)), handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.Unsubscribe(string(rune(i % 1000)))
	}
}

// BenchmarkHandlerCount benchmarks the HandlerCount operation.
func BenchmarkHandlerCount(b *testing.B) {
	v := viper.New()
	watcher := NewWatcher(v)
	handler := func(v *viper.Viper) error { return nil }

	// Add some handlers
	for i := 0; i < 100; i++ {
		watcher.Subscribe(string(rune(i)), handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = watcher.HandlerCount()
	}
}
