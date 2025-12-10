package middleware

import (
	"context"
	"time"

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

			// Create timeout context
			ctx, cancel := context.WithTimeout(c.Request(), config.Timeout)
			defer cancel()

			// Update request context
			c.SetRequest(ctx)

			// Create buffered done channel to prevent goroutine leak
			done := make(chan struct{}, 1)

			go func() {
				defer func() {
					// Prevent panic from causing channel to never close
					if r := recover(); r != nil {
						// Log panic if needed, but ensure channel signals completion
					}
					// Non-blocking send to done channel
					select {
					case done <- struct{}{}:
					default:
					}
				}()
				next(c)
			}()

			select {
			case <-done:
				// Request completed normally
			case <-ctx.Done():
				// Timeout occurred
				if ctx.Err() == context.DeadlineExceeded {
					response.Fail(c, errors.ErrRequestTimeout)
				}
			}
		}
	}
}
