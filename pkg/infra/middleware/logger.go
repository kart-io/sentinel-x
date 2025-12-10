package middleware

import (
	"log"
	"sync"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// fieldsPool is a sync.Pool for reusing fields slices to reduce heap allocations.
var fieldsPool = sync.Pool{
	New: func() interface{} {
		s := make([]interface{}, 0, 16)
		return &s
	},
}

// acquireFields retrieves a fields slice from the pool.
func acquireFields() *[]interface{} {
	return fieldsPool.Get().(*[]interface{})
}

// releaseFields resets and returns the fields slice to the pool.
func releaseFields(fields *[]interface{}) {
	// Reset slice to zero length but keep capacity
	*fields = (*fields)[:0]
	fieldsPool.Put(fields)
}

// LoggerConfig defines the config for Logger middleware.
type LoggerConfig struct {
	// SkipPaths is a list of paths to skip logging.
	SkipPaths []string

	// Output is the logger output function.
	// Default: log.Printf (used only if UseStructuredLogger is false)
	Output func(format string, args ...interface{})

	// UseStructuredLogger enables structured logging using kart-io/logger.
	// When true, Output is ignored and the global structured logger is used.
	// Default: true
	UseStructuredLogger bool
}

// DefaultLoggerConfig is the default Logger middleware config.
var DefaultLoggerConfig = LoggerConfig{
	SkipPaths:           []string{"/health", "/ready", "/metrics"},
	Output:              log.Printf,
	UseStructuredLogger: true,
}

// Logger returns a middleware that logs HTTP requests.
func Logger() transport.MiddlewareFunc {
	return LoggerWithConfig(DefaultLoggerConfig)
}

// LoggerWithConfig returns a Logger middleware with custom config.
func LoggerWithConfig(config LoggerConfig) transport.MiddlewareFunc {
	// Set defaults
	if config.Output == nil {
		config.Output = log.Printf
	}

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			// Get request info
			req := c.HTTPRequest()
			path := req.URL.Path

			// Skip logging for certain paths
			if skipPaths[path] {
				next(c)
				return
			}

			// Record start time
			start := time.Now()

			// Get request ID if available
			requestID := GetRequestID(c.Request())

			// Process request
			next(c)

			// Calculate latency
			latency := time.Since(start)

			// Log the request
			if config.UseStructuredLogger {
				// Acquire fields slice from pool
				fields := acquireFields()
				defer releaseFields(fields)

				// Populate fields
				*fields = append(*fields,
					"method", req.Method,
					"path", path,
					"remote_addr", req.RemoteAddr,
					"latency", latency.String(),
					"latency_ms", latency.Milliseconds(),
				)
				if requestID != "" {
					*fields = append(*fields, "request_id", requestID)
				}
				logger.Infow("HTTP Request", (*fields)...)
			} else {
				// Use legacy printf-style logging
				if requestID != "" {
					config.Output("[%s] %s %s %s %v",
						requestID,
						req.Method,
						path,
						req.RemoteAddr,
						latency,
					)
				} else {
					config.Output("%s %s %s %v",
						req.Method,
						path,
						req.RemoteAddr,
						latency,
					)
				}
			}
		}
	}
}
