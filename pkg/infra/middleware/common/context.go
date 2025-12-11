// Package common provides shared utilities for middleware packages.
// This package contains common types and functions used across multiple
// middleware subpackages, avoiding circular dependencies.
package common

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

// Header constants used across middleware.
const (
	// HeaderXRequestID is the header name for request ID.
	HeaderXRequestID = "X-Request-ID"
	// HeaderTraceID is the header name for trace ID.
	HeaderTraceID = "X-Trace-ID"
)

// RequestIDKey is the context key type for request ID.
// Exported for use by other packages.
type RequestIDKey struct{}

// GetRequestID returns the request ID from the context.
// Returns empty string if not found.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// WithRequestID stores the request ID in the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey{}, requestID)
}

// requestIDCounter is the atomic counter for fallback request ID generation.
var requestIDCounter uint64

// GenerateRequestID generates a random request ID using cryptographic random bytes.
// If random generation fails, it falls back to a deterministic ID.
func GenerateRequestID() string {
	b := make([]byte, 16)
	n, err := rand.Read(b)
	if err != nil || n != 16 {
		return generateFallbackRequestID()
	}
	return hex.EncodeToString(b)
}

// generateFallbackRequestID generates a deterministic request ID when random generation fails.
func generateFallbackRequestID() string {
	timestamp := time.Now().Unix()
	counter := atomic.AddUint64(&requestIDCounter, 1)
	return fmt.Sprintf("%x-%x", timestamp, counter)
}
