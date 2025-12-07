package option

import (
	"testing"
	"time"
)

func TestValidation_ConfigConflictResolution(t *testing.T) {
	tests := []struct {
		name        string
		opt         *LogOption
		wantEnabled bool
		wantError   bool
		description string
	}{
		{
			name: "flattened endpoint overrides nested",
			opt: &LogOption{
				Engine:       "slog",
				Level:        "INFO",
				OTLPEndpoint: "http://flattened:4317",
				OTLP: &OTLPOption{
					Endpoint: "http://nested:4318",
					Protocol: "grpc",
				},
			},
			wantEnabled: true,
			wantError:   false,
			description: "Flattened endpoint should take priority and enable OTLP",
		},
		{
			name: "explicit disable overrides endpoint presence",
			opt: &LogOption{
				Engine:       "slog",
				Level:        "INFO",
				OTLPEndpoint: "http://localhost:4317",
				OTLP: &OTLPOption{
					Enabled: boolPtr(false),
				},
			},
			wantEnabled: false,
			wantError:   false,
			description: "Explicit enabled=false should override endpoint presence",
		},
		{
			name: "nested endpoint enables when no flattened",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				OTLP: &OTLPOption{
					Endpoint: "http://nested:4317",
				},
			},
			wantEnabled: true,
			wantError:   false,
			description: "Nested endpoint should enable OTLP when no flattened endpoint",
		},
		{
			name: "no endpoints means disabled",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				OTLP:   &OTLPOption{},
			},
			wantEnabled: false,
			wantError:   false,
			description: "No endpoints should result in disabled OTLP",
		},
		{
			name: "engine validation with fallback",
			opt: &LogOption{
				Engine: "invalid-engine",
				Level:  "INFO",
				OTLP:   &OTLPOption{},
			},
			wantEnabled: false,
			wantError:   false,
			description: "Invalid engine should fallback to slog",
		},
		{
			name: "invalid level should fail validation",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INVALID_LEVEL",
				OTLP:   &OTLPOption{},
			},
			wantEnabled: false,
			wantError:   true,
			description: "Invalid level should return validation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opt.Validate()

			// Check error expectation
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return // Skip further checks if validation failed as expected
			}

			// Check OTLP enabled state
			if got := tt.opt.IsOTLPEnabled(); got != tt.wantEnabled {
				t.Errorf("IsOTLPEnabled() = %v, want %v. %s", got, tt.wantEnabled, tt.description)
			}

			// Check specific behaviors
			switch tt.name {
			case "flattened endpoint overrides nested":
				if tt.opt.OTLP.Endpoint != "http://flattened:4317" {
					t.Errorf("Expected nested endpoint to use flattened value, got %s", tt.opt.OTLP.Endpoint)
				}

			case "engine validation with fallback":
				if tt.opt.Engine != "slog" {
					t.Errorf("Expected invalid engine to fallback to slog, got %s", tt.opt.Engine)
				}

			case "nested endpoint enables when no flattened":
				if tt.opt.OTLP.Enabled == nil || !*tt.opt.OTLP.Enabled {
					t.Error("Expected OTLP to be auto-enabled with nested endpoint")
				}
			}
		})
	}
}

func TestValidation_OTLPDefaults(t *testing.T) {
	tests := []struct {
		name     string
		opt      *LogOption
		expected struct {
			protocol string
			timeout  time.Duration
		}
	}{
		{
			name: "OTLP defaults applied when enabled",
			opt: &LogOption{
				Engine:       "slog",
				Level:        "INFO",
				OTLPEndpoint: "http://localhost:4317",
				OTLP:         &OTLPOption{}, // Empty, should get defaults
			},
			expected: struct {
				protocol string
				timeout  time.Duration
			}{
				protocol: "grpc",
				timeout:  10 * time.Second,
			},
		},
		{
			name: "OTLP preserves user values",
			opt: &LogOption{
				Engine:       "slog",
				Level:        "INFO",
				OTLPEndpoint: "http://localhost:4317",
				OTLP: &OTLPOption{
					Protocol: "http",
					Timeout:  5 * time.Second,
				},
			},
			expected: struct {
				protocol string
				timeout  time.Duration
			}{
				protocol: "http",
				timeout:  5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opt.Validate()
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if !tt.opt.IsOTLPEnabled() {
				t.Fatal("Expected OTLP to be enabled")
			}

			if tt.opt.OTLP.Protocol != tt.expected.protocol {
				t.Errorf("Expected protocol %s, got %s", tt.expected.protocol, tt.opt.OTLP.Protocol)
			}

			if tt.opt.OTLP.Timeout != tt.expected.timeout {
				t.Errorf("Expected timeout %v, got %v", tt.expected.timeout, tt.opt.OTLP.Timeout)
			}
		})
	}
}

func TestValidation_IsOTLPEnabledHelper(t *testing.T) {
	opt := &LogOption{
		Engine:       "slog",
		Level:        "INFO",
		OTLPEndpoint: "http://localhost:4317",
		OTLP:         &OTLPOption{},
	}

	// Before validation, IsOTLPEnabled should return false
	if opt.IsOTLPEnabled() {
		t.Error("Expected IsOTLPEnabled to return false before validation")
	}

	// After validation, it should return true
	err := opt.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !opt.IsOTLPEnabled() {
		t.Error("Expected IsOTLPEnabled to return true after validation")
	}
}

func TestValidation_EdgeCases(t *testing.T) {
	// Test nil OTLP handling
	opt := &LogOption{
		Engine: "slog",
		Level:  "INFO",
		OTLP:   nil,
	}

	err := opt.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if opt.OTLP == nil {
		t.Error("Expected OTLP to be initialized after validation")
	}

	if opt.IsOTLPEnabled() {
		t.Error("Expected OTLP to remain disabled when no endpoint provided")
	}
}
