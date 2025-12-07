package option

import (
	"reflect"
	"testing"
)

func TestLogOption_WithInitialFields(t *testing.T) {
	opt := &LogOption{}

	fields := map[string]interface{}{
		"service.name":    "test-service",
		"service.version": "v1.0.0",
		"environment":     "test",
	}

	result := opt.WithInitialFields(fields)

	// Should return the same instance (fluent interface)
	if result != opt {
		t.Error("WithInitialFields should return the same LogOption instance")
	}

	// Check that fields were added
	if !reflect.DeepEqual(opt.InitialFields, fields) {
		t.Errorf("Expected InitialFields to be %v, got %v", fields, opt.InitialFields)
	}
}

func TestLogOption_WithInitialFields_NilInitialFields(t *testing.T) {
	opt := &LogOption{
		InitialFields: nil,
	}

	fields := map[string]interface{}{
		"service.name": "test-service",
	}

	opt.WithInitialFields(fields)

	if opt.InitialFields == nil {
		t.Error("InitialFields should be initialized when it was nil")
	}

	if opt.InitialFields["service.name"] != "test-service" {
		t.Errorf("Expected service.name to be 'test-service', got %v", opt.InitialFields["service.name"])
	}
}

func TestLogOption_WithInitialFields_MergeFields(t *testing.T) {
	opt := &LogOption{
		InitialFields: map[string]interface{}{
			"service.name": "old-service",
			"environment":  "dev",
		},
	}

	newFields := map[string]interface{}{
		"service.name":    "new-service", // Should override
		"service.version": "v2.0.0",      // Should add
	}

	opt.WithInitialFields(newFields)

	expected := map[string]interface{}{
		"service.name":    "new-service",
		"service.version": "v2.0.0",
		"environment":     "dev", // Should remain
	}

	if !reflect.DeepEqual(opt.InitialFields, expected) {
		t.Errorf("Expected InitialFields to be %v, got %v", expected, opt.InitialFields)
	}
}

func TestLogOption_AddInitialField(t *testing.T) {
	opt := &LogOption{}

	result := opt.AddInitialField("service.name", "test-service")

	// Should return the same instance (fluent interface)
	if result != opt {
		t.Error("AddInitialField should return the same LogOption instance")
	}

	// Check that field was added
	if opt.InitialFields["service.name"] != "test-service" {
		t.Errorf("Expected service.name to be 'test-service', got %v", opt.InitialFields["service.name"])
	}
}

func TestLogOption_AddInitialField_NilInitialFields(t *testing.T) {
	opt := &LogOption{
		InitialFields: nil,
	}

	opt.AddInitialField("service.name", "test-service")

	if opt.InitialFields == nil {
		t.Error("InitialFields should be initialized when it was nil")
	}

	if opt.InitialFields["service.name"] != "test-service" {
		t.Errorf("Expected service.name to be 'test-service', got %v", opt.InitialFields["service.name"])
	}
}

func TestLogOption_AddInitialField_Chaining(t *testing.T) {
	opt := &LogOption{}

	result := opt.AddInitialField("service.name", "test-service").
		AddInitialField("service.version", "v1.0.0").
		AddInitialField("environment", "test")

	// Should return the same instance (fluent interface)
	if result != opt {
		t.Error("AddInitialField chaining should return the same LogOption instance")
	}

	expected := map[string]interface{}{
		"service.name":    "test-service",
		"service.version": "v1.0.0",
		"environment":     "test",
	}

	if !reflect.DeepEqual(opt.InitialFields, expected) {
		t.Errorf("Expected InitialFields to be %v, got %v", expected, opt.InitialFields)
	}
}

func TestLogOption_GetInitialFields(t *testing.T) {
	opt := &LogOption{
		InitialFields: map[string]interface{}{
			"service.name":    "test-service",
			"service.version": "v1.0.0",
		},
	}

	fields := opt.GetInitialFields()

	expected := map[string]interface{}{
		"service.name":    "test-service",
		"service.version": "v1.0.0",
	}

	if !reflect.DeepEqual(fields, expected) {
		t.Errorf("Expected GetInitialFields to return %v, got %v", expected, fields)
	}

	// Modify the returned map to test that it's a copy
	fields["modified"] = "value"

	// Original should not be modified
	if _, exists := opt.InitialFields["modified"]; exists {
		t.Error("GetInitialFields should return a copy, original InitialFields was modified")
	}
}

func TestLogOption_GetInitialFields_NilInitialFields(t *testing.T) {
	opt := &LogOption{
		InitialFields: nil,
	}

	fields := opt.GetInitialFields()

	if fields == nil {
		t.Error("GetInitialFields should return empty map when InitialFields is nil, not nil")
	}

	if len(fields) != 0 {
		t.Errorf("Expected empty map, got %v", fields)
	}
}

func TestLogOption_InitialFields_Integration(t *testing.T) {
	// Test a typical usage scenario
	opt := DefaultLogOption()

	// Add service information
	opt.WithInitialFields(map[string]interface{}{
		"service.name":    "my-service",
		"service.version": "v1.2.3",
	}).AddInitialField("environment", "production").
		AddInitialField("datacenter", "us-west-2")

	fields := opt.GetInitialFields()

	expected := map[string]interface{}{
		"service.name":    "my-service",
		"service.version": "v1.2.3",
		"environment":     "production",
		"datacenter":      "us-west-2",
	}

	if !reflect.DeepEqual(fields, expected) {
		t.Errorf("Expected integrated usage to result in %v, got %v", expected, fields)
	}

	// Test that original option has the fields
	if !reflect.DeepEqual(opt.InitialFields, expected) {
		t.Errorf("Expected original InitialFields to be %v, got %v", expected, opt.InitialFields)
	}
}
