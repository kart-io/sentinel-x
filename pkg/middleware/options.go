package middleware

import (
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// Options contains all middleware configuration.
type Options struct {
	// Recovery options
	Recovery        RecoveryOptions
	DisableRecovery bool

	// RequestID options
	RequestID        RequestIDOptions
	DisableRequestID bool

	// Logger options
	Logger        LoggerOptions
	DisableLogger bool

	// CORS options
	CORS        CORSOptions
	DisableCORS bool

	// Timeout options
	Timeout        TimeoutOptions
	DisableTimeout bool

	// Health options
	Health        HealthOptions
	DisableHealth bool

	// Metrics options
	Metrics        MetricsOptions
	DisableMetrics bool

	// Pprof options
	Pprof        PprofOptions
	DisablePprof bool
}

// RecoveryOptions defines recovery middleware options.
type RecoveryOptions struct {
	// EnableStackTrace includes stack trace in error response.
	EnableStackTrace bool
	// OnPanic is called when a panic occurs.
	OnPanic func(ctx transport.Context, err interface{}, stack []byte)
}

// RequestIDOptions defines request ID middleware options.
type RequestIDOptions struct {
	// Header is the header name for request ID.
	Header string
	// Generator is the function to generate request IDs.
	Generator func() string
}

// LoggerOptions defines logger middleware options.
type LoggerOptions struct {
	// SkipPaths is a list of paths to skip logging.
	SkipPaths []string
	// Output is the logger output function.
	Output func(format string, args ...interface{})
}

// CORSOptions defines CORS middleware options.
type CORSOptions struct {
	// AllowOrigins is a list of origins that may access the resource.
	AllowOrigins []string
	// AllowMethods is a list of methods allowed.
	AllowMethods []string
	// AllowHeaders is a list of headers that can be used.
	AllowHeaders []string
	// ExposeHeaders is a list of headers browsers are allowed to access.
	ExposeHeaders []string
	// AllowCredentials indicates whether credentials are allowed.
	AllowCredentials bool
	// MaxAge indicates how long preflight results can be cached.
	MaxAge int
}

// TimeoutOptions defines timeout middleware options.
type TimeoutOptions struct {
	// Timeout is the request timeout duration.
	Timeout time.Duration
	// SkipPaths is a list of paths to skip timeout.
	SkipPaths []string
}

// HealthOptions defines health check options.
type HealthOptions struct {
	// Path is the health check endpoint path.
	Path string
	// LivenessPath is the liveness probe path.
	LivenessPath string
	// ReadinessPath is the readiness probe path.
	ReadinessPath string
	// Checker is a custom health check function.
	Checker func() error
}

// MetricsOptions defines metrics options.
type MetricsOptions struct {
	// Path is the metrics endpoint path.
	Path string
	// Namespace is the metrics namespace.
	Namespace string
	// Subsystem is the metrics subsystem.
	Subsystem string
}

// PprofOptions defines pprof options.
type PprofOptions struct {
	// Prefix is the URL prefix for pprof endpoints.
	// Default: "/debug/pprof"
	Prefix string
	// EnableCmdline enables /debug/pprof/cmdline endpoint.
	EnableCmdline bool
	// EnableProfile enables /debug/pprof/profile endpoint.
	EnableProfile bool
	// EnableSymbol enables /debug/pprof/symbol endpoint.
	EnableSymbol bool
	// EnableTrace enables /debug/pprof/trace endpoint.
	EnableTrace bool
	// BlockProfileRate sets the rate for block profiling.
	// Set to 1 for full profiling, 0 to disable.
	BlockProfileRate int
	// MutexProfileFraction sets the fraction of mutex contention events reported.
	// Set to 1 for full profiling, 0 to disable.
	MutexProfileFraction int
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
