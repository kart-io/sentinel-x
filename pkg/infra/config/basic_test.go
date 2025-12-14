package config

import (
	"testing"

	"github.com/spf13/viper"
)

// TestReloadableInterface verifies the Reloadable interface exists.
func TestReloadableInterface(_ *testing.T) {
	var _ Reloadable = (*simpleReloadable)(nil)
}

type simpleReloadable struct {
	called bool
}

func (m *simpleReloadable) OnConfigChange(_ interface{}) error {
	m.called = true
	return nil
}

// TestWatcherCreation tests basic watcher creation.
func TestWatcherCreation(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	if watcher == nil {
		t.Fatal("NewWatcher returned nil")
	}

	if watcher.viper != v {
		t.Error("Watcher viper instance does not match")
	}

	if watcher.watching {
		t.Error("Watcher should not be watching initially")
	}

	if count := watcher.HandlerCount(); count != 0 {
		t.Errorf("Expected 0 handlers initially, got %d", count)
	}
}

// TestSubscribeBasic tests basic subscription.
func TestSubscribeBasic(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	handler := func(_ *viper.Viper) error {
		return nil
	}

	watcher.Subscribe("test", handler)

	if count := watcher.HandlerCount(); count != 1 {
		t.Errorf("Expected 1 handler, got %d", count)
	}
}

// TestUnsubscribeBasic tests basic unsubscription.
func TestUnsubscribeBasic(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	handler := func(_ *viper.Viper) error {
		return nil
	}

	watcher.Subscribe("test", handler)
	watcher.Unsubscribe("test")

	if count := watcher.HandlerCount(); count != 0 {
		t.Errorf("Expected 0 handlers after unsubscribe, got %d", count)
	}
}

// TestUnsubscribeNonExistent tests unsubscribing a non-existent handler.
func TestUnsubscribeNonExistent(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	// Should not panic
	watcher.Unsubscribe("non-existent")

	if count := watcher.HandlerCount(); count != 0 {
		t.Errorf("Expected 0 handlers, got %d", count)
	}
}

// TestMultipleHandlers tests multiple handler registration.
func TestMultipleHandlers(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	handler1 := func(_ *viper.Viper) error { return nil }
	handler2 := func(_ *viper.Viper) error { return nil }
	handler3 := func(_ *viper.Viper) error { return nil }

	watcher.Subscribe("h1", handler1)
	watcher.Subscribe("h2", handler2)
	watcher.Subscribe("h3", handler3)

	if count := watcher.HandlerCount(); count != 3 {
		t.Errorf("Expected 3 handlers, got %d", count)
	}

	watcher.Unsubscribe("h2")

	if count := watcher.HandlerCount(); count != 2 {
		t.Errorf("Expected 2 handlers after one unsubscribe, got %d", count)
	}
}

// TestHandlerReplacement tests that subscribing with same ID replaces handler.
func TestHandlerReplacement(t *testing.T) {
	v := viper.New()
	watcher := NewWatcher(v)

	handler1 := func(_ *viper.Viper) error { return nil }
	handler2 := func(_ *viper.Viper) error { return nil }

	watcher.Subscribe("handler", handler1)
	watcher.Subscribe("handler", handler2)

	if count := watcher.HandlerCount(); count != 1 {
		t.Errorf("Expected 1 handler after replacement, got %d", count)
	}
}

// TestReloadableSubscriberCreation tests ReloadableSubscriber creation.
func TestReloadableSubscriberCreation(t *testing.T) {
	mock := &simpleReloadable{}
	target := make(map[string]interface{})

	subscriber := NewReloadableSubscriber(mock, "test.key", target)

	if subscriber == nil {
		t.Fatal("NewReloadableSubscriber returned nil")
	}

	if subscriber.component != mock {
		t.Error("Subscriber component mismatch")
	}

	if subscriber.configKey != "test.key" {
		t.Errorf("Expected configKey 'test.key', got '%s'", subscriber.configKey)
	}
}

// TestReloadableSubscriberHandler tests the handler creation.
func TestReloadableSubscriberHandler(t *testing.T) {
	type TestConfig struct {
		Name  string `mapstructure:"name"`
		Value int    `mapstructure:"value"`
	}

	mock := &simpleReloadable{}
	target := &TestConfig{}

	subscriber := NewReloadableSubscriber(mock, "test", target)
	handler := subscriber.Handler()

	if handler == nil {
		t.Fatal("Handler is nil")
	}

	// Create viper instance with test data
	v := viper.New()
	v.Set("test.name", "example")
	v.Set("test.value", 42)

	// Call the handler
	err := handler(v)
	if err != nil {
		t.Errorf("Handler failed: %v", err)
	}

	// Verify the component was called
	if !mock.called {
		t.Error("Component OnConfigChange was not called")
	}

	// Verify target was populated
	if target.Name != "example" {
		t.Errorf("Expected name 'example', got '%s'", target.Name)
	}

	if target.Value != 42 {
		t.Errorf("Expected value 42, got %d", target.Value)
	}
}
