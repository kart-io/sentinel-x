// Package options 提供中间件配置选项。
package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware/common"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
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

// Option is a function that configures Options.
type Option func(*Options)

// NewOptions creates default middleware options.
func NewOptions() *Options {
	return &Options{
		Recovery: RecoveryOptions{
			EnableStackTrace: false,
		},
		RequestID: RequestIDOptions{
			Header: common.HeaderXRequestID,
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
		Auth: AuthOptions{
			TokenLookup:      "header:Authorization",
			AuthScheme:       "Bearer",
			SkipPaths:        []string{},
			SkipPathPrefixes: []string{},
		},
		Authz: AuthzOptions{
			SkipPaths:        []string{},
			SkipPathPrefixes: []string{},
		},
		DisableCORS:    true,  // CORS disabled by default
		DisableTimeout: true,  // Timeout disabled by default
		DisablePprof:   true,  // Pprof disabled by default (security)
		DisableAuth:    false, // Auth enabled by default (security)
		DisableAuthz:   false, // Authz enabled by default (security)
	}
}

// Validate validates the middleware options.
func (o *Options) Validate() error {
	var errs []error

	// Validate timeout configuration
	if !o.DisableTimeout {
		if o.Timeout.Timeout <= 0 {
			errs = append(errs, fmt.Errorf("timeout must be positive when timeout middleware is enabled"))
		}
	}

	// Validate CORS configuration
	if !o.DisableCORS {
		if len(o.CORS.AllowOrigins) == 0 {
			errs = append(errs, fmt.Errorf("cors: at least one allowed origin must be specified when CORS is enabled"))
		}
		if len(o.CORS.AllowMethods) == 0 {
			errs = append(errs, fmt.Errorf("cors: at least one allowed method must be specified when CORS is enabled"))
		}
		if o.CORS.MaxAge < 0 {
			errs = append(errs, fmt.Errorf("cors: max age cannot be negative"))
		}
	}

	// Validate authentication configuration
	if !o.DisableAuth {
		if o.Auth.TokenLookup == "" {
			errs = append(errs, fmt.Errorf("auth: token lookup configuration is required"))
		}
		if o.Auth.AuthScheme == "" {
			errs = append(errs, fmt.Errorf("auth: auth scheme is required"))
		}
	}

	// Validate health check paths
	if !o.DisableHealth {
		if o.Health.Path == "" && o.Health.LivenessPath == "" && o.Health.ReadinessPath == "" {
			errs = append(errs, fmt.Errorf("health: at least one health check path must be configured"))
		}
	}

	// Validate metrics configuration
	if !o.DisableMetrics {
		if o.Metrics.Path == "" {
			errs = append(errs, fmt.Errorf("metrics: metrics path must be configured"))
		}
	}

	// Validate pprof configuration
	if !o.DisablePprof {
		if o.Pprof.Prefix == "" {
			errs = append(errs, fmt.Errorf("pprof: pprof prefix must be configured"))
		}
		if o.Pprof.BlockProfileRate < 0 {
			errs = append(errs, fmt.Errorf("pprof: block profile rate cannot be negative"))
		}
		if o.Pprof.MutexProfileFraction < 0 {
			errs = append(errs, fmt.Errorf("pprof: mutex profile fraction cannot be negative"))
		}
	}

	// Validate request ID configuration
	if !o.DisableRequestID {
		if o.RequestID.Header == "" {
			errs = append(errs, fmt.Errorf("request-id: header name must be configured"))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("middleware validation errors: %v", errs)
	}
	return nil
}

// Complete completes the middleware options with defaults.
func (o *Options) Complete() error {
	if o.Logger.Output == nil {
		o.Logger.Output = log.Printf
	}
	return nil
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

// RecoveryOptions defines recovery middleware options.
type RecoveryOptions struct {
	EnableStackTrace bool                                                       `json:"enable-stack-trace" mapstructure:"enable-stack-trace"`
	OnPanic          func(ctx transport.Context, err interface{}, stack []byte) `json:"-" mapstructure:"-"`
}

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
