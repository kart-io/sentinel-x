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

	"github.com/gin-gonic/gin"
	mwauth "github.com/kart-io/sentinel-x/pkg/infra/middleware/auth"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/security"
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

// Middleware name constants.
const (
	MiddlewareRecovery  = options.MiddlewareRecovery
	MiddlewareRequestID = options.MiddlewareRequestID
	MiddlewareLogger    = options.MiddlewareLogger
	MiddlewareCORS      = options.MiddlewareCORS
	MiddlewareTimeout   = options.MiddlewareTimeout
	MiddlewareHealth    = options.MiddlewareHealth
	MiddlewareMetrics   = options.MiddlewareMetrics
	MiddlewarePprof     = options.MiddlewarePprof
	MiddlewareAuth      = options.MiddlewareAuth
	MiddlewareAuthz     = options.MiddlewareAuthz
)

// NewOptions creates default middleware options.
var NewOptions = options.NewOptions

// ============================================================================
// Observability exports (Logger, Metrics, Tracing)
// ============================================================================

// Observability type aliases for backward compatibility.
type (
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

	// EnhancedLogger returns an enhanced middleware that logs HTTP requests with context propagation.
	EnhancedLogger = observability.EnhancedLogger

	// LoggerWithOptions returns a logger middleware using pure config + runtime dependencies.
	// 这是推荐的 API，适用于配置中心场景。
	LoggerWithOptions = observability.LoggerWithOptions
)

// Tracing re-exports observability.Tracing.
func Tracing(opts ...TracingOption) gin.HandlerFunc {
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
func MetricsMiddlewareWithOptions(opts MetricsOptions) gin.HandlerFunc {
	return observability.MetricsWithOptions(opts)
}

// RegisterMetricsRoutesWithOptions registers metrics endpoint.
// This is a wrapper to convert MetricsOptions to observability.MetricsOptions.
func RegisterMetricsRoutesWithOptions(engine *gin.Engine, opts MetricsOptions) {
	observability.RegisterMetricsRoutesWithOptions(engine, opts)
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
	// RateLimiter is an alias for resilience.RateLimiter.
	RateLimiter = resilience.RateLimiter

	// MemoryRateLimiter is an alias for resilience.MemoryRateLimiter.
	MemoryRateLimiter = resilience.MemoryRateLimiter
)

// Recovery functions re-exports.
var (
	// Recovery returns a middleware that recovers from panics.
	Recovery = resilience.Recovery
)

// Timeout re-exports resilience.Timeout.
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return resilience.Timeout(timeout)
}

// TimeoutWithOptions re-exports resilience.TimeoutWithOptions.
// 这是推荐的构造函数，直接使用 pkg/options/middleware.TimeoutOptions。
var TimeoutWithOptions = resilience.TimeoutWithOptions

// RateLimit functions re-exports.
var (
	// RateLimit returns a rate limiting middleware.
	RateLimit = resilience.RateLimit

	// RateLimitWithOptions returns a RateLimit middleware with options.
	// 这是推荐的 API。
	RateLimitWithOptions = resilience.RateLimitWithOptions

	// NewMemoryRateLimiter creates a new memory-based rate limiter.
	NewMemoryRateLimiter = resilience.NewMemoryRateLimiter
)

// ============================================================================
// Security exports (CORS, SecurityHeaders)
// ============================================================================

// CORS functions re-exports.
var (
	// CORS returns a middleware that adds CORS headers.
	CORS = security.CORS

	// CORSWithOptions returns a CORS middleware with CORSOptions.
	// 这是推荐的构造函数，直接使用 pkg/options/middleware.CORSOptions。
	CORSWithOptions = security.CORSWithOptions
)

// SecurityHeaders functions re-exports.
var (
	// SecurityHeaders returns a middleware that adds security headers.
	SecurityHeaders = security.SecurityHeaders

	// SecurityHeadersWithOptions returns a SecurityHeaders middleware with options.
	// 这是推荐的 API。
	SecurityHeadersWithOptions = security.SecurityHeadersWithOptions
)

// ============================================================================
// Auth exports (AuthWithOptions, AuthzWithOptions)
// ============================================================================

// Auth type aliases for backward compatibility.
type (
	// AuthOptions is an alias for options.AuthOptions (configuration struct).
	AuthOptions = options.AuthOptions

	// AuthzOptions is an alias for options.AuthzOptions (configuration struct).
	AuthzOptions = options.AuthzOptions
)

// Auth functions re-exports.
var (
	// AuthWithOptions returns an authentication middleware using pure config + runtime dependencies.
	// 这是推荐的 API，适用于配置中心场景。
	AuthWithOptions = mwauth.AuthWithOptions

	// NewAuthOptions creates default auth options.
	NewAuthOptions = options.NewAuthOptions
)

// Authz functions re-exports.
var (
	// AuthzWithOptions returns an authorization middleware using pure config + runtime dependencies.
	// 这是推荐的 API，适用于配置中心场景。
	AuthzWithOptions = mwauth.AuthzWithOptions

	// NewAuthzOptions creates default authz options.
	NewAuthzOptions = options.NewAuthzOptions
)
