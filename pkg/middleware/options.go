package middleware

import (
	"log"
	"time"

	"github.com/spf13/pflag"

	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// Options contains all middleware configuration.
type Options struct {
	// Recovery options
	Recovery        RecoveryOptions `json:"recovery" mapstructure:"recovery"`
	DisableRecovery bool            `json:"disable-recovery" mapstructure:"disable-recovery"`

	// RequestID options
	RequestID        RequestIDOptions `json:"request-id" mapstructure:"request-id"`
	DisableRequestID bool             `json:"disable-request-id" mapstructure:"disable-request-id"`

	// Logger options
	Logger        LoggerOptions `json:"logger" mapstructure:"logger"`
	DisableLogger bool          `json:"disable-logger" mapstructure:"disable-logger"`

	// CORS options
	CORS        CORSOptions `json:"cors" mapstructure:"cors"`
	DisableCORS bool        `json:"disable-cors" mapstructure:"disable-cors"`

	// Timeout options
	Timeout        TimeoutOptions `json:"timeout" mapstructure:"timeout"`
	DisableTimeout bool           `json:"disable-timeout" mapstructure:"disable-timeout"`

	// Health options
	Health        HealthOptions `json:"health" mapstructure:"health"`
	DisableHealth bool          `json:"disable-health" mapstructure:"disable-health"`

	// Metrics options
	Metrics        MetricsOptions `json:"metrics" mapstructure:"metrics"`
	DisableMetrics bool           `json:"disable-metrics" mapstructure:"disable-metrics"`

	// Pprof options
	Pprof        PprofOptions `json:"pprof" mapstructure:"pprof"`
	DisablePprof bool         `json:"disable-pprof" mapstructure:"disable-pprof"`
}

// RecoveryOptions defines recovery middleware options.
type RecoveryOptions struct {
	// EnableStackTrace includes stack trace in error response.
	EnableStackTrace bool `json:"enable-stack-trace" mapstructure:"enable-stack-trace"`
	// OnPanic is called when a panic occurs (not configurable via config file).
	OnPanic func(ctx transport.Context, err interface{}, stack []byte) `json:"-" mapstructure:"-"`
}

// RequestIDOptions defines request ID middleware options.
type RequestIDOptions struct {
	// Header is the header name for request ID.
	Header string `json:"header" mapstructure:"header"`
	// Generator is the function to generate request IDs (not configurable via config file).
	Generator func() string `json:"-" mapstructure:"-"`
}

// LoggerOptions defines logger middleware options.
type LoggerOptions struct {
	// SkipPaths is a list of paths to skip logging.
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`
	// Output is the logger output function (not configurable via config file).
	Output func(format string, args ...interface{}) `json:"-" mapstructure:"-"`
}

// CORSOptions defines CORS middleware options.
type CORSOptions struct {
	// AllowOrigins is a list of origins that may access the resource.
	AllowOrigins []string `json:"allow-origins" mapstructure:"allow-origins"`
	// AllowMethods is a list of methods allowed.
	AllowMethods []string `json:"allow-methods" mapstructure:"allow-methods"`
	// AllowHeaders is a list of headers that can be used.
	AllowHeaders []string `json:"allow-headers" mapstructure:"allow-headers"`
	// ExposeHeaders is a list of headers browsers are allowed to access.
	ExposeHeaders []string `json:"expose-headers" mapstructure:"expose-headers"`
	// AllowCredentials indicates whether credentials are allowed.
	AllowCredentials bool `json:"allow-credentials" mapstructure:"allow-credentials"`
	// MaxAge indicates how long preflight results can be cached.
	MaxAge int `json:"max-age" mapstructure:"max-age"`
}

// TimeoutOptions defines timeout middleware options.
type TimeoutOptions struct {
	// Timeout is the request timeout duration.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
	// SkipPaths is a list of paths to skip timeout.
	SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`
}

// HealthOptions defines health check options.
type HealthOptions struct {
	// Path is the health check endpoint path.
	Path string `json:"path" mapstructure:"path"`
	// LivenessPath is the liveness probe path.
	LivenessPath string `json:"liveness-path" mapstructure:"liveness-path"`
	// ReadinessPath is the readiness probe path.
	ReadinessPath string `json:"readiness-path" mapstructure:"readiness-path"`
	// Checker is a custom health check function (not configurable via config file).
	Checker func() error `json:"-" mapstructure:"-"`
}

// MetricsOptions defines metrics options.
type MetricsOptions struct {
	// Path is the metrics endpoint path.
	Path string `json:"path" mapstructure:"path"`
	// Namespace is the metrics namespace.
	Namespace string `json:"namespace" mapstructure:"namespace"`
	// Subsystem is the metrics subsystem.
	Subsystem string `json:"subsystem" mapstructure:"subsystem"`
}

// PprofOptions defines pprof options.
type PprofOptions struct {
	// Prefix is the URL prefix for pprof endpoints.
	Prefix string `json:"prefix" mapstructure:"prefix"`
	// EnableCmdline enables /debug/pprof/cmdline endpoint.
	EnableCmdline bool `json:"enable-cmdline" mapstructure:"enable-cmdline"`
	// EnableProfile enables /debug/pprof/profile endpoint.
	EnableProfile bool `json:"enable-profile" mapstructure:"enable-profile"`
	// EnableSymbol enables /debug/pprof/symbol endpoint.
	EnableSymbol bool `json:"enable-symbol" mapstructure:"enable-symbol"`
	// EnableTrace enables /debug/pprof/trace endpoint.
	EnableTrace bool `json:"enable-trace" mapstructure:"enable-trace"`
	// BlockProfileRate sets the rate for block profiling.
	BlockProfileRate int `json:"block-profile-rate" mapstructure:"block-profile-rate"`
	// MutexProfileFraction sets the fraction of mutex contention events reported.
	MutexProfileFraction int `json:"mutex-profile-fraction" mapstructure:"mutex-profile-fraction"`
}

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions creates default middleware options.
func NewOptions() *Options {
	return &Options{
		Recovery: RecoveryOptions{
			EnableStackTrace: false,
		},
		RequestID: RequestIDOptions{
			Header: HeaderXRequestID,
		},
		Logger: LoggerOptions{
			SkipPaths: []string{"/health", "/ready", "/live", "/metrics"},
			Output:    log.Printf,
		},
		CORS: CORSOptions{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
			MaxAge:       86400,
		},
		Timeout: TimeoutOptions{
			Timeout:   30 * time.Second,
			SkipPaths: []string{"/health", "/ready", "/live", "/metrics"},
		},
		Health: HealthOptions{
			Path:          "/health",
			LivenessPath:  "/live",
			ReadinessPath: "/ready",
		},
		Metrics: MetricsOptions{
			Path:      "/metrics",
			Namespace: "sentinel",
			Subsystem: "http",
		},
		Pprof: PprofOptions{
			Prefix:               "/debug/pprof",
			EnableCmdline:        true,
			EnableProfile:        true,
			EnableSymbol:         true,
			EnableTrace:          true,
			BlockProfileRate:     0,
			MutexProfileFraction: 0,
		},
		DisableCORS:    true, // CORS disabled by default
		DisableTimeout: true, // Timeout disabled by default
		DisablePprof:   true, // Pprof disabled by default (security)
	}
}

// AddFlags adds flags for middleware options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	// Recovery flags
	fs.BoolVar(&o.DisableRecovery, "middleware.disable-recovery", o.DisableRecovery, "Disable recovery middleware")
	fs.BoolVar(&o.Recovery.EnableStackTrace, "middleware.recovery.enable-stack-trace", o.Recovery.EnableStackTrace, "Enable stack trace in error responses")

	// RequestID flags
	fs.BoolVar(&o.DisableRequestID, "middleware.disable-request-id", o.DisableRequestID, "Disable request ID middleware")
	fs.StringVar(&o.RequestID.Header, "middleware.request-id.header", o.RequestID.Header, "Request ID header name")

	// Logger flags
	fs.BoolVar(&o.DisableLogger, "middleware.disable-logger", o.DisableLogger, "Disable logger middleware")

	// CORS flags
	fs.BoolVar(&o.DisableCORS, "middleware.disable-cors", o.DisableCORS, "Disable CORS middleware")
	fs.StringSliceVar(&o.CORS.AllowOrigins, "middleware.cors.allow-origins", o.CORS.AllowOrigins, "CORS allowed origins")
	fs.BoolVar(&o.CORS.AllowCredentials, "middleware.cors.allow-credentials", o.CORS.AllowCredentials, "CORS allow credentials")
	fs.IntVar(&o.CORS.MaxAge, "middleware.cors.max-age", o.CORS.MaxAge, "CORS preflight max age")

	// Timeout flags
	fs.BoolVar(&o.DisableTimeout, "middleware.disable-timeout", o.DisableTimeout, "Disable timeout middleware")
	fs.DurationVar(&o.Timeout.Timeout, "middleware.timeout.timeout", o.Timeout.Timeout, "Request timeout duration")

	// Health flags
	fs.BoolVar(&o.DisableHealth, "middleware.disable-health", o.DisableHealth, "Disable health check endpoints")
	fs.StringVar(&o.Health.Path, "middleware.health.path", o.Health.Path, "Health check endpoint path")
	fs.StringVar(&o.Health.LivenessPath, "middleware.health.liveness-path", o.Health.LivenessPath, "Liveness probe path")
	fs.StringVar(&o.Health.ReadinessPath, "middleware.health.readiness-path", o.Health.ReadinessPath, "Readiness probe path")

	// Metrics flags
	fs.BoolVar(&o.DisableMetrics, "middleware.disable-metrics", o.DisableMetrics, "Disable metrics endpoint")
	fs.StringVar(&o.Metrics.Path, "middleware.metrics.path", o.Metrics.Path, "Metrics endpoint path")
	fs.StringVar(&o.Metrics.Namespace, "middleware.metrics.namespace", o.Metrics.Namespace, "Metrics namespace")
	fs.StringVar(&o.Metrics.Subsystem, "middleware.metrics.subsystem", o.Metrics.Subsystem, "Metrics subsystem")

	// Pprof flags
	fs.BoolVar(&o.DisablePprof, "middleware.disable-pprof", o.DisablePprof, "Disable pprof endpoints")
	fs.StringVar(&o.Pprof.Prefix, "middleware.pprof.prefix", o.Pprof.Prefix, "Pprof URL prefix")
	fs.IntVar(&o.Pprof.BlockProfileRate, "middleware.pprof.block-profile-rate", o.Pprof.BlockProfileRate, "Block profile rate")
	fs.IntVar(&o.Pprof.MutexProfileFraction, "middleware.pprof.mutex-profile-fraction", o.Pprof.MutexProfileFraction, "Mutex profile fraction")
}

// Validate validates the middleware options.
func (o *Options) Validate() error {
	return nil
}

// Complete completes the middleware options with defaults.
func (o *Options) Complete() error {
	// Set default logger output if not set
	if o.Logger.Output == nil {
		o.Logger.Output = log.Printf
	}
	return nil
}

// WithRecovery configures recovery middleware.
func WithRecovery(opts ...func(*RecoveryOptions)) Option {
	return func(o *Options) {
		o.DisableRecovery = false
		for _, opt := range opts {
			opt(&o.Recovery)
		}
	}
}

// WithoutRecovery disables recovery middleware.
func WithoutRecovery() Option {
	return func(o *Options) {
		o.DisableRecovery = true
	}
}

// RecoveryWithStackTrace enables stack trace in recovery response.
func RecoveryWithStackTrace() func(*RecoveryOptions) {
	return func(o *RecoveryOptions) {
		o.EnableStackTrace = true
	}
}

// RecoveryWithPanicHandler sets custom panic handler.
func RecoveryWithPanicHandler(handler func(ctx transport.Context, err interface{}, stack []byte)) func(*RecoveryOptions) {
	return func(o *RecoveryOptions) {
		o.OnPanic = handler
	}
}

// WithRequestID configures request ID middleware.
func WithRequestID(opts ...func(*RequestIDOptions)) Option {
	return func(o *Options) {
		o.DisableRequestID = false
		for _, opt := range opts {
			opt(&o.RequestID)
		}
	}
}

// WithoutRequestID disables request ID middleware.
func WithoutRequestID() Option {
	return func(o *Options) {
		o.DisableRequestID = true
	}
}

// RequestIDWithHeader sets custom header name.
func RequestIDWithHeader(header string) func(*RequestIDOptions) {
	return func(o *RequestIDOptions) {
		o.Header = header
	}
}

// RequestIDWithGenerator sets custom ID generator.
func RequestIDWithGenerator(gen func() string) func(*RequestIDOptions) {
	return func(o *RequestIDOptions) {
		o.Generator = gen
	}
}

// WithLogger configures logger middleware.
func WithLogger(opts ...func(*LoggerOptions)) Option {
	return func(o *Options) {
		o.DisableLogger = false
		for _, opt := range opts {
			opt(&o.Logger)
		}
	}
}

// WithoutLogger disables logger middleware.
func WithoutLogger() Option {
	return func(o *Options) {
		o.DisableLogger = true
	}
}

// LoggerWithSkipPaths sets paths to skip logging.
func LoggerWithSkipPaths(paths ...string) func(*LoggerOptions) {
	return func(o *LoggerOptions) {
		o.SkipPaths = paths
	}
}

// LoggerWithOutput sets custom output function.
func LoggerWithOutput(output func(format string, args ...interface{})) func(*LoggerOptions) {
	return func(o *LoggerOptions) {
		o.Output = output
	}
}

// WithCORS configures and enables CORS middleware.
func WithCORS(opts ...func(*CORSOptions)) Option {
	return func(o *Options) {
		o.DisableCORS = false
		for _, opt := range opts {
			opt(&o.CORS)
		}
	}
}

// WithoutCORS disables CORS middleware.
func WithoutCORS() Option {
	return func(o *Options) {
		o.DisableCORS = true
	}
}

// CORSWithOrigins sets allowed origins.
func CORSWithOrigins(origins ...string) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.AllowOrigins = origins
	}
}

// CORSWithMethods sets allowed methods.
func CORSWithMethods(methods ...string) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.AllowMethods = methods
	}
}

// CORSWithHeaders sets allowed headers.
func CORSWithHeaders(headers ...string) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.AllowHeaders = headers
	}
}

// CORSWithCredentials enables credentials.
func CORSWithCredentials() func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.AllowCredentials = true
	}
}

// CORSWithMaxAge sets preflight cache duration.
func CORSWithMaxAge(maxAge int) func(*CORSOptions) {
	return func(o *CORSOptions) {
		o.MaxAge = maxAge
	}
}

// WithTimeout configures and enables timeout middleware.
func WithTimeout(timeout time.Duration, opts ...func(*TimeoutOptions)) Option {
	return func(o *Options) {
		o.DisableTimeout = false
		o.Timeout.Timeout = timeout
		for _, opt := range opts {
			opt(&o.Timeout)
		}
	}
}

// WithoutTimeout disables timeout middleware.
func WithoutTimeout() Option {
	return func(o *Options) {
		o.DisableTimeout = true
	}
}

// TimeoutWithSkipPaths sets paths to skip timeout.
func TimeoutWithSkipPaths(paths ...string) func(*TimeoutOptions) {
	return func(o *TimeoutOptions) {
		o.SkipPaths = paths
	}
}

// WithHealth configures and enables health check endpoints.
func WithHealth(opts ...func(*HealthOptions)) Option {
	return func(o *Options) {
		o.DisableHealth = false
		for _, opt := range opts {
			opt(&o.Health)
		}
	}
}

// WithoutHealth disables health check endpoints.
func WithoutHealth() Option {
	return func(o *Options) {
		o.DisableHealth = true
	}
}

// HealthWithPath sets the main health check path.
func HealthWithPath(path string) func(*HealthOptions) {
	return func(o *HealthOptions) {
		o.Path = path
	}
}

// HealthWithLivenessPath sets the liveness probe path.
func HealthWithLivenessPath(path string) func(*HealthOptions) {
	return func(o *HealthOptions) {
		o.LivenessPath = path
	}
}

// HealthWithReadinessPath sets the readiness probe path.
func HealthWithReadinessPath(path string) func(*HealthOptions) {
	return func(o *HealthOptions) {
		o.ReadinessPath = path
	}
}

// HealthWithChecker sets custom health check function.
func HealthWithChecker(checker func() error) func(*HealthOptions) {
	return func(o *HealthOptions) {
		o.Checker = checker
	}
}

// WithMetrics configures and enables metrics endpoint.
func WithMetrics(opts ...func(*MetricsOptions)) Option {
	return func(o *Options) {
		o.DisableMetrics = false
		for _, opt := range opts {
			opt(&o.Metrics)
		}
	}
}

// WithoutMetrics disables metrics endpoint.
func WithoutMetrics() Option {
	return func(o *Options) {
		o.DisableMetrics = true
	}
}

// MetricsWithPath sets the metrics endpoint path.
func MetricsWithPath(path string) func(*MetricsOptions) {
	return func(o *MetricsOptions) {
		o.Path = path
	}
}

// MetricsWithNamespace sets the metrics namespace.
func MetricsWithNamespace(namespace string) func(*MetricsOptions) {
	return func(o *MetricsOptions) {
		o.Namespace = namespace
	}
}

// MetricsWithSubsystem sets the metrics subsystem.
func MetricsWithSubsystem(subsystem string) func(*MetricsOptions) {
	return func(o *MetricsOptions) {
		o.Subsystem = subsystem
	}
}

// WithPprof configures and enables pprof endpoints.
func WithPprof(opts ...func(*PprofOptions)) Option {
	return func(o *Options) {
		o.DisablePprof = false
		for _, opt := range opts {
			opt(&o.Pprof)
		}
	}
}

// WithoutPprof disables pprof endpoints.
func WithoutPprof() Option {
	return func(o *Options) {
		o.DisablePprof = true
	}
}

// PprofWithPrefix sets the URL prefix for pprof endpoints.
func PprofWithPrefix(prefix string) func(*PprofOptions) {
	return func(o *PprofOptions) {
		o.Prefix = prefix
	}
}

// PprofWithBlockProfileRate sets the block profile rate.
func PprofWithBlockProfileRate(rate int) func(*PprofOptions) {
	return func(o *PprofOptions) {
		o.BlockProfileRate = rate
	}
}

// PprofWithMutexProfileFraction sets the mutex profile fraction.
func PprofWithMutexProfileFraction(fraction int) func(*PprofOptions) {
	return func(o *PprofOptions) {
		o.MutexProfileFraction = fraction
	}
}

// PprofDisableCmdline disables the cmdline endpoint.
func PprofDisableCmdline() func(*PprofOptions) {
	return func(o *PprofOptions) {
		o.EnableCmdline = false
	}
}

// PprofDisableProfile disables the profile endpoint.
func PprofDisableProfile() func(*PprofOptions) {
	return func(o *PprofOptions) {
		o.EnableProfile = false
	}
}

// PprofDisableSymbol disables the symbol endpoint.
func PprofDisableSymbol() func(*PprofOptions) {
	return func(o *PprofOptions) {
		o.EnableSymbol = false
	}
}

// PprofDisableTrace disables the trace endpoint.
func PprofDisableTrace() func(*PprofOptions) {
	return func(o *PprofOptions) {
		o.EnableTrace = false
	}
}
