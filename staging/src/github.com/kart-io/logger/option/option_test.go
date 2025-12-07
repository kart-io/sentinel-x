package option

import (
	"reflect"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

func TestDefaultLogOption(t *testing.T) {
	opt := DefaultLogOption()

	if opt.Engine != "slog" {
		t.Errorf("Expected Engine to be 'slog', got %s", opt.Engine)
	}

	if opt.Level != "INFO" {
		t.Errorf("Expected Level to be 'INFO', got %s", opt.Level)
	}

	if opt.Format != "json" {
		t.Errorf("Expected Format to be 'json', got %s", opt.Format)
	}

	expectedPaths := []string{"stdout"}
	if !reflect.DeepEqual(opt.OutputPaths, expectedPaths) {
		t.Errorf("Expected OutputPaths to be %v, got %v", expectedPaths, opt.OutputPaths)
	}

	if opt.Development != false {
		t.Errorf("Expected Development to be false, got %t", opt.Development)
	}

	if opt.OTLP == nil {
		t.Fatal("Expected OTLP to be initialized")
	}

	if opt.OTLP.Protocol != "grpc" {
		t.Errorf("Expected OTLP Protocol to be 'grpc', got %s", opt.OTLP.Protocol)
	}

	if opt.OTLP.Timeout != 10*time.Second {
		t.Errorf("Expected OTLP Timeout to be 10s, got %v", opt.OTLP.Timeout)
	}

	// Test Rotation defaults
	if opt.Rotation == nil {
		t.Fatal("Expected Rotation to be initialized")
	}

	if opt.Rotation.MaxSize != 100 {
		t.Errorf("Expected Rotation MaxSize to be 100, got %d", opt.Rotation.MaxSize)
	}

	if opt.Rotation.MaxAge != 15 {
		t.Errorf("Expected Rotation MaxAge to be 15, got %d", opt.Rotation.MaxAge)
	}

	if opt.Rotation.MaxBackups != 30 {
		t.Errorf("Expected Rotation MaxBackups to be 30, got %d", opt.Rotation.MaxBackups)
	}

	if opt.Rotation.Compress != true {
		t.Errorf("Expected Rotation Compress to be true, got %t", opt.Rotation.Compress)
	}

	if opt.Rotation.RotateInterval != "7d" {
		t.Errorf("Expected Rotation RotateInterval to be '7d', got %s", opt.Rotation.RotateInterval)
	}
}

func TestLogOption_AddFlags(t *testing.T) {
	opt := DefaultLogOption()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	opt.AddFlags(fs)

	// Test that flags are registered
	expectedFlags := []string{
		"engine", "level", "format", "output-paths", "otlp-endpoint",
		"development", "disable-caller", "disable-stacktrace",
		"otlp.endpoint", "otlp.protocol", "otlp.timeout",
		"rotation.max-size", "rotation.max-age", "rotation.max-backups",
		"rotation.compress", "rotation.rotate-interval",
	}

	for _, flagName := range expectedFlags {
		if fs.Lookup(flagName) == nil {
			t.Errorf("Flag %s was not registered", flagName)
		}
	}

	// Test flag default values
	if flag := fs.Lookup("engine"); flag.DefValue != "slog" {
		t.Errorf("Expected engine default to be 'slog', got %s", flag.DefValue)
	}

	if flag := fs.Lookup("level"); flag.DefValue != "INFO" {
		t.Errorf("Expected level default to be 'INFO', got %s", flag.DefValue)
	}

	if flag := fs.Lookup("otlp.protocol"); flag.DefValue != "grpc" {
		t.Errorf("Expected otlp.protocol default to be 'grpc', got %s", flag.DefValue)
	}

	// Test rotation flag defaults
	if flag := fs.Lookup("rotation.max-size"); flag.DefValue != "100" {
		t.Errorf("Expected rotation.max-size default to be '100', got %s", flag.DefValue)
	}

	if flag := fs.Lookup("rotation.compress"); flag.DefValue != "true" {
		t.Errorf("Expected rotation.compress default to be 'true', got %s", flag.DefValue)
	}

	if flag := fs.Lookup("rotation.rotate-interval"); flag.DefValue != "7d" {
		t.Errorf("Expected rotation.rotate-interval default to be '7d', got %s", flag.DefValue)
	}
}

func TestLogOption_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opt     *LogOption
		wantErr bool
	}{
		{
			name:    "valid default option",
			opt:     DefaultLogOption(),
			wantErr: false,
		},
		{
			name: "invalid log level",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INVALID",
				Format: "json",
			},
			wantErr: true,
		},
		{
			name: "invalid engine gets corrected",
			opt: &LogOption{
				Engine: "invalid",
				Level:  "INFO",
				Format: "json",
				OTLP:   &OTLPOption{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opt.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check that invalid engine was corrected to slog
			if tt.name == "invalid engine gets corrected" && tt.opt.Engine != "slog" {
				t.Errorf("Expected invalid engine to be corrected to 'slog', got %s", tt.opt.Engine)
			}
		})
	}
}

func TestLogOption_resolveOTLPConfig(t *testing.T) {
	tests := []struct {
		name     string
		opt      *LogOption
		expected bool
	}{
		{
			name: "OTLP enabled with flattened endpoint",
			opt: &LogOption{
				OTLPEndpoint: "http://localhost:4317",
				OTLP:         &OTLPOption{},
			},
			expected: true,
		},
		{
			name: "OTLP enabled with nested endpoint",
			opt: &LogOption{
				OTLP: &OTLPOption{
					Endpoint: "http://localhost:4317",
				},
			},
			expected: true,
		},
		{
			name: "OTLP explicitly disabled",
			opt: &LogOption{
				OTLPEndpoint: "http://localhost:4317",
				OTLP: &OTLPOption{
					Enabled: boolPtr(false),
				},
			},
			expected: false,
		},
		{
			name: "No endpoint provided",
			opt: &LogOption{
				OTLP: &OTLPOption{},
			},
			expected: false,
		},
		{
			name: "Flattened endpoint priority over nested",
			opt: &LogOption{
				OTLPEndpoint: "http://flattened:4317",
				OTLP: &OTLPOption{
					Endpoint: "http://nested:4317",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opt.resolveOTLPConfig()

			if got := tt.opt.IsOTLPEnabled(); got != tt.expected {
				t.Errorf("IsOTLPEnabled() = %v, expected %v", got, tt.expected)
			}

			// Check that defaults are applied when enabled
			if tt.opt.IsOTLPEnabled() {
				if tt.opt.OTLP.Protocol == "" {
					t.Error("Expected OTLP protocol to be set when enabled")
				}
				if tt.opt.OTLP.Timeout == 0 {
					t.Error("Expected OTLP timeout to be set when enabled")
				}
			}

			// Test flattened endpoint priority
			if tt.name == "Flattened endpoint priority over nested" {
				if tt.opt.OTLP.Endpoint != "http://flattened:4317" {
					t.Errorf("Expected nested endpoint to use flattened value, got %s", tt.opt.OTLP.Endpoint)
				}
			}
		})
	}
}

func TestLogOption_IsOTLPEnabled(t *testing.T) {
	tests := []struct {
		name     string
		opt      *LogOption
		expected bool
	}{
		{
			name: "enabled with endpoint",
			opt: &LogOption{
				OTLP: &OTLPOption{
					Enabled:  boolPtr(true),
					Endpoint: "http://localhost:4317",
				},
			},
			expected: true,
		},
		{
			name: "disabled",
			opt: &LogOption{
				OTLP: &OTLPOption{
					Enabled:  boolPtr(false),
					Endpoint: "http://localhost:4317",
				},
			},
			expected: false,
		},
		{
			name: "enabled without endpoint",
			opt: &LogOption{
				OTLP: &OTLPOption{
					Enabled: boolPtr(true),
				},
			},
			expected: false,
		},
		{
			name:     "nil OTLP",
			opt:      &LogOption{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opt.IsOTLPEnabled(); got != tt.expected {
				t.Errorf("IsOTLPEnabled() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestLogOption_StructTags(t *testing.T) {
	// Test that struct fields have correct tags
	optType := reflect.TypeOf(LogOption{})

	tests := []struct {
		fieldName string
		jsonTag   string
		mapTag    string
	}{
		{"Engine", "engine", "engine"},
		{"Level", "level", "level"},
		{"Format", "format", "format"},
		{"OutputPaths", "output_paths", "output_paths"},
		{"OTLPEndpoint", "otlp_endpoint", "otlp_endpoint"},
		{"OTLP", "otlp", "otlp"},
		{"Development", "development", "development"},
		{"DisableCaller", "disable_caller", "disable_caller"},
		{"DisableStacktrace", "disable_stacktrace", "disable_stacktrace"},
		{"Rotation", "rotation", "rotation"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field, found := optType.FieldByName(tt.fieldName)
			if !found {
				t.Fatalf("Field %s not found", tt.fieldName)
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag != tt.jsonTag {
				t.Errorf("Field %s: expected json tag %s, got %s", tt.fieldName, tt.jsonTag, jsonTag)
			}

			mapTag := field.Tag.Get("mapstructure")
			if mapTag != tt.mapTag {
				t.Errorf("Field %s: expected mapstructure tag %s, got %s", tt.fieldName, tt.mapTag, mapTag)
			}
		})
	}
}

// TestRotationOption_Validation tests the rotation configuration validation
func TestRotationOption_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opt     *LogOption
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rotation config",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				Rotation: &RotationOption{
					MaxSize:        50,
					MaxAge:         7,
					MaxBackups:     10,
					Compress:       true,
					RotateInterval: "24h",
				},
			},
			wantErr: false,
		},
		{
			name: "nil rotation config is valid",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
			},
			wantErr: false,
		},
		{
			name: "negative max_size should fail",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				Rotation: &RotationOption{
					MaxSize: -10,
				},
			},
			wantErr: true,
			errMsg:  "rotation max_size must be non-negative",
		},
		{
			name: "negative max_age should fail",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				Rotation: &RotationOption{
					MaxAge: -5,
				},
			},
			wantErr: true,
			errMsg:  "rotation max_age must be non-negative",
		},
		{
			name: "negative max_backups should fail",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				Rotation: &RotationOption{
					MaxBackups: -3,
				},
			},
			wantErr: true,
			errMsg:  "rotation max_backups must be non-negative",
		},
		{
			name: "invalid rotate_interval should fail",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				Rotation: &RotationOption{
					RotateInterval: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "invalid rotation rotate_interval",
		},
		{
			name: "valid common interval formats",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				Rotation: &RotationOption{
					RotateInterval: "1h",
				},
			},
			wantErr: false,
		},
		{
			name: "zero values get defaults",
			opt: &LogOption{
				Engine: "slog",
				Level:  "INFO",
				Rotation: &RotationOption{
					MaxSize:        0,
					MaxAge:         0,
					MaxBackups:     0,
					RotateInterval: "",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opt.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}

			// Test that zero values get default values after validation
			if tt.name == "zero values get defaults" && err == nil {
				rotation := tt.opt.Rotation
				if rotation.MaxSize != 100 {
					t.Errorf("Expected MaxSize to be set to default 100, got %d", rotation.MaxSize)
				}
				if rotation.MaxAge != 15 {
					t.Errorf("Expected MaxAge to be set to default 15, got %d", rotation.MaxAge)
				}
				if rotation.MaxBackups != 30 {
					t.Errorf("Expected MaxBackups to be set to default 30, got %d", rotation.MaxBackups)
				}
				if rotation.RotateInterval != "7d" {
					t.Errorf("Expected RotateInterval to be set to default '7d', got %s", rotation.RotateInterval)
				}
			}
		})
	}
}

// TestLogOption_IsRotationEnabled tests the IsRotationEnabled method
func TestLogOption_IsRotationEnabled(t *testing.T) {
	tests := []struct {
		name     string
		opt      *LogOption
		expected bool
	}{
		{
			name: "nil rotation config",
			opt: &LogOption{
				OutputPaths: []string{"stdout"},
			},
			expected: false,
		},
		{
			name: "rotation config with only stdout output",
			opt: &LogOption{
				OutputPaths: []string{"stdout"},
				Rotation: &RotationOption{
					MaxSize: 100,
				},
			},
			expected: false,
		},
		{
			name: "rotation config with only stderr output",
			opt: &LogOption{
				OutputPaths: []string{"stderr"},
				Rotation: &RotationOption{
					MaxSize: 100,
				},
			},
			expected: false,
		},
		{
			name: "rotation config with file output",
			opt: &LogOption{
				OutputPaths: []string{"./app.log"},
				Rotation: &RotationOption{
					MaxSize: 100,
				},
			},
			expected: true,
		},
		{
			name: "rotation config with mixed outputs",
			opt: &LogOption{
				OutputPaths: []string{"stdout", "./app.log"},
				Rotation: &RotationOption{
					MaxSize: 100,
				},
			},
			expected: true,
		},
		{
			name: "rotation config with multiple file outputs",
			opt: &LogOption{
				OutputPaths: []string{"./app.log", "./error.log"},
				Rotation: &RotationOption{
					MaxSize: 100,
				},
			},
			expected: true,
		},
		{
			name: "no output paths configured",
			opt: &LogOption{
				OutputPaths: []string{},
				Rotation: &RotationOption{
					MaxSize: 100,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opt.IsRotationEnabled(); got != tt.expected {
				t.Errorf("IsRotationEnabled() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// TestRotationOption_Integration tests rotation config in combination with other settings
func TestRotationOption_Integration(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *LogOption
		expectValid bool
		checkAfter  func(t *testing.T, opt *LogOption)
	}{
		{
			name: "rotation with OTLP enabled (both should work)",
			setup: func() *LogOption {
				return &LogOption{
					Engine:       "zap",
					Level:        "INFO",
					OutputPaths:  []string{"./app.log"},
					OTLPEndpoint: "http://localhost:4317",
					Rotation: &RotationOption{
						MaxSize:        50,
						MaxAge:         10,
						MaxBackups:     20,
						Compress:       false,
						RotateInterval: "12h",
					},
					OTLP: &OTLPOption{},
				}
			},
			expectValid: true,
			checkAfter: func(t *testing.T, opt *LogOption) {
				if !opt.IsOTLPEnabled() {
					t.Error("Expected OTLP to be enabled")
				}
				if !opt.IsRotationEnabled() {
					t.Error("Expected rotation to be enabled")
				}
			},
		},
		{
			name: "rotation with console output only",
			setup: func() *LogOption {
				return &LogOption{
					Engine:      "slog",
					Level:       "DEBUG",
					OutputPaths: []string{"stdout", "stderr"},
					Rotation: &RotationOption{
						MaxSize: 100,
					},
				}
			},
			expectValid: true,
			checkAfter: func(t *testing.T, opt *LogOption) {
				if opt.IsRotationEnabled() {
					t.Error("Expected rotation to be disabled for console-only outputs")
				}
			},
		},
		{
			name: "default config has rotation enabled for file outputs",
			setup: func() *LogOption {
				opt := DefaultLogOption()
				opt.OutputPaths = []string{"./test.log"}
				return opt
			},
			expectValid: true,
			checkAfter: func(t *testing.T, opt *LogOption) {
				if !opt.IsRotationEnabled() {
					t.Error("Expected rotation to be enabled for file output")
				}
				if opt.Rotation.MaxSize != 100 {
					t.Errorf("Expected default MaxSize 100, got %d", opt.Rotation.MaxSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := tt.setup()
			err := opt.Validate()

			if (err == nil) != tt.expectValid {
				t.Errorf("Validate() error = %v, expectValid %v", err, tt.expectValid)
			}

			if tt.checkAfter != nil {
				tt.checkAfter(t, opt)
			}
		})
	}
}

// contains checks if a string contains a substring (helper function)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

// findSubstring is a simple substring search helper
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}
