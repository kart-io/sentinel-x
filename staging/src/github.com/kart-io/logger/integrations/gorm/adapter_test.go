package gorm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kart-io/logger/core"
)

// Mock logger for testing
type mockLogger struct {
	infoCalls  []mockCall
	warnCalls  []mockCall
	errorCalls []mockCall
}

type mockCall struct {
	msg    string
	fields []interface{}
}

func (m *mockLogger) Debug(args ...interface{})                       {}
func (m *mockLogger) Info(args ...interface{})                        {}
func (m *mockLogger) Warn(args ...interface{})                        {}
func (m *mockLogger) Error(args ...interface{})                       {}
func (m *mockLogger) Fatal(args ...interface{})                       {}
func (m *mockLogger) Debugf(template string, args ...interface{})     {}
func (m *mockLogger) Infof(template string, args ...interface{})      {}
func (m *mockLogger) Warnf(template string, args ...interface{})      {}
func (m *mockLogger) Errorf(template string, args ...interface{})     {}
func (m *mockLogger) Fatalf(template string, args ...interface{})     {}
func (m *mockLogger) Debugw(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Warnw(msg string, keysAndValues ...interface{}) {
	m.warnCalls = append(m.warnCalls, mockCall{msg: msg, fields: keysAndValues})
}

func (m *mockLogger) Errorw(msg string, keysAndValues ...interface{}) {
	m.errorCalls = append(m.errorCalls, mockCall{msg: msg, fields: keysAndValues})
}
func (m *mockLogger) Fatalw(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) SetLevel(level core.Level)                       {}
func (m *mockLogger) Flush() error                                    { return nil }

func (m *mockLogger) Infow(msg string, keysAndValues ...interface{}) {
	m.infoCalls = append(m.infoCalls, mockCall{msg: msg, fields: keysAndValues})
}

func (m *mockLogger) With(keysAndValues ...interface{}) core.Logger {
	return m
}

func (m *mockLogger) WithCtx(ctx context.Context, keysAndValues ...interface{}) core.Logger {
	return m
}

func (m *mockLogger) WithCallerSkip(skip int) core.Logger {
	return m
}

func TestNewGormAdapter(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	if adapter.Name() != "GORM" {
		t.Errorf("Expected name 'GORM', got '%s'", adapter.Name())
	}

	if adapter.Version() != "v1.x" {
		t.Errorf("Expected version 'v1.x', got '%s'", adapter.Version())
	}

	if adapter.GetLogger() != mockLog {
		t.Error("Expected logger to be set correctly")
	}

	if adapter.logLevel != Info {
		t.Errorf("Expected log level Info, got %v", adapter.logLevel)
	}
}

func TestNewGormAdapterWithConfig(t *testing.T) {
	mockLog := &mockLogger{}
	config := Config{
		LogLevel:                  Warn,
		SlowThreshold:             500 * time.Millisecond,
		IgnoreRecordNotFoundError: false,
	}

	adapter := NewGormAdapterWithConfig(mockLog, config)

	if adapter.logLevel != Warn {
		t.Errorf("Expected log level Warn, got %v", adapter.logLevel)
	}

	if adapter.slowThreshold != 500*time.Millisecond {
		t.Errorf("Expected slow threshold 500ms, got %v", adapter.slowThreshold)
	}

	if adapter.ignoreRecordNotFoundError != false {
		t.Error("Expected ignoreRecordNotFoundError to be false")
	}
}

func TestGormAdapter_LogMode(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	newAdapter := adapter.LogMode(Error)

	// Should return a new instance
	if newAdapter == adapter {
		t.Error("Expected LogMode to return a new instance")
	}

	// Check if the log level was changed
	if gormAdapter, ok := newAdapter.(*GormAdapter); ok {
		if gormAdapter.logLevel != Error {
			t.Errorf("Expected log level Error, got %v", gormAdapter.logLevel)
		}
	} else {
		t.Error("Expected LogMode to return a GormAdapter")
	}
}

func TestGormAdapter_Info(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	ctx := context.Background()
	adapter.Info(ctx, "test info message %s", "param1")

	// Since we're using Infof in the implementation, this won't be captured
	// in our mock's Infow calls, but we can verify the method doesn't panic
}

func TestGormAdapter_Warn(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	ctx := context.Background()
	adapter.Warn(ctx, "test warn message %s", "param1")

	// Similar to Info, this uses Warnf internally
}

func TestGormAdapter_Error(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	ctx := context.Background()
	adapter.Error(ctx, "test error message %s", "param1")

	// Similar to Info, this uses Errorf internally
}

func TestGormAdapter_Trace_NormalQuery(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	ctx := context.Background()
	begin := time.Now().Add(-50 * time.Millisecond) // Simulate 50ms query

	adapter.Trace(ctx, begin, func() (string, int64) {
		return "SELECT * FROM users", 10
	}, nil)

	// Should log as info (normal query)
	if len(mockLog.infoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(mockLog.infoCalls))
	}

	if mockLog.infoCalls[0].msg != "Database query executed" {
		t.Errorf("Expected 'Database query executed', got '%s'", mockLog.infoCalls[0].msg)
	}
}

func TestGormAdapter_Trace_SlowQuery(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)
	adapter.SetSlowThreshold(10 * time.Millisecond) // Very low threshold for testing

	ctx := context.Background()
	begin := time.Now().Add(-50 * time.Millisecond) // Simulate 50ms query (slow)

	adapter.Trace(ctx, begin, func() (string, int64) {
		return "SELECT * FROM users", 10
	}, nil)

	// Should log as warning (slow query)
	if len(mockLog.warnCalls) != 1 {
		t.Errorf("Expected 1 warn call, got %d", len(mockLog.warnCalls))
	}

	if mockLog.warnCalls[0].msg != "Slow database query detected" {
		t.Errorf("Expected 'Slow database query detected', got '%s'", mockLog.warnCalls[0].msg)
	}
}

func TestGormAdapter_Trace_ErrorQuery(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	ctx := context.Background()
	begin := time.Now().Add(-50 * time.Millisecond)
	err := errors.New("database connection failed")

	adapter.Trace(ctx, begin, func() (string, int64) {
		return "SELECT * FROM users", 0
	}, err)

	// Should log as error
	if len(mockLog.errorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(mockLog.errorCalls))
	}

	if mockLog.errorCalls[0].msg != "Database query failed" {
		t.Errorf("Expected 'Database query failed', got '%s'", mockLog.errorCalls[0].msg)
	}
}

func TestGormAdapter_Trace_RecordNotFoundError(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	ctx := context.Background()
	begin := time.Now().Add(-50 * time.Millisecond)
	err := errors.New("record not found")

	adapter.Trace(ctx, begin, func() (string, int64) {
		return "SELECT * FROM users WHERE id = 1", 0
	}, err)

	// Should not log error because ignoreRecordNotFoundError is true by default
	if len(mockLog.errorCalls) != 0 {
		t.Errorf("Expected 0 error calls, got %d", len(mockLog.errorCalls))
	}

	// But should still log as info query
	if len(mockLog.infoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(mockLog.infoCalls))
	}
}

func TestGormAdapter_LogQuery(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	adapter.LogQuery("SELECT * FROM users", 1000000, "param1", "value1")

	if len(mockLog.infoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(mockLog.infoCalls))
	}

	fields := mockLog.infoCalls[0].fields
	if len(fields) < 8 { // component, operation, query, duration_ns + 2 custom params
		t.Errorf("Expected at least 8 fields, got %d", len(fields))
	}

	// Check specific fields
	found := false
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) && fields[i] == "component" && fields[i+1] == "gorm" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected component field to be 'gorm'")
	}
}

func TestGormAdapter_LogError(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	err := errors.New("connection failed")
	adapter.LogError(err, "SELECT * FROM users", "param1", "value1")

	if len(mockLog.errorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(mockLog.errorCalls))
	}

	if mockLog.errorCalls[0].msg != "Database query failed" {
		t.Errorf("Expected 'Database query failed', got '%s'", mockLog.errorCalls[0].msg)
	}
}

func TestGormAdapter_LogSlowQuery(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	adapter.LogSlowQuery("SELECT * FROM users", 1000000000, 200000000) // 1s vs 200ms

	if len(mockLog.warnCalls) != 1 {
		t.Errorf("Expected 1 warn call, got %d", len(mockLog.warnCalls))
	}

	if mockLog.warnCalls[0].msg != "Slow database query detected" {
		t.Errorf("Expected 'Slow database query detected', got '%s'", mockLog.warnCalls[0].msg)
	}

	// Check slowdown factor calculation
	fields := mockLog.warnCalls[0].fields
	found := false
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) && fields[i] == "slowdown_factor" {
			if factor, ok := fields[i+1].(float64); ok && factor == 5.0 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Expected slowdown_factor to be 5.0")
	}
}

func TestGormAdapter_ThresholdAndIgnoreSettings(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewGormAdapter(mockLog)

	// Test SetSlowThreshold and GetSlowThreshold
	newThreshold := 500 * time.Millisecond
	adapter.SetSlowThreshold(newThreshold)

	if adapter.GetSlowThreshold() != newThreshold {
		t.Errorf("Expected slow threshold %v, got %v", newThreshold, adapter.GetSlowThreshold())
	}

	// Test SetIgnoreRecordNotFoundError and GetIgnoreRecordNotFoundError
	adapter.SetIgnoreRecordNotFoundError(false)

	if adapter.GetIgnoreRecordNotFoundError() != false {
		t.Error("Expected ignoreRecordNotFoundError to be false")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.LogLevel != Info {
		t.Errorf("Expected log level Info, got %v", config.LogLevel)
	}

	if config.SlowThreshold != 200*time.Millisecond {
		t.Errorf("Expected slow threshold 200ms, got %v", config.SlowThreshold)
	}

	if config.IgnoreRecordNotFoundError != true {
		t.Error("Expected ignoreRecordNotFoundError to be true")
	}
}
