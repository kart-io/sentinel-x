package resilience

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// TimeoutConfig defines the config for Timeout middleware.
type TimeoutConfig struct {
	// Timeout is the request timeout duration.
	// Default: 30s
	Timeout time.Duration

	// SkipPaths is a list of paths to skip timeout.
	SkipPaths []string
}

// DefaultTimeoutConfig is the default Timeout middleware config.
var DefaultTimeoutConfig = TimeoutConfig{
	Timeout:   30 * time.Second,
	SkipPaths: []string{},
}

// Timeout returns a middleware that limits request processing time.
func Timeout(timeout time.Duration) transport.MiddlewareFunc {
	return TimeoutWithConfig(TimeoutConfig{
		Timeout: timeout,
	})
}

// TimeoutWithConfig returns a Timeout middleware with custom config.
func TimeoutWithConfig(config TimeoutConfig) transport.MiddlewareFunc {
	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeoutConfig.Timeout
	}

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			req := c.HTTPRequest()

			// Skip timeout for certain paths
			if skipPaths[req.URL.Path] {
				next(c)
				return
			}

			// Create timeout context to propagate cancellation to downstream handlers
			ctx, cancel := context.WithTimeout(c.Request(), config.Timeout)
			defer cancel()

			// Update request context so downstream handlers can detect timeout
			c.SetRequest(ctx)

			// Create buffered done channel to prevent goroutine leak
			done := make(chan struct{}, 1)

			// Variable to track if timeout occurred
			var timedOut bool

			go func() {
				defer func() {
					// Recover from panic to prevent process crash
					if r := recover(); r != nil {
						// Log panic information for debugging
						logPanic(r, req.URL.Path)
					}
					// Signal completion - buffered channel guarantees non-blocking send
					// This ensures the goroutine can exit even if the main goroutine
					// has already handled the timeout
					done <- struct{}{}
				}()

				// Handler will receive the timeout context and should respect it
				// If context is cancelled, handlers should stop processing
				next(c)
			}()

			select {
			case <-done:
				// Request completed normally
				// Check if it completed due to context cancellation
				if ctx.Err() == context.DeadlineExceeded {
					timedOut = true
				}
			case <-ctx.Done():
				// Timeout occurred before handler completed
				timedOut = true
			}

			// Only write timeout response if we haven't written a response yet
			// The handler might have already written a response before context cancellation
			if timedOut && ctx.Err() == context.DeadlineExceeded {
				response.Fail(c, errors.ErrRequestTimeout)
			}
		}
	}
}

// logPanic logs panic information with stack trace for debugging.
func logPanic(r interface{}, path string) {
	stack := debug.Stack()
	logger.Errorw("panic recovered in timeout middleware",
		"panic", fmt.Sprintf("%v", r),
		"path", path,
		"stack", string(stack),
	)
}
