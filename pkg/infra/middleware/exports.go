// Package middleware provides HTTP middleware components.
//
// This file re-exports types from subpackages for backward compatibility.
// New code should import the appropriate subpackage directly:
//
//	import "github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
//	import "github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
//	import "github.com/kart-io/sentinel-x/pkg/infra/middleware/security"
package middleware

import (
	"time"

	mwauth "github.com/kart-io/sentinel-x/pkg/infra/middleware/auth"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/security"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	loggeropts "github.com/kart-io/sentinel-x/pkg/options/logger"
	options "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// ============================================================================
// Options exports (Configuration types)
// ============================================================================

// Options type aliases for backward compatibility.
type (
	// Options is the main middleware options container.
	Options = options.Options

	// RecoveryOptions defines recovery middleware options.
	RecoveryOptions = options.RecoveryOptions

	// RequestIDOptions defines request ID middleware options.
	RequestIDOptions = options.RequestIDOptions

	// LoggerOptions defines logger middleware options.
	LoggerOptions = options.LoggerOptions

	// CORSOptions defines CORS middleware options.
	CORSOptions = options.CORSOptions

	// TimeoutOptions defines timeout middleware options.
	TimeoutOptions = options.TimeoutOptions

	// HealthOptions defines health check options.
	HealthOptions = options.HealthOptions

	// MetricsOptions defines metrics options.
	MetricsOptions = options.MetricsOptions

	// PprofOptions defines pprof options.
	PprofOptions = options.PprofOptions
)

// NewOptions creates default middleware options.
var NewOptions = options.NewOptions

// ============================================================================
// Observability exports (Logger, Metrics, Tracing)
// ============================================================================

// Observability type aliases for backward compatibility.
type (
	// LoggerConfig is an alias for observability.LoggerConfig.
	LoggerConfig = observability.LoggerConfig

	// EnhancedLoggerConfig is an alias for loggeropts.EnhancedLoggerConfig.
	EnhancedLoggerConfig = loggeropts.EnhancedLoggerConfig

	// TracingOptions is an alias for observability.TracingOptions.
	TracingOptions = observability.TracingOptions

	// TracingOption is an alias for observability.TracingOption.
	TracingOption = observability.TracingOption

	// MetricsCollector is an alias for observability.MetricsCollector.
	MetricsCollector = observability.MetricsCollector
)

// Logger functions re-exports.
var (
	// Logger returns a middleware that logs HTTP requests.
	Logger = observability.Logger

	// LoggerWithConfig returns a Logger middleware with custom config.
	LoggerWithConfig = observability.LoggerWithConfig

	// EnhancedLogger returns an enhanced middleware that logs HTTP requests with context propagation.
	EnhancedLogger = observability.EnhancedLogger
)

// Tracing re-exports observability.Tracing.
func Tracing(opts ...TracingOption) transport.MiddlewareFunc {
	return observability.Tracing(opts...)
}

// NewTracingOptions re-exports observability.NewTracingOptions.
var NewTracingOptions = observability.NewTracingOptions

// Tracing option functions.
var (
	WithTracerName              = observability.WithTracerName
	WithSpanNameFormatter       = observability.WithSpanNameFormatter
	WithRequestBodyCapture      = observability.WithRequestBodyCapture
	WithResponseBodyCapture     = observability.WithResponseBodyCapture
	WithTracingSkipPaths        = observability.WithTracingSkipPaths
	WithTracingSkipPathPrefixes = observability.WithTracingSkipPathPrefixes
	WithAttributeExtractor      = observability.WithAttributeExtractor
)

// MetricsMiddlewareWithOptions creates a middleware that collects metrics.
// This is a wrapper to convert MetricsOptions to observability.MetricsOptions.
func MetricsMiddlewareWithOptions(opts MetricsOptions) transport.MiddlewareFunc {
	return observability.MetricsMiddleware(opts)
}

// RegisterMetricsRoutesWithOptions registers metrics endpoint.
// This is a wrapper to convert MetricsOptions to observability.MetricsOptions.
func RegisterMetricsRoutesWithOptions(router transport.Router, opts MetricsOptions) {
	observability.RegisterMetricsRoutes(router, opts)
}

// Metrics functions re-exports.
var (
	GetMetricsCollector   = observability.GetMetricsCollector
	ResetMetricsCollector = observability.ResetMetricsCollector
	ResetMetrics          = observability.ResetMetrics
	NewMetricsCollector   = observability.NewMetricsCollector
)

// Tracing helper functions.
var (
	ExtractTraceID = observability.ExtractTraceID
	ExtractSpanID  = observability.ExtractSpanID
)

// TracerNameFromObservability re-exports the tracer name constant.
const TracerNameFromObservability = observability.TracerName

// ============================================================================
// Resilience exports (Recovery, Timeout, RateLimit)
// ============================================================================

// Resilience type aliases for backward compatibility.
type (
	// RecoveryConfig is an alias for resilience.RecoveryConfig.
	RecoveryConfig = resilience.RecoveryConfig

	// TimeoutConfig is an alias for resilience.TimeoutConfig.
	TimeoutConfig = resilience.TimeoutConfig

	// RateLimitConfig is an alias for resilience.RateLimitConfig.
	RateLimitConfig = resilience.RateLimitConfig

	// RateLimiter is an alias for resilience.RateLimiter.
	RateLimiter = resilience.RateLimiter

	// MemoryRateLimiter is an alias for resilience.MemoryRateLimiter.
	MemoryRateLimiter = resilience.MemoryRateLimiter
)

// Recovery functions re-exports.
var (
	// Recovery returns a middleware that recovers from panics.
	Recovery = resilience.Recovery

	// RecoveryWithConfig returns a Recovery middleware with custom config.
	RecoveryWithConfig = resilience.RecoveryWithConfig

	// DefaultRecoveryConfig is the default Recovery middleware config.
	DefaultRecoveryConfig = resilience.DefaultRecoveryConfig
)

// Timeout re-exports resilience.Timeout.
func Timeout(timeout time.Duration) transport.MiddlewareFunc {
	return resilience.Timeout(timeout)
}

// TimeoutWithConfig re-exports resilience.TimeoutWithConfig.
var TimeoutWithConfig = resilience.TimeoutWithConfig

// DefaultTimeoutConfig is the default Timeout middleware config.
var DefaultTimeoutConfig = resilience.DefaultTimeoutConfig

// RateLimit functions re-exports.
var (
	// RateLimit returns a rate limiting middleware.
	RateLimit = resilience.RateLimit

	// RateLimitWithConfig returns a RateLimit middleware with custom config.
	RateLimitWithConfig = resilience.RateLimitWithConfig

	// NewMemoryRateLimiter creates a new memory-based rate limiter.
	NewMemoryRateLimiter = resilience.NewMemoryRateLimiter
)

// ============================================================================
// Security exports (CORS, SecurityHeaders)
// ============================================================================

// Security type aliases for backward compatibility.
type (
	// CORSConfig is an alias for security.CORSConfig.
	CORSConfig = security.CORSConfig

	// SecurityHeadersConfig is an alias for security.HeadersConfig.
	SecurityHeadersConfig = security.HeadersConfig
)

// CORS functions re-exports.
var (
	// CORS returns a middleware that adds CORS headers.
	CORS = security.CORS

	// CORSWithConfig returns a CORS middleware with custom config.
	CORSWithConfig = security.CORSWithConfig

	// DefaultCORSConfig is the default CORS middleware config.
	DefaultCORSConfig = security.DefaultCORSConfig
)

// SecurityHeaders functions re-exports.
var (
	// SecurityHeaders returns a middleware that adds security headers.
	SecurityHeaders = security.Headers

	// SecurityHeadersWithConfig returns a SecurityHeaders middleware with custom config.
	SecurityHeadersWithConfig = security.HeadersWithConfig

	// DefaultSecurityHeadersConfig is the default SecurityHeaders middleware config.
	DefaultSecurityHeadersConfig = security.DefaultHeadersConfig
)

// ============================================================================
// Auth exports (Auth, Authz)
// ============================================================================

// Auth type aliases for backward compatibility.
// AuthOptions 和 AuthzOptions 来自 options 子包（配置结构体）
// AuthOption 和 AuthzOption 来自 auth 中间件子包（中间件配置）
type (
	// AuthOptions is an alias for options.AuthOptions (configuration struct).
	AuthOptions = options.AuthOptions

	// AuthOption is an alias for mwauth.Option (middleware option).
	AuthOption = mwauth.Option

	// AuthzOptions is an alias for options.AuthzOptions (configuration struct).
	AuthzOptions = options.AuthzOptions

	// AuthzOption is an alias for mwauth.AuthzOption (middleware option).
	AuthzOption = mwauth.AuthzOption

	// ActionMapping is an alias for mwauth.ActionMapping.
	ActionMapping = mwauth.ActionMapping
)

// Auth functions re-exports.
var (
	// Auth returns an authentication middleware.
	Auth = mwauth.Auth

	// NewAuthOptions creates default auth options.
	NewAuthOptions = mwauth.NewOptions

	// AuthWithAuthenticator sets the authenticator.
	AuthWithAuthenticator = mwauth.WithAuthenticator

	// AuthWithTokenLookup sets how to extract the token.
	AuthWithTokenLookup = mwauth.WithTokenLookup

	// AuthWithAuthScheme sets the authorization scheme.
	AuthWithAuthScheme = mwauth.WithAuthScheme

	// AuthWithSkipPaths sets paths to skip authentication.
	AuthWithSkipPaths = mwauth.WithSkipPaths

	// AuthWithSkipPathPrefixes sets path prefixes to skip authentication.
	AuthWithSkipPathPrefixes = mwauth.WithSkipPathPrefixes

	// AuthWithErrorHandler sets the error handler.
	AuthWithErrorHandler = mwauth.WithErrorHandler

	// AuthWithSuccessHandler sets the success handler.
	AuthWithSuccessHandler = mwauth.WithSuccessHandler
)

// Authz functions re-exports.
var (
	// Authz returns an authorization middleware.
	Authz = mwauth.Authz

	// NewAuthzOptions creates default authz options.
	NewAuthzOptions = mwauth.NewAuthzOptions

	// AuthzWithAuthorizer sets the authorizer.
	AuthzWithAuthorizer = mwauth.AuthzWithAuthorizer

	// AuthzWithSkipPaths sets paths to skip authorization.
	AuthzWithSkipPaths = mwauth.AuthzWithSkipPaths

	// AuthzWithSkipPathPrefixes sets path prefixes to skip authorization.
	AuthzWithSkipPathPrefixes = mwauth.AuthzWithSkipPathPrefixes

	// DefaultActionMapping is the default HTTP method to action mapping.
	DefaultActionMapping = mwauth.DefaultActionMapping
)
