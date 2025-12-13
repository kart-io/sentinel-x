package gin

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// LoggerConfig defines configuration for the logging middleware
type LoggerConfig struct {
	*Config
	// Formatter defines the log formatter function
	Formatter LogFormatter
	// Output defines where to write logs (defaults to logger)
	Output io.Writer
}

// DefaultLoggerConfig returns the default logger config
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Config:    &Config{},
		Formatter: nil, // Will use default formatter
		Output:    nil, // Will use logger
	}
}

// LoggerWithConfig returns a Gin logging middleware with configuration
func LoggerWithConfig(adapter *GinAdapter, config LoggerConfig) HandlerFunc {
	// Set default formatter if not provided
	if config.Formatter == nil {
		config.Formatter = adapter.defaultLogFormatter
	}

	// Update adapter config if provided
	if config.Config != nil {
		adapter.SetConfig(*config.Config)
	}

	return adapter.LoggerWithFormatter(config.Formatter)
}

// CustomRecoveryWithWriter returns a recovery middleware with custom writer and handler
func CustomRecoveryWithWriter(adapter *GinAdapter, out io.Writer, recovery RecoveryFunc) HandlerFunc {
	if recovery == nil {
		recovery = defaultRecoveryHandler(adapter)
	}

	return func(c Context) {
		defer func() {
			if err := recover(); err != nil {
				recovery(c, err)
			}
		}()
		c.Next()
	}
}

// RecoveryFunc defines the recovery handler function signature
type RecoveryFunc func(c Context, err interface{})

// defaultRecoveryHandler returns the default recovery handler
func defaultRecoveryHandler(adapter *GinAdapter) RecoveryFunc {
	return func(c Context, err interface{}) {
		// Log panic details
		fields := []interface{}{
			"component", "gin",
			"operation", "panic_recovery",
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
			"client_ip", c.ClientIP(),
			"user_agent", c.Request().UserAgent(),
			"panic", fmt.Sprintf("%v", err),
		}

		// Add stack trace if available
		if errMsg, ok := err.(error); ok {
			fields = append(fields, "error", errMsg.Error())
		} else {
			fields = append(fields, "error", fmt.Sprintf("%v", err))
		}

		adapter.GetLogger().Errorw("Panic recovered in Gin handler", fields...)

		// Abort with status 500
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

// RequestBodyLogger is a middleware that logs request body
func (g *GinAdapter) RequestBodyLogger(maxSize int64) HandlerFunc {
	return func(c Context) {
		if maxSize <= 0 {
			maxSize = g.config.MaxBodySize
		}

		// Skip if request body logging is disabled
		if !g.config.LogRequestBody {
			c.Next()
			return
		}

		// Skip for certain content types
		contentType := c.ContentType()
		if strings.Contains(contentType, "multipart/") ||
			strings.Contains(contentType, "application/octet-stream") {
			c.Next()
			return
		}

		// Read and restore request body
		if c.Request().Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request().Body, maxSize))
			if err == nil {
				// Restore the body for next handlers
				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Log request body if not empty
				if len(bodyBytes) > 0 {
					fields := []interface{}{
						"component", "gin",
						"operation", "request_body",
						"method", c.Request().Method,
						"path", c.Request().URL.Path,
						"content_type", contentType,
						"body_size", len(bodyBytes),
						"body", string(bodyBytes),
					}

					g.GetLogger().Debugw("Request body logged", fields...)
				}
			}
		}

		c.Next()
	}
}

// ResponseBodyLogger is a middleware that logs response body
func (g *GinAdapter) ResponseBodyLogger() HandlerFunc {
	return func(c Context) {
		// Skip if response body logging is disabled
		if !g.config.LogResponseBody {
			c.Next()
			return
		}

		// Create a response recorder to capture the response
		writer := &responseRecorder{
			ResponseWriter: c.Writer(),
			body:           &bytes.Buffer{},
		}

		// Replace the writer temporarily
		// Note: This would require access to Gin's context writer
		// In practice, you'd need to implement this based on actual Gin types

		c.Next()

		// Log response body after processing
		if writer.body.Len() > 0 && writer.body.Len() <= int(g.config.MaxBodySize) {
			fields := []interface{}{
				"component", "gin",
				"operation", "response_body",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status_code", writer.Status(),
				"body_size", writer.body.Len(),
				"body", writer.body.String(),
			}

			g.GetLogger().Debugw("Response body logged", fields...)
		}
	}
}

// responseRecorder wraps ResponseWriter to capture response body
type responseRecorder struct {
	ResponseWriter
	body *bytes.Buffer
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	// Write to both the original writer and our buffer
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}

// RequestIDMiddleware adds a request ID to the context and logs
func (g *GinAdapter) RequestIDMiddleware(headerName string) HandlerFunc {
	if headerName == "" {
		headerName = "X-Request-ID"
	}

	return func(c Context) {
		requestID := c.GetHeader(headerName)
		if requestID == "" {
			// Generate a new request ID (simplified version)
			requestID = generateRequestID()
		}

		// Set request ID in context
		c.Set("request_id", requestID)
		c.Header(headerName, requestID)

		// Log request ID
		g.GetLogger().Debugw("Request ID set",
			"component", "gin",
			"operation", "request_id",
			"request_id", requestID,
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
		)

		c.Next()
	}
}

// UserContextMiddleware extracts user information and adds to context
func (g *GinAdapter) UserContextMiddleware(userIDHeader string) HandlerFunc {
	if userIDHeader == "" {
		userIDHeader = "X-User-ID"
	}

	return func(c Context) {
		userID := c.GetHeader(userIDHeader)
		if userID != "" {
			c.Set("user_id", userID)

			g.GetLogger().Debugw("User context set",
				"component", "gin",
				"operation", "user_context",
				"user_id", userID,
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
			)
		}

		c.Next()
	}
}

// HealthCheckSkipper is a middleware that skips logging for health check endpoints
func (g *GinAdapter) HealthCheckSkipper(healthPaths ...string) HandlerFunc {
	if len(healthPaths) == 0 {
		healthPaths = []string{"/health", "/healthz", "/ping", "/status"}
	}

	return func(c Context) {
		path := c.Request().URL.Path

		// Skip logging for health check paths
		for _, healthPath := range healthPaths {
			if path == healthPath {
				// Set a flag to skip logging
				c.Set("skip_logging", true)
				break
			}
		}

		c.Next()
	}
}

// MetricsMiddleware logs request metrics
func (g *GinAdapter) MetricsMiddleware() HandlerFunc {
	return func(c Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)

		fields := []interface{}{
			"component", "gin",
			"operation", "request_metrics",
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
			"status_code", c.Writer().Status(),
			"duration_ms", float64(duration.Nanoseconds()) / 1e6,
			"body_size", c.Writer().Size(),
			"client_ip", c.ClientIP(),
		}

		// Add user context if available
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, "user_id", userID)
		}

		// Add request ID if available
		if requestID, exists := c.Get("request_id"); exists {
			fields = append(fields, "request_id", requestID)
		}

		g.GetLogger().Infow("Request metrics", fields...)
	}
}

// ErrorHandlerMiddleware handles and logs errors
func (g *GinAdapter) ErrorHandlerMiddleware() HandlerFunc {
	return func(c Context) {
		c.Next()

		// Process any errors that occurred
		if len(c.Request().Header.Get("errors")) > 0 {
			// This would need to be implemented based on how Gin handles errors
			// For now, we'll just log a generic error handling message
			g.GetLogger().Debugw("Error handler executed",
				"component", "gin",
				"operation", "error_handler",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status_code", c.Writer().Status(),
			)
		}
	}
}

// PathFilterMiddleware filters logs based on path patterns
func (g *GinAdapter) PathFilterMiddleware(skipPatterns []string) HandlerFunc {
	var compiledPatterns []*regexp.Regexp

	for _, pattern := range skipPatterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			compiledPatterns = append(compiledPatterns, compiled)
		}
	}

	return func(c Context) {
		path := c.Request().URL.Path

		// Check if path matches any skip pattern
		for _, pattern := range compiledPatterns {
			if pattern.MatchString(path) {
				c.Set("skip_logging", true)
				break
			}
		}

		c.Next()
	}
}

// Helper functions

// generateRequestID generates a simple request ID
func generateRequestID() string {
	// This is a simplified version - in practice you might want to use
	// a more robust UUID generation library
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// Convenience functions for creating middleware combinations

// DefaultMiddleware returns a set of default middleware
func (g *GinAdapter) DefaultMiddleware() []HandlerFunc {
	return []HandlerFunc{
		g.RequestIDMiddleware(""),
		g.Logger(),
		g.Recovery(),
	}
}

// DebugMiddleware returns middleware suitable for debug environments
func (g *GinAdapter) DebugMiddleware() []HandlerFunc {
	return []HandlerFunc{
		g.RequestIDMiddleware(""),
		g.RequestBodyLogger(0),
		g.Logger(),
		g.MetricsMiddleware(),
		g.Recovery(),
	}
}

// ProductionMiddleware returns middleware suitable for production environments
func (g *GinAdapter) ProductionMiddleware(healthPaths ...string) []HandlerFunc {
	return []HandlerFunc{
		g.RequestIDMiddleware(""),
		g.HealthCheckSkipper(healthPaths...),
		g.UserContextMiddleware(""),
		g.Logger(),
		g.Recovery(),
	}
}
