package core

import "strings"

// Level represents the logging level.
type Level int8

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in production.
	DebugLevel Level = iota - 1
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel
)

// String returns a lower-case ASCII representation of the log level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return "unknown"
	}
}

// ParseLevel parses a level based on the lower-case or all-caps ASCII
// representation of the log level. If the provided ASCII representation
// is invalid an error is returned.
func ParseLevel(text string) (Level, error) {
	switch strings.ToUpper(strings.TrimSpace(text)) {
	case "DEBUG":
		return DebugLevel, nil
	case "INFO":
		return InfoLevel, nil
	case "WARN", "WARNING":
		return WarnLevel, nil
	case "ERROR":
		return ErrorLevel, nil
	case "FATAL":
		return FatalLevel, nil
	default:
		return InfoLevel, &ParseLevelError{text}
	}
}

// ParseLevelError is returned when parsing an invalid level string.
type ParseLevelError struct {
	text string
}

func (e *ParseLevelError) Error() string {
	return "invalid level: " + e.text
}
