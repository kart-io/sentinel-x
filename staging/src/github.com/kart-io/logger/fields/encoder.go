package fields

import (
	"encoding/json"
	"time"
)

// EncoderConfig defines how fields should be encoded consistently
// across different logging engines.
type EncoderConfig struct {
	TimeLayout     string
	LevelFormatter LevelFormatter
	CallerFormat   CallerFormatter
}

// LevelFormatter defines how log levels should be formatted.
type LevelFormatter int

const (
	// UppercaseLevelFormatter formats levels as "DEBUG", "INFO", etc.
	UppercaseLevelFormatter LevelFormatter = iota
	// LowercaseLevelFormatter formats levels as "debug", "info", etc.
	LowercaseLevelFormatter
)

// CallerFormatter defines how caller information should be formatted.
type CallerFormatter int

const (
	// ShortCallerFormatter formats as "file.go:123"
	ShortCallerFormatter CallerFormatter = iota
	// FullCallerFormatter formats as "/full/path/file.go:123"
	FullCallerFormatter
)

// DefaultEncoderConfig returns the default encoding configuration.
func DefaultEncoderConfig() *EncoderConfig {
	return &EncoderConfig{
		TimeLayout:     time.RFC3339Nano,
		LevelFormatter: LowercaseLevelFormatter,
		CallerFormat:   ShortCallerFormatter,
	}
}

// StandardizedOutput represents a log entry with standardized field names.
type StandardizedOutput struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:",inline"`
}

// ToJSON converts the standardized output to JSON bytes.
func (so *StandardizedOutput) ToJSON() ([]byte, error) {
	// Merge fields into the main structure for inline JSON output
	output := make(map[string]interface{})
	output[TimestampField] = so.Timestamp
	output[LevelField] = so.Level
	output[MessageField] = so.Message

	if so.Caller != "" {
		output[CallerField] = so.Caller
	}

	// Add custom fields
	for k, v := range so.Fields {
		output[k] = v
	}

	return json.Marshal(output)
}
