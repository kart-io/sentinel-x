package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// RequestID header names.
const (
	HeaderXRequestID = "X-Request-ID"
)

// RequestIDConfig defines the config for RequestID middleware.
type RequestIDConfig struct {
	// Header is the header name to use for request ID.
	// Default: "X-Request-ID"
	Header string

	// Generator is the function to generate request IDs.
	// Default: generates a random 16-byte hex string
	Generator func() string

	// ContextKey is the key to store request ID in context.
	// Default: "request_id"
	ContextKey string
}

// DefaultRequestIDConfig is the default RequestID middleware config.
var DefaultRequestIDConfig = RequestIDConfig{
	Header:     HeaderXRequestID,
	Generator:  generateRequestID,
	ContextKey: "request_id",
}

// requestIDKey is the context key for request ID.
type requestIDKey struct{}

// RequestID returns a middleware that adds a unique request ID to each request.
// The request ID is added to:
//   - Response header (X-Request-ID)
//   - Request context (can be retrieved with GetRequestID)
func RequestID() transport.MiddlewareFunc {
	return RequestIDWithConfig(DefaultRequestIDConfig)
}

// RequestIDWithConfig returns a RequestID middleware with custom config.
func RequestIDWithConfig(config RequestIDConfig) transport.MiddlewareFunc {
	// Set defaults
	if config.Header == "" {
		config.Header = HeaderXRequestID
	}
	if config.Generator == nil {
		config.Generator = generateRequestID
	}
	if config.ContextKey == "" {
		config.ContextKey = "request_id"
	}

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			// Check if request ID already exists in header
			requestID := c.Header(config.Header)
			if requestID == "" {
				requestID = config.Generator()
			}

			// Set request ID in response header
			c.SetHeader(config.Header, requestID)

			// Store request ID in context
			ctx := context.WithValue(c.Request(), requestIDKey{}, requestID)
			c.SetRequest(ctx)

			next(c)
		}
	}
}

// GetRequestID returns the request ID from the context.
// Returns empty string if not found.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// generateRequestID generates a random request ID.
func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
