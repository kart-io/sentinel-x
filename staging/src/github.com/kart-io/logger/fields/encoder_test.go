package fields

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestDefaultEncoderConfig(t *testing.T) {
	config := DefaultEncoderConfig()

	if config.TimeLayout != time.RFC3339Nano {
		t.Errorf("Expected TimeLayout to be RFC3339Nano, got %s", config.TimeLayout)
	}

	if config.LevelFormatter != LowercaseLevelFormatter {
		t.Errorf("Expected LevelFormatter to be LowercaseLevelFormatter, got %v", config.LevelFormatter)
	}

	if config.CallerFormat != ShortCallerFormatter {
		t.Errorf("Expected CallerFormat to be ShortCallerFormatter, got %v", config.CallerFormat)
	}
}

func TestLevelFormatter(t *testing.T) {
	// Test that formatter constants are defined
	if UppercaseLevelFormatter != 0 {
		t.Errorf("Expected UppercaseLevelFormatter to be 0, got %d", UppercaseLevelFormatter)
	}
	if LowercaseLevelFormatter != 1 {
		t.Errorf("Expected LowercaseLevelFormatter to be 1, got %d", LowercaseLevelFormatter)
	}
}

func TestCallerFormatter(t *testing.T) {
	// Test that formatter constants are defined
	if ShortCallerFormatter != 0 {
		t.Errorf("Expected ShortCallerFormatter to be 0, got %d", ShortCallerFormatter)
	}
	if FullCallerFormatter != 1 {
		t.Errorf("Expected FullCallerFormatter to be 1, got %d", FullCallerFormatter)
	}
}

func TestStandardizedOutput_ToJSON(t *testing.T) {
	tests := []struct {
		name   string
		output StandardizedOutput
		want   map[string]interface{}
	}{
		{
			name: "basic output",
			output: StandardizedOutput{
				Timestamp: "2023-12-01T10:00:00Z",
				Level:     "INFO",
				Message:   "test message",
				Caller:    "main.go:42",
				Fields:    map[string]interface{}{},
			},
			want: map[string]interface{}{
				"timestamp": "2023-12-01T10:00:00Z",
				"level":     "INFO",
				"message":   "test message",
				"caller":    "main.go:42",
			},
		},
		{
			name: "output without caller",
			output: StandardizedOutput{
				Timestamp: "2023-12-01T10:00:00Z",
				Level:     "ERROR",
				Message:   "error occurred",
				Fields:    map[string]interface{}{},
			},
			want: map[string]interface{}{
				"timestamp": "2023-12-01T10:00:00Z",
				"level":     "ERROR",
				"message":   "error occurred",
			},
		},
		{
			name: "output with custom fields",
			output: StandardizedOutput{
				Timestamp: "2023-12-01T10:00:00Z",
				Level:     "DEBUG",
				Message:   "debug info",
				Fields: map[string]interface{}{
					"user_id":    "123",
					"request_id": "req-456",
					"duration":   "100ms",
				},
			},
			want: map[string]interface{}{
				"timestamp":  "2023-12-01T10:00:00Z",
				"level":      "DEBUG",
				"message":    "debug info",
				"user_id":    "123",
				"request_id": "req-456",
				"duration":   "100ms",
			},
		},
		{
			name: "output with tracing fields",
			output: StandardizedOutput{
				Timestamp: "2023-12-01T10:00:00Z",
				Level:     "INFO",
				Message:   "traced operation",
				Fields: map[string]interface{}{
					"trace_id": "trace-123",
					"span_id":  "span-456",
				},
			},
			want: map[string]interface{}{
				"timestamp": "2023-12-01T10:00:00Z",
				"level":     "INFO",
				"message":   "traced operation",
				"trace_id":  "trace-123",
				"span_id":   "span-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := tt.output.ToJSON()
			if err != nil {
				t.Fatalf("ToJSON() error = %v", err)
			}

			var got map[string]interface{}
			err = json.Unmarshal(jsonBytes, &got)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStandardizedOutput_JSONStructure(t *testing.T) {
	// Test that the JSON structure is consistent and follows our field standards
	output := StandardizedOutput{
		Timestamp: "2023-12-01T10:00:00.000Z",
		Level:     "INFO",
		Message:   "test message",
		Caller:    "file.go:123",
		Fields: map[string]interface{}{
			"custom_field": "value",
		},
	}

	jsonBytes, err := output.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	// Check required fields are present
	requiredFields := []string{TimestampField, LevelField, MessageField}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Required field %s missing from JSON output", field)
		}
	}

	// Check that caller field uses standard name
	if caller, exists := result[CallerField]; !exists || caller != "file.go:123" {
		t.Errorf("Caller field not properly set: %v", result[CallerField])
	}

	// Check that custom fields are included
	if customValue, exists := result["custom_field"]; !exists || customValue != "value" {
		t.Errorf("Custom field not included: %v", result["custom_field"])
	}
}

func TestStandardizedOutput_FieldConsistency(t *testing.T) {
	// Test that our standardized output uses the exact field names
	// defined in our constants
	output := StandardizedOutput{
		Timestamp: "2023-12-01T10:00:00Z",
		Level:     "INFO",
		Message:   "consistency test",
	}

	jsonBytes, err := output.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify that the JSON keys match our field constants exactly
	expectedFields := map[string]string{
		TimestampField: "timestamp",
		LevelField:     "level",
		MessageField:   "message",
	}

	for constant, expected := range expectedFields {
		if constant != expected {
			t.Errorf("Field constant %s does not match expected value %s", constant, expected)
		}
		if _, exists := result[constant]; !exists {
			t.Errorf("Field %s not found in JSON output", constant)
		}
	}
}

func TestEncoderConfig_CustomConfig(t *testing.T) {
	// Test that we can create custom encoder configurations
	customConfig := &EncoderConfig{
		TimeLayout:     time.RFC3339,
		LevelFormatter: LowercaseLevelFormatter,
		CallerFormat:   FullCallerFormatter,
	}

	if customConfig.TimeLayout != time.RFC3339 {
		t.Errorf("Custom time layout not set correctly")
	}
	if customConfig.LevelFormatter != LowercaseLevelFormatter {
		t.Errorf("Custom level formatter not set correctly")
	}
	if customConfig.CallerFormat != FullCallerFormatter {
		t.Errorf("Custom caller format not set correctly")
	}
}

func TestStandardizedOutput_EmptyFields(t *testing.T) {
	// Test handling of nil or empty Fields map
	tests := []struct {
		name   string
		fields map[string]interface{}
	}{
		{"nil fields", nil},
		{"empty fields", map[string]interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := StandardizedOutput{
				Timestamp: "2023-12-01T10:00:00Z",
				Level:     "INFO",
				Message:   "test",
				Fields:    tt.fields,
			}

			jsonBytes, err := output.ToJSON()
			if err != nil {
				t.Fatalf("ToJSON() with %s failed: %v", tt.name, err)
			}

			var result map[string]interface{}
			err = json.Unmarshal(jsonBytes, &result)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			// Should still have the core fields
			if len(result) < 3 {
				t.Errorf("Expected at least 3 fields, got %d", len(result))
			}
		})
	}
}
