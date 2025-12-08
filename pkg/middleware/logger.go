package middleware

import (
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// LoggerConfig defines the config for Logger middleware.
type LoggerConfig struct {
	// SkipPaths is a list of paths to skip logging.
	SkipPaths []string

	// Output is the logger output function.
	// Default: log.Printf
	Output func(format string, args ...interface{})
}

// DefaultLoggerConfig is the default Logger middleware config.
var DefaultLoggerConfig = LoggerConfig{
	SkipPaths: []string{"/health", "/ready", "/metrics"},
	Output:    log.Printf,
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
