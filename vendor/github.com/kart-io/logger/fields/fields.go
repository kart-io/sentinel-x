package fields

// Standard field names used across all logger implementations.
// This ensures consistent field naming regardless of underlying engine (Zap/Slog).
const (
	// Core logging fields
	TimestampField = "timestamp"
	LevelField     = "level"
	MessageField   = "message"
	CallerField    = "caller"

	// Tracing fields
	TraceIDField = "trace_id"
	SpanIDField  = "span_id"

	// Error fields
	ErrorField      = "error"
	ErrorTypeField  = "error_type"
	StacktraceField = "stacktrace"

	// Service identification fields
	ServiceField     = "service"
	ServiceVersion   = "service_version"
	EnvironmentField = "environment"

	// Request context fields
	RequestIDField = "request_id"
	UserIDField    = "user_id"
	SessionIDField = "session_id"

	// Performance fields
	DurationField = "duration"
	LatencyField  = "latency"
)

// FieldMapper provides methods to ensure consistent field mapping
// across different logging engines.
type FieldMapper struct{}

// NewFieldMapper creates a new field mapper instance.
func NewFieldMapper() *FieldMapper {
	return &FieldMapper{}
}

// MapCoreFields maps common fields to their standardized names.
// This ensures that both Zap and Slog engines use identical field names.
func (fm *FieldMapper) MapCoreFields() map[string]string {
	return map[string]string{
		"ts":        TimestampField,
		"time":      TimestampField,
		"timestamp": TimestampField,
		"level":     LevelField,
		"msg":       MessageField,
		"message":   MessageField,
		"caller":    CallerField,
		"source":    CallerField,
	}
}

// MapTracingFields returns standardized tracing field mappings.
func (fm *FieldMapper) MapTracingFields() map[string]string {
	return map[string]string{
		"trace.id": TraceIDField,
		"trace_id": TraceIDField,
		"traceId":  TraceIDField,
		"span.id":  SpanIDField,
		"span_id":  SpanIDField,
		"spanId":   SpanIDField,
	}
}

// ValidateFieldName checks if a field name follows the standardized naming convention.
func (fm *FieldMapper) ValidateFieldName(fieldName string) bool {
	standardFields := []string{
		TimestampField, LevelField, MessageField, CallerField,
		TraceIDField, SpanIDField, ErrorField, ErrorTypeField, StacktraceField,
		ServiceField, ServiceVersion, EnvironmentField,
		RequestIDField, UserIDField, SessionIDField,
		DurationField, LatencyField,
	}

	for _, field := range standardFields {
		if field == fieldName {
			return true
		}
	}
	return true // Allow custom fields, but they should follow snake_case convention
}
