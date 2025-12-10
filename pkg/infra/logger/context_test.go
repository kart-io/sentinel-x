package logger

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func TestWithRequestID(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		wantField bool
	}{
		{
			name:      "valid request ID",
			requestID: "req-123",
			wantField: true,
		},
		{
			name:      "empty request ID",
			requestID: "",
			wantField: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithRequestID(ctx, tt.requestID)

			fields := GetContextFields(ctx)
			hasRequestID := false
			for i := 0; i < len(fields); i += 2 {
				if fields[i] == "request_id" {
					hasRequestID = true
					if fields[i+1] != tt.requestID {
						t.Errorf("expected request_id=%s, got %s", tt.requestID, fields[i+1])
					}
				}
			}

			if hasRequestID != tt.wantField {
				t.Errorf("hasRequestID=%v, want %v", hasRequestID, tt.wantField)
			}
		})
	}
}

func TestWithTraceID(t *testing.T) {
	tests := []struct {
		name      string
		traceID   string
		wantField bool
	}{
		{
			name:      "valid trace ID",
			traceID:   "trace-456",
			wantField: true,
		},
		{
			name:      "empty trace ID",
			traceID:   "",
			wantField: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithTraceID(ctx, tt.traceID)

			fields := GetContextFields(ctx)
			hasTraceID := false
			for i := 0; i < len(fields); i += 2 {
				if fields[i] == "trace_id" {
					hasTraceID = true
					if fields[i+1] != tt.traceID {
						t.Errorf("expected trace_id=%s, got %s", tt.traceID, fields[i+1])
					}
				}
			}

			if hasTraceID != tt.wantField {
				t.Errorf("hasTraceID=%v, want %v", hasTraceID, tt.wantField)
			}
		})
	}
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user-789"

	ctx = WithUserID(ctx, userID)

	fields := GetContextFields(ctx)
	found := false
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "user_id" {
			found = true
			if fields[i+1] != userID {
				t.Errorf("expected user_id=%s, got %s", userID, fields[i+1])
			}
		}
	}

	if !found {
		t.Error("user_id field not found")
	}
}

func TestWithTenantID(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant-001"

	ctx = WithTenantID(ctx, tenantID)

	fields := GetContextFields(ctx)
	found := false
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "tenant_id" {
			found = true
			if fields[i+1] != tenantID {
				t.Errorf("expected tenant_id=%s, got %s", tenantID, fields[i+1])
			}
		}
	}

	if !found {
		t.Error("tenant_id field not found")
	}
}

func TestWithError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantNil bool
	}{
		{
			name:    "standard error",
			err:     errors.New("test error"),
			wantNil: false,
		},
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithError(ctx, tt.err)

			fields := GetContextFields(ctx)

			if tt.wantNil {
				if len(fields) > 0 {
					t.Error("expected no fields for nil error")
				}
				return
			}

			foundMessage := false
			foundType := false
			for i := 0; i < len(fields); i += 2 {
				if fields[i] == "error_message" {
					foundMessage = true
					if fields[i+1] != tt.err.Error() {
						t.Errorf("expected error_message=%s, got %s", tt.err.Error(), fields[i+1])
					}
				}
				if fields[i] == "error_type" {
					foundType = true
				}
			}

			if !foundMessage {
				t.Error("error_message field not found")
			}
			if !foundType {
				t.Error("error_type field not found")
			}
		})
	}
}

func TestWithFields(t *testing.T) {
	tests := []struct {
		name           string
		keysAndValues  []interface{}
		expectedFields map[string]interface{}
	}{
		{
			name:          "even number of arguments",
			keysAndValues: []interface{}{"key1", "value1", "key2", 42},
			expectedFields: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		},
		{
			name:           "odd number of arguments",
			keysAndValues:  []interface{}{"key1", "value1", "key2"},
			expectedFields: map[string]interface{}{"key1": "value1"},
		},
		{
			name:           "empty arguments",
			keysAndValues:  []interface{}{},
			expectedFields: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithFields(ctx, tt.keysAndValues...)

			fields := GetContextFields(ctx)
			fieldsMap := make(map[string]interface{})
			for i := 0; i < len(fields); i += 2 {
				if key, ok := fields[i].(string); ok {
					fieldsMap[key] = fields[i+1]
				}
			}

			if len(fieldsMap) != len(tt.expectedFields) {
				t.Errorf("expected %d fields, got %d", len(tt.expectedFields), len(fieldsMap))
			}

			for key, expectedValue := range tt.expectedFields {
				if value, ok := fieldsMap[key]; !ok {
					t.Errorf("field %s not found", key)
				} else if value != expectedValue {
					t.Errorf("field %s: expected %v, got %v", key, expectedValue, value)
				}
			}
		})
	}
}

func TestExtractOpenTelemetryFields(t *testing.T) {
	// Create a no-op tracer for testing
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("test-tracer")

	tests := []struct {
		name          string
		setupContext  func() context.Context
		expectTraceID bool
		expectSpanID  bool
	}{
		{
			name: "context with valid span",
			setupContext: func() context.Context {
				ctx := context.Background()
				// Create a mock span context
				ctx, _ = tracer.Start(ctx, "test-span")
				return ctx
			},
			expectTraceID: false, // noop tracer doesn't create valid IDs
			expectSpanID:  false,
		},
		{
			name: "context without span",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectTraceID: false,
			expectSpanID:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			ctx = ExtractOpenTelemetryFields(ctx)

			fields := GetContextFields(ctx)
			hasTraceID := false
			hasSpanID := false

			for i := 0; i < len(fields); i += 2 {
				if fields[i] == "trace_id" {
					hasTraceID = true
				}
				if fields[i] == "span_id" {
					hasSpanID = true
				}
			}

			if hasTraceID != tt.expectTraceID {
				t.Errorf("hasTraceID=%v, want %v", hasTraceID, tt.expectTraceID)
			}
			if hasSpanID != tt.expectSpanID {
				t.Errorf("hasSpanID=%v, want %v", hasSpanID, tt.expectSpanID)
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	// Initialize a test logger
	opts := option.DefaultLogOption()
	opts.Level = "DEBUG"
	opts.Format = "json"
	log, err := logger.New(opts)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	logger.SetGlobal(log)

	t.Run("get logger without context fields", func(t *testing.T) {
		ctx := context.Background()
		log := GetLogger(ctx)
		if log == nil {
			t.Fatal("expected non-nil logger")
		}
	})

	t.Run("get logger with context fields", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithRequestID(ctx, "req-123")
		ctx = WithUserID(ctx, "user-456")

		log := GetLogger(ctx)
		if log == nil {
			t.Fatal("expected non-nil logger")
		}
	})
}

func TestContextLogger(t *testing.T) {
	// Initialize a test logger
	opts := option.DefaultLogOption()
	opts.Level = "DEBUG"
	opts.Format = "json"
	log, err := logger.New(opts)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	logger.SetGlobal(log)

	t.Run("new context logger", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithRequestID(ctx, "req-789")

		cl := NewContextLogger(ctx)
		if cl == nil {
			t.Fatal("expected non-nil context logger")
		}

		if cl.Context() != ctx {
			t.Error("context mismatch")
		}
	})

	t.Run("with context", func(t *testing.T) {
		ctx1 := context.Background()
		ctx1 = WithRequestID(ctx1, "req-001")

		cl := NewContextLogger(ctx1)

		ctx2 := context.Background()
		ctx2 = WithRequestID(ctx2, "req-002")

		cl2 := cl.WithContext(ctx2)
		if cl2.Context() != ctx2 {
			t.Error("context not updated")
		}
	})

	t.Run("with fields", func(t *testing.T) {
		ctx := context.Background()
		cl := NewContextLogger(ctx)

		cl2 := cl.WithFields("key1", "value1", "key2", 42)
		if cl2 == nil {
			t.Fatal("expected non-nil logger")
		}
	})
}

func TestUnwrapError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedCount int
	}{
		{
			name:          "nil error",
			err:           nil,
			expectedCount: 0,
		},
		{
			name:          "single error",
			err:           errors.New("error 1"),
			expectedCount: 1,
		},
		{
			name:          "wrapped error",
			err:           errors.Join(errors.New("error 1"), errors.New("error 2")),
			expectedCount: 1, // errors.Join creates a single error with multiple messages
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := UnwrapError(tt.err)
			if len(messages) != tt.expectedCount {
				t.Errorf("expected %d messages, got %d", tt.expectedCount, len(messages))
			}
		})
	}
}

func TestLoggerFieldsClone(t *testing.T) {
	lf := newLoggerFields()
	lf.set("key1", "value1")
	lf.set("key2", 42)

	clone := lf.clone()

	// Verify clone has same fields
	if len(clone.fields) != len(lf.fields) {
		t.Errorf("expected %d fields in clone, got %d", len(lf.fields), len(clone.fields))
	}

	// Modify clone and verify original is unchanged
	clone.set("key3", "value3")
	if len(lf.fields) == len(clone.fields) {
		t.Error("clone modification affected original")
	}
}

func TestLoggerFieldsToSlice(t *testing.T) {
	lf := newLoggerFields()
	lf.set("key1", "value1")
	lf.set("key2", 42)

	slice := lf.toSlice()

	if len(slice) != 4 { // 2 keys * 2 (key, value)
		t.Errorf("expected 4 elements, got %d", len(slice))
	}

	// Verify fields are present
	fieldsMap := make(map[string]interface{})
	for i := 0; i < len(slice); i += 2 {
		if key, ok := slice[i].(string); ok {
			fieldsMap[key] = slice[i+1]
		}
	}

	if fieldsMap["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %v", fieldsMap["key1"])
	}
	if fieldsMap["key2"] != 42 {
		t.Errorf("expected key2=42, got %v", fieldsMap["key2"])
	}
}

func BenchmarkWithRequestID(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = WithRequestID(ctx, "req-123")
	}
}

func BenchmarkGetLogger(b *testing.B) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GetLogger(ctx)
	}
}

func BenchmarkGetContextFields(b *testing.B) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")
	ctx = WithTraceID(ctx, "trace-789")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GetContextFields(ctx)
	}
}
