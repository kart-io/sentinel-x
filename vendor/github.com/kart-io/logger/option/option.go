package option

import (
	"fmt"
	"time"

	"github.com/kart-io/logger/core"
	"github.com/spf13/pflag"
)

// LogOption represents the complete logger configuration.
type LogOption struct {
	// Engine specifies which logging engine to use ("zap" or "slog")
	Engine string `json:"engine" mapstructure:"engine"`

	// Level sets the minimum logging level
	Level string `json:"level" mapstructure:"level"`

	// Format specifies output format ("json" or "console")
	Format string `json:"format" mapstructure:"format"`

	// OutputPaths specifies where logs should be written
	OutputPaths []string `json:"output_paths" mapstructure:"output_paths"`

	// InitialFields are fields added to every log entry (like service.name, service.version)
	// These fields are added at logger creation time, not from configuration files
	// If service.name or service.version are not provided, they default to "unknown"
	InitialFields map[string]interface{} `json:"-" mapstructure:"-"`

	// OTLP configuration (flattened and nested)
	OTLPEndpoint string      `json:"otlp_endpoint" mapstructure:"otlp_endpoint" yaml:"otlp_endpoint"`
	OTLP         *OTLPOption `json:"otlp" mapstructure:"otlp" yaml:"otlp"`

	// Development mode enables caller info and stacktraces
	Development bool `json:"development" mapstructure:"development"`

	// DisableCaller disables automatic caller detection
	DisableCaller bool `json:"disable_caller" mapstructure:"disable_caller"`

	// DisableStacktrace disables automatic stacktrace capture
	DisableStacktrace bool `json:"disable_stacktrace" mapstructure:"disable_stacktrace"`

	// Rotation configuration for file output (only applies when writing to files, not for OTLP)
	Rotation *RotationOption `json:"rotation" mapstructure:"rotation" yaml:"rotation"`
}

// RotationOption contains log file rotation configuration.
// This configuration only applies to file outputs, not OTLP endpoints.
type RotationOption struct {
	// MaxSize is the maximum size in megabytes of the log file before it gets rotated
	MaxSize int `json:"max_size" mapstructure:"max_size" yaml:"max_size"`

	// MaxAge is the maximum number of days to retain old log files
	MaxAge int `json:"max_age" mapstructure:"max_age" yaml:"max_age"`

	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int `json:"max_backups" mapstructure:"max_backups" yaml:"max_backups"`

	// Compress determines if the rotated log files should be compressed using gzip
	Compress bool `json:"compress" mapstructure:"compress" yaml:"compress"`

	// RotateInterval specifies the rotation interval (e.g., "24h", "7d")
	// This is parsed as time.Duration, common values: "1h", "24h", "7d"
	RotateInterval string `json:"rotate_interval" mapstructure:"rotate_interval" yaml:"rotate_interval"`
}

// OTLPOption contains OTLP-specific configuration.
// ServiceName and ServiceVersion are handled via -ldflags during build time
// using the github.com/kart-io/version package.
type OTLPOption struct {
	Enabled  *bool             `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	Endpoint string            `json:"endpoint" mapstructure:"endpoint" yaml:"endpoint"`
	Protocol string            `json:"protocol" mapstructure:"protocol" yaml:"protocol"`
	Timeout  time.Duration     `json:"timeout" mapstructure:"timeout" yaml:"timeout"`
	Headers  map[string]string `json:"headers" mapstructure:"headers" yaml:"headers"`
	Insecure bool              `json:"insecure" mapstructure:"insecure" yaml:"insecure"`
}

// DefaultLogOption returns a configuration with sensible defaults.
func DefaultLogOption() *LogOption {
	return &LogOption{
		Engine:            "slog",
		Level:             "INFO",
		Format:            "json",
		OutputPaths:       []string{"stdout"},
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		OTLP: &OTLPOption{
			Protocol: "grpc",
			Timeout:  10 * time.Second,
			Insecure: true, // Default to insecure for development
		},
		Rotation: &RotationOption{
			MaxSize:        100,  // 100MB
			MaxAge:         15,   // 15 days
			MaxBackups:     30,   // 30 backup files
			Compress:       true, // Compress old files
			RotateInterval: "7d", // Rotate every 7 days
		},
	}
}

// AddFlags adds configuration flags to the provided pflag.FlagSet.
func (opt *LogOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opt.Engine, "engine", "slog", "Logging engine (zap|slog)")
	fs.StringVar(&opt.Level, "level", "INFO", "Log level (DEBUG|INFO|WARN|ERROR|FATAL)")
	fs.StringVar(&opt.Format, "format", "json", "Log format (json|console)")
	fs.StringSliceVar(&opt.OutputPaths, "output-paths", []string{"stdout"}, "Output paths for logs")
	fs.StringVar(&opt.OTLPEndpoint, "otlp-endpoint", "", "OTLP endpoint URL")
	fs.BoolVar(&opt.Development, "development", false, "Enable development mode")
	fs.BoolVar(&opt.DisableCaller, "disable-caller", false, "Disable caller detection")
	fs.BoolVar(&opt.DisableStacktrace, "disable-stacktrace", false, "Disable stacktrace capture")

	// OTLP nested options
	if opt.OTLP == nil {
		opt.OTLP = &OTLPOption{}
	}
	fs.StringVar(&opt.OTLP.Endpoint, "otlp.endpoint", "", "OTLP nested endpoint URL")
	fs.StringVar(&opt.OTLP.Protocol, "otlp.protocol", "grpc", "OTLP protocol (grpc|http)")
	fs.DurationVar(&opt.OTLP.Timeout, "otlp.timeout", 10*time.Second, "OTLP timeout duration")

	// Rotation options (only applies to file outputs)
	if opt.Rotation == nil {
		opt.Rotation = &RotationOption{}
	}
	fs.IntVar(&opt.Rotation.MaxSize, "rotation.max-size", 100, "Maximum size in MB of the log file before rotation")
	fs.IntVar(&opt.Rotation.MaxAge, "rotation.max-age", 15, "Maximum number of days to retain old log files")
	fs.IntVar(&opt.Rotation.MaxBackups, "rotation.max-backups", 30, "Maximum number of old log files to retain")
	fs.BoolVar(&opt.Rotation.Compress, "rotation.compress", true, "Compress rotated log files using gzip")
	fs.StringVar(&opt.Rotation.RotateInterval, "rotation.rotate-interval", "7d", "Log rotation interval (e.g., 1h, 24h, 7d)")
}

// Validate checks the configuration for consistency and applies intelligent defaults.
func (opt *LogOption) Validate() error {
	if opt == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	// Parse and validate log level
	if _, err := core.ParseLevel(opt.Level); err != nil {
		return err
	}

	// Apply OTLP intelligent configuration resolution
	opt.resolveOTLPConfig()

	// Validate engine selection
	if opt.Engine != "zap" && opt.Engine != "slog" {
		opt.Engine = "slog" // Default fallback
	}

	// Validate rotation configuration
	if err := opt.validateRotationConfig(); err != nil {
		return err
	}

	return nil
}

// resolveOTLPConfig implements the intelligent OTLP configuration resolution
// as specified in the requirements document.
func (opt *LogOption) resolveOTLPConfig() {
	if opt.OTLP == nil {
		opt.OTLP = &OTLPOption{}
	}

	// Check for environment variable override (highest priority)
	// Note: Environment variables are read at startup and require reload for runtime changes

	// Apply flattened configuration logic
	if opt.OTLPEndpoint != "" {
		// If explicit enabled=false is set, respect user intent
		if opt.OTLP.Enabled != nil && !*opt.OTLP.Enabled {
			// User explicitly disabled OTLP, keep disabled
			return
		}

		// Auto-enable OTLP when endpoint is provided (intelligent detection)
		if opt.OTLP.Enabled == nil {
			enabled := true
			opt.OTLP.Enabled = &enabled
		}

		// Use flattened endpoint (priority over nested endpoint)
		opt.OTLP.Endpoint = opt.OTLPEndpoint
	} else {
		// No flattened endpoint, use nested configuration
		if opt.OTLP.Enabled == nil && opt.OTLP.Endpoint != "" {
			// Auto-enable if nested endpoint is provided
			enabled := true
			opt.OTLP.Enabled = &enabled
		}
	}

	// Apply defaults for enabled OTLP
	if opt.OTLP.Enabled != nil && *opt.OTLP.Enabled {
		if opt.OTLP.Protocol == "" {
			opt.OTLP.Protocol = "grpc"
		}
		if opt.OTLP.Timeout == 0 {
			opt.OTLP.Timeout = 10 * time.Second
		}
	}
}

// IsOTLPEnabled returns true if OTLP is enabled after configuration resolution.
func (opt *LogOption) IsOTLPEnabled() bool {
	return opt.OTLP != nil && opt.OTLP.Enabled != nil && *opt.OTLP.Enabled && opt.OTLP.Endpoint != ""
}

// IsEnabled returns true if OTLP is enabled.
func (opt *OTLPOption) IsEnabled() bool {
	return opt != nil && opt.Enabled != nil && *opt.Enabled && opt.Endpoint != ""
}

// WithInitialFields adds or updates fields in InitialFields map.
// These fields will be included in every log entry.
// If InitialFields is nil, it will be initialized.
func (opt *LogOption) WithInitialFields(fields map[string]interface{}) *LogOption {
	if opt.InitialFields == nil {
		opt.InitialFields = make(map[string]interface{})
	}

	for key, value := range fields {
		opt.InitialFields[key] = value
	}

	return opt
}

// AddInitialField adds a single field to InitialFields.
// If InitialFields is nil, it will be initialized.
func (opt *LogOption) AddInitialField(key string, value interface{}) *LogOption {
	if opt.InitialFields == nil {
		opt.InitialFields = make(map[string]interface{})
	}

	opt.InitialFields[key] = value
	return opt
}

// GetInitialFields returns a copy of the InitialFields map.
// Returns empty map if InitialFields is nil.
func (opt *LogOption) GetInitialFields() map[string]interface{} {
	if opt.InitialFields == nil {
		return make(map[string]interface{})
	}

	// Return a copy to prevent external modification
	fields := make(map[string]interface{})
	for key, value := range opt.InitialFields {
		fields[key] = value
	}

	return fields
}

// validateRotationConfig validates log rotation configuration.
// Rotation is only applicable for file outputs, not OTLP endpoints.
func (opt *LogOption) validateRotationConfig() error {
	if opt.Rotation == nil {
		return nil // No rotation config is valid
	}

	rotation := opt.Rotation

	// Validate MaxSize (must be positive)
	if rotation.MaxSize < 0 {
		return fmt.Errorf("rotation max_size must be non-negative, got %d", rotation.MaxSize)
	}

	// Validate MaxAge (must be non-negative)
	if rotation.MaxAge < 0 {
		return fmt.Errorf("rotation max_age must be non-negative, got %d", rotation.MaxAge)
	}

	// Validate MaxBackups (must be non-negative)
	if rotation.MaxBackups < 0 {
		return fmt.Errorf("rotation max_backups must be non-negative, got %d", rotation.MaxBackups)
	}

	// Validate RotateInterval if provided
	if rotation.RotateInterval != "" {
		validFormats := []string{"1h", "24h", "1d", "7d"}
		isValidFormat := false

		// First try parsing as duration
		if _, err := time.ParseDuration(rotation.RotateInterval); err == nil {
			isValidFormat = true
		} else {
			// Check against common formats
			for _, format := range validFormats {
				if rotation.RotateInterval == format {
					isValidFormat = true
					break
				}
			}
		}

		if !isValidFormat {
			return fmt.Errorf("invalid rotation rotate_interval '%s': must be a valid duration (e.g., '1h', '24h') or common format ('1d', '7d')", rotation.RotateInterval)
		}
	}

	// Apply sensible defaults ONLY if values are zero/empty
	// This preserves user-configured non-zero values
	if rotation.MaxSize <= 0 {
		rotation.MaxSize = 100 // 100MB default
	}
	if rotation.MaxAge <= 0 {
		rotation.MaxAge = 15 // 15 days default
	}
	if rotation.MaxBackups <= 0 {
		rotation.MaxBackups = 30 // 30 files default
	}
	if rotation.RotateInterval == "" {
		rotation.RotateInterval = "7d" // 7 days default
	}

	return nil
}

// IsRotationEnabled returns true if log rotation is configured and applicable.
// Rotation only applies when writing to files, not for OTLP-only configurations.
func (opt *LogOption) IsRotationEnabled() bool {
	if opt.Rotation == nil {
		return false
	}

	// Check if any file outputs are configured
	hasFileOutput := false
	for _, path := range opt.OutputPaths {
		if path != "stdout" && path != "stderr" {
			hasFileOutput = true
			break
		}
	}

	return hasFileOutput
}
