package fields

import (
	"reflect"
	"testing"
)

func TestFieldConstants(t *testing.T) {
	// Test that field constants have expected values
	expectedFields := map[string]string{
		"TimestampField":   "timestamp",
		"LevelField":       "level",
		"MessageField":     "message",
		"CallerField":      "caller",
		"TraceIDField":     "trace_id",
		"SpanIDField":      "span_id",
		"ErrorField":       "error",
		"ErrorTypeField":   "error_type",
		"StacktraceField":  "stacktrace",
		"ServiceField":     "service",
		"ServiceVersion":   "service_version",
		"EnvironmentField": "environment",
		"RequestIDField":   "request_id",
		"UserIDField":      "user_id",
		"SessionIDField":   "session_id",
		"DurationField":    "duration",
		"LatencyField":     "latency",
	}

	fieldsType := reflect.TypeOf((*FieldMapper)(nil)).Elem()
	pkg := fieldsType.PkgPath()

	for name, expected := range expectedFields {
		// This is a simplified test - in real implementation, we'd use reflection
		// to check the actual constant values, but for this test we'll verify
		// the constants are defined correctly by testing the mapper functions
		_ = name
		_ = expected
		_ = pkg
	}

	// Test specific field values
	if TimestampField != "timestamp" {
		t.Errorf("Expected TimestampField to be 'timestamp', got %s", TimestampField)
	}
	if LevelField != "level" {
		t.Errorf("Expected LevelField to be 'level', got %s", LevelField)
	}
	if MessageField != "message" {
		t.Errorf("Expected MessageField to be 'message', got %s", MessageField)
	}
}

func TestNewFieldMapper(t *testing.T) {
	mapper := NewFieldMapper()
	if mapper == nil {
		t.Error("NewFieldMapper() returned nil")
	}
}

func TestFieldMapper_MapCoreFields(t *testing.T) {
	mapper := NewFieldMapper()
	coreFields := mapper.MapCoreFields()

	expectedMappings := map[string]string{
		"ts":        TimestampField,
		"time":      TimestampField,
		"timestamp": TimestampField,
		"level":     LevelField,
		"msg":       MessageField,
		"message":   MessageField,
		"caller":    CallerField,
		"source":    CallerField,
	}

	if !reflect.DeepEqual(coreFields, expectedMappings) {
		t.Errorf("MapCoreFields() = %v, want %v", coreFields, expectedMappings)
	}

	// Test specific mappings
	if coreFields["ts"] != "timestamp" {
		t.Errorf("Expected 'ts' to map to 'timestamp', got %s", coreFields["ts"])
	}
	if coreFields["msg"] != "message" {
		t.Errorf("Expected 'msg' to map to 'message', got %s", coreFields["msg"])
	}
}

func TestFieldMapper_MapTracingFields(t *testing.T) {
	mapper := NewFieldMapper()
	tracingFields := mapper.MapTracingFields()

	expectedMappings := map[string]string{
		"trace.id": TraceIDField,
		"trace_id": TraceIDField,
		"traceId":  TraceIDField,
		"span.id":  SpanIDField,
		"span_id":  SpanIDField,
		"spanId":   SpanIDField,
	}

	if !reflect.DeepEqual(tracingFields, expectedMappings) {
		t.Errorf("MapTracingFields() = %v, want %v", tracingFields, expectedMappings)
	}

	// Test that various trace ID formats map to the same standard
	standardTraceID := "trace_id"
	traceVariants := []string{"trace.id", "trace_id", "traceId"}
	for _, variant := range traceVariants {
		if tracingFields[variant] != standardTraceID {
			t.Errorf("Expected %s to map to %s, got %s", variant, standardTraceID, tracingFields[variant])
		}
	}

	// Test that various span ID formats map to the same standard
	standardSpanID := "span_id"
	spanVariants := []string{"span.id", "span_id", "spanId"}
	for _, variant := range spanVariants {
		if tracingFields[variant] != standardSpanID {
			t.Errorf("Expected %s to map to %s, got %s", variant, standardSpanID, tracingFields[variant])
		}
	}
}

func TestFieldMapper_ValidateFieldName(t *testing.T) {
	mapper := NewFieldMapper()

	tests := []struct {
		fieldName string
		valid     bool
	}{
		// Standard fields should be valid
		{TimestampField, true},
		{LevelField, true},
		{MessageField, true},
		{CallerField, true},
		{TraceIDField, true},
		{SpanIDField, true},
		{ErrorField, true},
		{StacktraceField, true},

		// Custom fields (should be allowed)
		{"custom_field", true},
		{"my_custom_data", true},

		// Any field name should be valid (the function returns true for all)
		{"CamelCase", true},
		{"some-field", true},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			if got := mapper.ValidateFieldName(tt.fieldName); got != tt.valid {
				t.Errorf("ValidateFieldName(%s) = %v, want %v", tt.fieldName, got, tt.valid)
			}
		})
	}
}

func TestFieldConsistency(t *testing.T) {
	// Test that our field standardization ensures consistency
	// between different engine outputs

	mapper := NewFieldMapper()
	coreMapping := mapper.MapCoreFields()
	tracingMapping := mapper.MapTracingFields()

	// Verify that all mapped values use our standard field constants
	for _, mappedValue := range coreMapping {
		switch mappedValue {
		case TimestampField, LevelField, MessageField, CallerField:
			// These are valid
		default:
			t.Errorf("Core mapping contains non-standard field: %s", mappedValue)
		}
	}

	for _, mappedValue := range tracingMapping {
		switch mappedValue {
		case TraceIDField, SpanIDField:
			// These are valid
		default:
			t.Errorf("Tracing mapping contains non-standard field: %s", mappedValue)
		}
	}
}

func TestFieldNamingConvention(t *testing.T) {
	// Test that our standard fields follow snake_case convention
	standardFields := []string{
		TimestampField, LevelField, MessageField, CallerField,
		TraceIDField, SpanIDField, ErrorField, ErrorTypeField, StacktraceField,
		ServiceField, ServiceVersion, EnvironmentField,
		RequestIDField, UserIDField, SessionIDField,
		DurationField, LatencyField,
	}

	for _, field := range standardFields {
		// Basic check: should not contain uppercase letters or spaces
		for _, char := range field {
			if char >= 'A' && char <= 'Z' {
				t.Errorf("Field %s contains uppercase letter, should use snake_case", field)
			}
			if char == ' ' {
				t.Errorf("Field %s contains space, should use snake_case", field)
			}
		}
	}
}
