package middleware

import (
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/server/transport"
	"github.com/spf13/pflag"
)

// PathMatcher contains common path matching configuration.
type PathMatcher struct {
	SkipPaths        []string
	SkipPathPrefixes []string
}

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

	// Auth options (JWT authentication)
	Auth        AuthOptions `json:"auth" mapstructure:"auth"`
	DisableAuth bool        `json:"disable-auth" mapstructure:"disable-auth"`

	// Authz options (RBAC authorization)
	Authz        AuthzOptions `json:"authz" mapstructure:"authz"`
	DisableAuthz bool         `json:"disable-authz" mapstructure:"disable-authz"`
}

// RecoveryOptions defines recovery middleware options.
type RecoveryOptions struct {
	EnableStackTrace bool                                                          `json:"enable-stack-trace" mapstructure:"enable-stack-trace"`
	OnPanic          func(ctx transport.Context, err interface{}, stack []byte) `json:"-" mapstructure:"-"`
}

// RequestIDOptions defines request ID middleware options.
type RequestIDOptions struct {
	Header    string          `json:"header" mapstructure:"header"`
	Generator func() string `json:"-" mapstructure:"-"`
}

// LoggerOptions defines logger middleware options.
type LoggerOptions struct {
	SkipPaths           []string                                   `json:"skip-paths" mapstructure:"skip-paths"`
	UseStructuredLogger bool                                       `json:"use-structured-logger" mapstructure:"use-structured-logger"`
	Output              func(format string, args ...interface{}) `json:"-" mapstructure:"-"`
}

// CORSOptions defines CORS middleware options.
type CORSOptions struct {
	AllowOrigins     []string `json:"allow-origins" mapstructure:"allow-origins"`
	AllowMethods     []string `json:"allow-methods" mapstructure:"allow-methods"`
	AllowHeaders     []string `json:"allow-headers" mapstructure:"allow-headers"`
	ExposeHeaders    []string `json:"expose-headers" mapstructure:"expose-headers"`
	AllowCredentials bool     `json:"allow-credentials" mapstructure:"allow-credentials"`
	MaxAge           int      `json:"max-age" mapstructure:"max-age"`
}

// TimeoutOptions defines timeout middleware options.
type TimeoutOptions struct {
	Timeout   time.Duration `json:"timeout" mapstructure:"timeout"`
	SkipPaths []string      `json:"skip-paths" mapstructure:"skip-paths"`
}

// HealthOptions defines health check options.
type HealthOptions struct {
	Path          string         `json:"path" mapstructure:"path"`
	LivenessPath  string         `json:"liveness-path" mapstructure:"liveness-path"`
	ReadinessPath string         `json:"readiness-path" mapstructure:"readiness-path"`
	Checker       func() error `json:"-" mapstructure:"-"`
}

// MetricsOptions defines metrics options.
type MetricsOptions struct {
	Path      string `json:"path" mapstructure:"path"`
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Subsystem string `json:"subsystem" mapstructure:"subsystem"`
}

// PprofOptions defines pprof options.
type PprofOptions struct {
	Prefix               string `json:"prefix" mapstructure:"prefix"`
	EnableCmdline        bool   `json:"enable-cmdline" mapstructure:"enable-cmdline"`
	EnableProfile        bool   `json:"enable-profile" mapstructure:"enable-profile"`
	EnableSymbol         bool   `json:"enable-symbol" mapstructure:"enable-symbol"`
	EnableTrace          bool   `json:"enable-trace" mapstructure:"enable-trace"`
	BlockProfileRate     int    `json:"block-profile-rate" mapstructure:"block-profile-rate"`
	MutexProfileFraction int    `json:"mutex-profile-fraction" mapstructure:"mutex-profile-fraction"`
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
			SkipPaths:           []string{"/health", "/ready", "/live", "/metrics"},
			UseStructuredLogger: true,
			Output:              log.Printf,
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
		DisableAuth:    true, // Auth disabled by default
		DisableAuthz:   true, // Authz disabled by default
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

	// Auth flags
	fs.BoolVar(&o.DisableAuth, "middleware.disable-auth", o.DisableAuth, "Disable authentication middleware")

	// Authz flags
	fs.BoolVar(&o.DisableAuthz, "middleware.disable-authz", o.DisableAuthz, "Disable authorization middleware")
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

// Generic middleware enable/disable functions

// WithRecovery configures recovery middleware.
func WithRecovery(enableStackTrace bool, onPanic func(ctx transport.Context, err interface{}, stack []byte)) Option {
	return func(o *Options) {
		o.DisableRecovery = false
		o.Recovery.EnableStackTrace = enableStackTrace
		if onPanic != nil {
			o.Recovery.OnPanic = onPanic
		}
	}
}

// WithoutRecovery disables recovery middleware.
func WithoutRecovery() Option {
	return func(o *Options) { o.DisableRecovery = true }
}

// WithRequestID configures request ID middleware.
func WithRequestID(header string, generator func() string) Option {
	return func(o *Options) {
		o.DisableRequestID = false
		if header != "" {
			o.RequestID.Header = header
		}
		if generator != nil {
			o.RequestID.Generator = generator
		}
	}
}

// WithoutRequestID disables request ID middleware.
func WithoutRequestID() Option {
	return func(o *Options) { o.DisableRequestID = true }
}

// WithLogger configures logger middleware.
func WithLogger(skipPaths []string, output func(format string, args ...interface{})) Option {
	return func(o *Options) {
		o.DisableLogger = false
		if skipPaths != nil {
			o.Logger.SkipPaths = skipPaths
		}
		if output != nil {
			o.Logger.Output = output
		}
	}
}

// WithoutLogger disables logger middleware.
func WithoutLogger() Option {
	return func(o *Options) { o.DisableLogger = true }
}

// WithCORS configures and enables CORS middleware.
func WithCORS(origins []string, methods []string, headers []string, credentials bool, maxAge int) Option {
	return func(o *Options) {
		o.DisableCORS = false
		if origins != nil {
			o.CORS.AllowOrigins = origins
		}
		if methods != nil {
			o.CORS.AllowMethods = methods
		}
		if headers != nil {
			o.CORS.AllowHeaders = headers
		}
		o.CORS.AllowCredentials = credentials
		if maxAge > 0 {
			o.CORS.MaxAge = maxAge
		}
	}
}

// WithoutCORS disables CORS middleware.
func WithoutCORS() Option {
	return func(o *Options) { o.DisableCORS = true }
}

// WithTimeout configures and enables timeout middleware.
func WithTimeout(timeout time.Duration, skipPaths []string) Option {
	return func(o *Options) {
		o.DisableTimeout = false
		o.Timeout.Timeout = timeout
		if skipPaths != nil {
			o.Timeout.SkipPaths = skipPaths
		}
	}
}

// WithoutTimeout disables timeout middleware.
func WithoutTimeout() Option {
	return func(o *Options) { o.DisableTimeout = true }
}

// WithHealth configures and enables health check endpoints.
func WithHealth(path, livenessPath, readinessPath string, checker func() error) Option {
	return func(o *Options) {
		o.DisableHealth = false
		if path != "" {
			o.Health.Path = path
		}
		if livenessPath != "" {
			o.Health.LivenessPath = livenessPath
		}
		if readinessPath != "" {
			o.Health.ReadinessPath = readinessPath
		}
		if checker != nil {
			o.Health.Checker = checker
		}
	}
}

// WithoutHealth disables health check endpoints.
func WithoutHealth() Option {
	return func(o *Options) { o.DisableHealth = true }
}

// WithMetrics configures and enables metrics endpoint.
func WithMetrics(path, namespace, subsystem string) Option {
	return func(o *Options) {
		o.DisableMetrics = false
		if path != "" {
			o.Metrics.Path = path
		}
		if namespace != "" {
			o.Metrics.Namespace = namespace
		}
		if subsystem != "" {
			o.Metrics.Subsystem = subsystem
		}
	}
}

// WithoutMetrics disables metrics endpoint.
func WithoutMetrics() Option {
	return func(o *Options) { o.DisableMetrics = true }
}

// WithPprof configures and enables pprof endpoints.
func WithPprof(prefix string, blockRate, mutexFraction int) Option {
	return func(o *Options) {
		o.DisablePprof = false
		if prefix != "" {
			o.Pprof.Prefix = prefix
		}
		if blockRate >= 0 {
			o.Pprof.BlockProfileRate = blockRate
		}
		if mutexFraction >= 0 {
			o.Pprof.MutexProfileFraction = mutexFraction
		}
	}
}

// WithoutPprof disables pprof endpoints.
func WithoutPprof() Option {
	return func(o *Options) { o.DisablePprof = true }
}

// Note: WithAuth/WithoutAuth are defined in auth.go
// Note: WithAuthz/WithoutAuthz are defined in authz.go
