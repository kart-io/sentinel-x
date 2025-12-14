package observability

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	applogger "github.com/kart-io/sentinel-x/pkg/infra/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/requestutil"
	loggeropts "github.com/kart-io/sentinel-x/pkg/options/logger"
)

// responseWriter is a wrapper around http.ResponseWriter that captures the status code and response size.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	body         *bytes.Buffer
	captureBody  bool
}

// newResponseWriter creates a new responseWriter.
func newResponseWriter(w http.ResponseWriter, captureBody bool) *responseWriter {
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code
		captureBody:    captureBody,
	}
	if captureBody {
		rw.body = &bytes.Buffer{}
	}
	return rw
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size and optionally the body.
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)

	if rw.captureBody && rw.body != nil {
		rw.body.Write(b)
	}
	return n, err
}

// Status returns the captured status code.
func (rw *responseWriter) Status() int {
	return rw.statusCode
}

// BytesWritten returns the number of bytes written.
func (rw *responseWriter) BytesWritten() int64 {
	return rw.bytesWritten
}

// Body returns the captured response body.
func (rw *responseWriter) Body() string {
	if rw.body == nil {
		return ""
	}
	return rw.body.String()
}

// EnhancedLogger returns a middleware that logs HTTP requests with enhanced details.
// It supports capturing request/response bodies, conditional logging, and sensitive data redaction.
func EnhancedLogger(opts *loggeropts.EnhancedLoggerConfig) func(http.Handler) http.Handler {
	if opts == nil {
		opts = loggeropts.NewEnhancedLoggerConfig()
	}

	// Pool for buffers to reduce allocation
	bufPool := sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Skip logging for health checks if configured
			if opts.SkipHealthChecks && (r.URL.Path == "/health" || r.URL.Path == "/healthz" || r.URL.Path == "/livez" || r.URL.Path == "/readyz") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip logging for specific paths
			for _, path := range opts.SkipPaths {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Capture request body if configured
			var reqBody []byte
			if opts.LogRequestBody && r.Body != nil {
				// Read body
				buf := bufPool.Get().(*bytes.Buffer)
				buf.Reset()
				_, err := io.Copy(buf, r.Body)
				if err == nil {
					reqBody = buf.Bytes()
					// Restore body for next handlers
					r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
				}
				bufPool.Put(buf)
			}

			// Wrap response writer to capture status and size
			rw := newResponseWriter(w, opts.LogResponseBody)

			// Process request
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Extract trace ID
			traceID := r.Header.Get(requestutil.HeaderTraceID)
			if traceID == "" {
				traceID = r.Header.Get("X-Request-ID")
			}

			// Build fields as key-value pairs
			keysAndValues := []interface{}{
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"ip", requestutil.GetClientIP(r),
				"status", rw.statusCode,
				"size", rw.bytesWritten,
				"duration", duration,
				"user_agent", r.UserAgent(),
			}

			if traceID != "" {
				keysAndValues = append(keysAndValues, "trace_id", traceID)
			}

			// Add request headers if configured
			if len(opts.CaptureHeaders) > 0 {
				headers := make(map[string]string)
				for _, h := range opts.CaptureHeaders {
					if val := r.Header.Get(h); val != "" {
						headers[h] = val
					}
				}
				if len(headers) > 0 {
					keysAndValues = append(keysAndValues, "headers", headers)
				}
			}

			// Add request body if captured
			if len(reqBody) > 0 {
				// Redact sensitive data
				redactedBody := redactSensitiveData(string(reqBody), opts.SensitiveHeaders)
				// Truncate if too long
				if len(redactedBody) > opts.MaxBodyLogSize {
					redactedBody = redactedBody[:opts.MaxBodyLogSize] + "...(truncated)"
				}
				keysAndValues = append(keysAndValues, "request_body", redactedBody)
			}

			// Add response body if captured
			if rw.captureBody && rw.body != nil && rw.body.Len() > 0 {
				respBody := rw.body.String()
				// Redact sensitive data
				redactedBody := redactSensitiveData(respBody, opts.SensitiveHeaders)
				// Truncate if too long
				if len(redactedBody) > opts.MaxBodyLogSize {
					redactedBody = redactedBody[:opts.MaxBodyLogSize] + "...(truncated)"
				}
				keysAndValues = append(keysAndValues, "response_body", redactedBody)
			}

			// Log the request
			logger := applogger.GetLogger(context.Background())
			message := "HTTP Request"

			switch {
			case rw.statusCode >= 500:
				logger.Errorw(message, keysAndValues...)
			case rw.statusCode >= 400:
				logger.Warnw(message, keysAndValues...)
			default:
				logger.Infow(message, keysAndValues...)
			}
		})
	}
}

// redactSensitiveData redacts sensitive information from a string.
// This can be expanded to handle JSON fields, form data, etc.
func redactSensitiveData(data string, patterns []string) string {
	// Simple implementation - can be enhanced with regex patterns
	result := data
	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(result), strings.ToLower(pattern)) {
			result = "[REDACTED]"
			break
		}
	}
	return result
}
