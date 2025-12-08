// Package middleware provides common HTTP middleware for Sentinel-X.
//
// This package includes:
//   - Recovery: Panic recovery with JSON error response
//   - RequestID: Adds unique request ID to each request
//   - Logger: Request logging middleware
//   - CORS: Cross-Origin Resource Sharing support
//   - Timeout: Request timeout handling
//   - Health: Health check endpoints (/health, /live, /ready)
//   - Metrics: Prometheus-compatible metrics endpoint (/metrics)
//   - Pprof: Go profiling endpoints (/debug/pprof/*)
//
// Usage with options:
//
//	server := http.NewServer(
//	    http.WithAddr(":8080"),
//	    http.WithMiddleware(
//	        middleware.WithCORS(),
//	        middleware.WithTimeout(10*time.Second),
//	        middleware.WithHealth(),
//	        middleware.WithMetrics(),
//	        middleware.WithPprof(), // Enable pprof (disabled by default)
//	    ),
//	)
//
// Disable specific middleware:
//
//	server := http.NewServer(
//	    http.WithMiddleware(
//	        middleware.WithoutLogger(),
//	        middleware.WithoutRequestID(),
//	    ),
//	)
//
// Configure middleware options:
//
//	server := http.NewServer(
//	    http.WithMiddleware(
//	        middleware.WithRecovery(
//	            middleware.RecoveryWithStackTrace(),
//	        ),
//	        middleware.WithCORS(
//	            middleware.CORSWithOrigins("https://example.com"),
//	            middleware.CORSWithCredentials(),
//	        ),
//	        middleware.WithHealth(
//	            middleware.HealthWithPath("/healthz"),
//	        ),
//	        middleware.WithPprof(
//	            middleware.PprofWithPrefix("/debug/pprof"),
//	            middleware.PprofWithBlockProfileRate(1),
//	        ),
//	    ),
//	)
package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// Chain creates a middleware chain from multiple middlewares.
// Middlewares are executed in the order they are provided.
func Chain(middlewares ...transport.MiddlewareFunc) transport.MiddlewareFunc {
	return func(next transport.HandlerFunc) transport.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
