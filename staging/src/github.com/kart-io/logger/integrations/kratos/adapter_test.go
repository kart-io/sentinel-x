package kratos

import (
	"context"
	"errors"
	"testing"

	"github.com/kart-io/logger/core"
)

// Mock logger for testing Kratos adapter
type mockLogger struct {
	debugCalls []mockCall
	infoCalls  []mockCall
	warnCalls  []mockCall
	errorCalls []mockCall
	fatalCalls []mockCall
}

type mockCall struct {
	msg    string
	fields []interface{}
}

func (m *mockLogger) Debug(args ...interface{})                   {}
func (m *mockLogger) Info(args ...interface{})                    {}
func (m *mockLogger) Warn(args ...interface{})                    {}
func (m *mockLogger) Error(args ...interface{})                   {}
func (m *mockLogger) Fatal(args ...interface{})                   {}
func (m *mockLogger) Debugf(template string, args ...interface{}) {}
func (m *mockLogger) Infof(template string, args ...interface{})  {}
func (m *mockLogger) Warnf(template string, args ...interface{})  {}
func (m *mockLogger) Errorf(template string, args ...interface{}) {}
func (m *mockLogger) Fatalf(template string, args ...interface{}) {}
func (m *mockLogger) SetLevel(level core.Level)                   {}
func (m *mockLogger) Flush() error                                { return nil }

func (m *mockLogger) Debugw(msg string, keysAndValues ...interface{}) {
	m.debugCalls = append(m.debugCalls, mockCall{msg: msg, fields: keysAndValues})
}

func (m *mockLogger) Infow(msg string, keysAndValues ...interface{}) {
	m.infoCalls = append(m.infoCalls, mockCall{msg: msg, fields: keysAndValues})
}

func (m *mockLogger) Warnw(msg string, keysAndValues ...interface{}) {
	m.warnCalls = append(m.warnCalls, mockCall{msg: msg, fields: keysAndValues})
}

func (m *mockLogger) Errorw(msg string, keysAndValues ...interface{}) {
	m.errorCalls = append(m.errorCalls, mockCall{msg: msg, fields: keysAndValues})
}

func (m *mockLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	m.fatalCalls = append(m.fatalCalls, mockCall{msg: msg, fields: keysAndValues})
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

func TestNewKratosAdapter(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	if adapter.Name() != "Kratos" {
		t.Errorf("Expected name 'Kratos', got '%s'", adapter.Name())
	}

	if adapter.Version() != "v2.x" {
		t.Errorf("Expected version 'v2.x', got '%s'", adapter.Version())
	}

	if adapter.GetLogger() != mockLog {
		t.Error("Expected logger to be set correctly")
	}

	if len(adapter.keyvals) != 0 {
		t.Errorf("Expected empty keyvals, got %d items", len(adapter.keyvals))
	}
}

func TestKratosAdapter_Log_Debug(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	err := adapter.Log(LevelDebug, "msg", "debug message", "key", "value")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockLog.debugCalls) != 1 {
		t.Errorf("Expected 1 debug call, got %d", len(mockLog.debugCalls))
	}

	call := mockLog.debugCalls[0]
	if call.msg != "debug message" {
		t.Errorf("Expected message 'debug message', got '%s'", call.msg)
	}

	// Check that component field is present
	found := false
	for i := 0; i < len(call.fields); i += 2 {
		if i+1 < len(call.fields) && call.fields[i] == "component" && call.fields[i+1] == "kratos" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected component field to be 'kratos'")
	}
}

func TestKratosAdapter_Log_Info(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	err := adapter.Log(LevelInfo, "msg", "info message", "request_id", "123")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockLog.infoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(mockLog.infoCalls))
	}

	if mockLog.infoCalls[0].msg != "info message" {
		t.Errorf("Expected message 'info message', got '%s'", mockLog.infoCalls[0].msg)
	}
}

func TestKratosAdapter_Log_Warn(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	err := adapter.Log(LevelWarn, "msg", "warning message")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockLog.warnCalls) != 1 {
		t.Errorf("Expected 1 warn call, got %d", len(mockLog.warnCalls))
	}
}

func TestKratosAdapter_Log_Error(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	err := adapter.Log(LevelError, "msg", "error message", "error", "connection failed")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockLog.errorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(mockLog.errorCalls))
	}
}

func TestKratosAdapter_Log_Fatal(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	err := adapter.Log(LevelFatal, "msg", "fatal message")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockLog.fatalCalls) != 1 {
		t.Errorf("Expected 1 fatal call, got %d", len(mockLog.fatalCalls))
	}
}

func TestKratosAdapter_Log_NoMessage(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	err := adapter.Log(LevelInfo, "key", "value", "another_key", "another_value")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockLog.infoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(mockLog.infoCalls))
	}

	// Should use default message when no msg key is found
	if mockLog.infoCalls[0].msg != "Kratos log message" {
		t.Errorf("Expected default message, got '%s'", mockLog.infoCalls[0].msg)
	}
}

func TestKratosAdapter_With(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	// Add some keyvals with With
	newAdapter := adapter.With("service", "user-service", "version", "1.0.0")

	// Verify it returns a new instance
	if newAdapter == adapter {
		t.Error("Expected With() to return a new adapter instance")
	}

	// Cast to check keyvals
	if kratosAdapter, ok := newAdapter.(*KratosAdapter); ok {
		if len(kratosAdapter.keyvals) != 4 {
			t.Errorf("Expected 4 keyvals, got %d", len(kratosAdapter.keyvals))
		}
	} else {
		t.Error("Expected With() to return a KratosAdapter")
	}

	// Test that new adapter includes both old and new keyvals
	err := newAdapter.Log(LevelInfo, "msg", "test message")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockLog.infoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(mockLog.infoCalls))
	}

	// Check that the persistent keyvals are included
	fields := mockLog.infoCalls[0].fields
	foundService := false
	foundVersion := false

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if fields[i] == "service" && fields[i+1] == "user-service" {
				foundService = true
			}
			if fields[i] == "version" && fields[i+1] == "1.0.0" {
				foundVersion = true
			}
		}
	}

	if !foundService || !foundVersion {
		t.Error("Expected persistent keyvals to be included in log output")
	}
}

func TestKratosAdapter_LogRequest(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	adapter.LogRequest("GET", "/api/users", 200, 1500000000, "user123") // 1.5s

	if len(mockLog.infoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(mockLog.infoCalls))
	}

	call := mockLog.infoCalls[0]
	if call.msg != "HTTP GET /api/users" {
		t.Errorf("Expected 'HTTP GET /api/users', got '%s'", call.msg)
	}

	// Check specific fields
	fields := call.fields
	checks := map[string]interface{}{
		"method":      "GET",
		"path":        "/api/users",
		"status_code": 200,
		"user_id":     "user123",
	}

	for expectedKey, expectedValue := range checks {
		found := false
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) && fields[i] == expectedKey && fields[i+1] == expectedValue {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field %s=%v not found", expectedKey, expectedValue)
		}
	}
}

func TestKratosAdapter_LogRequest_ErrorStatusCode(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	adapter.LogRequest("POST", "/api/users", 500, 1000000000, "") // 500 error, no user ID

	// Should log as error level for 5xx status codes
	if len(mockLog.errorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(mockLog.errorCalls))
	}

	if len(mockLog.infoCalls) != 0 {
		t.Errorf("Expected 0 info calls, got %d", len(mockLog.infoCalls))
	}
}

func TestKratosAdapter_LogRequest_WarnStatusCode(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	adapter.LogRequest("GET", "/api/users/999", 404, 500000000, "user123")

	// Should log as warn level for 4xx status codes
	if len(mockLog.warnCalls) != 1 {
		t.Errorf("Expected 1 warn call, got %d", len(mockLog.warnCalls))
	}
}

func TestKratosAdapter_LogMiddleware(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	adapter.LogMiddleware("auth-middleware", 5000000) // 5ms

	if len(mockLog.debugCalls) != 1 {
		t.Errorf("Expected 1 debug call, got %d", len(mockLog.debugCalls))
	}

	call := mockLog.debugCalls[0]
	if call.msg != "Middleware executed" {
		t.Errorf("Expected 'Middleware executed', got '%s'", call.msg)
	}

	// Check middleware_name field
	found := false
	for i := 0; i < len(call.fields); i += 2 {
		if i+1 < len(call.fields) && call.fields[i] == "middleware_name" && call.fields[i+1] == "auth-middleware" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected middleware_name field to be 'auth-middleware'")
	}
}

func TestKratosAdapter_LogError(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	err := errors.New("database connection failed")
	adapter.LogError(err, "POST", "/api/users", 500)

	if len(mockLog.errorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(mockLog.errorCalls))
	}

	call := mockLog.errorCalls[0]
	if call.msg != "HTTP request failed" {
		t.Errorf("Expected 'HTTP request failed', got '%s'", call.msg)
	}

	// Check error field
	found := false
	for i := 0; i < len(call.fields); i += 2 {
		if i+1 < len(call.fields) && call.fields[i] == "error" && call.fields[i+1] == "database connection failed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected error field to contain error message")
	}
}

func TestKratosAdapter_extractMessage(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	tests := []struct {
		name     string
		keyvals  []interface{}
		expected string
	}{
		{
			name:     "msg key",
			keyvals:  []interface{}{"msg", "test message", "key", "value"},
			expected: "test message",
		},
		{
			name:     "message key",
			keyvals:  []interface{}{"message", "another message", "key", "value"},
			expected: "another message",
		},
		{
			name:     "event key",
			keyvals:  []interface{}{"event", "user login", "user_id", "123"},
			expected: "user login",
		},
		{
			name:     "no message key",
			keyvals:  []interface{}{"key1", "value1", "key2", "value2"},
			expected: "",
		},
		{
			name:     "non-string value",
			keyvals:  []interface{}{"msg", 42, "key", "value"},
			expected: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.extractMessage(tt.keyvals)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestKratosAdapter_getLogLevelForStatusCode(t *testing.T) {
	mockLog := &mockLogger{}
	adapter := NewKratosAdapter(mockLog)

	tests := []struct {
		statusCode int
		expected   core.Level
	}{
		{200, core.InfoLevel},
		{201, core.InfoLevel},
		{301, core.InfoLevel},
		{400, core.WarnLevel},
		{404, core.WarnLevel},
		{499, core.WarnLevel},
		{500, core.ErrorLevel},
		{502, core.ErrorLevel},
		{100, core.InfoLevel}, // default case
	}

	for _, tt := range tests {
		result := adapter.getLogLevelForStatusCode(tt.statusCode)
		if result != tt.expected {
			t.Errorf("For status code %d, expected %v, got %v", tt.statusCode, tt.expected, result)
		}
	}
}
