package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// RecoveryConfig defines the config for Recovery middleware.
type RecoveryConfig struct {
	// EnableStackTrace includes stack trace in error response (for development).
	EnableStackTrace bool

	// OnPanic is called when a panic occurs.
	// Can be used for logging or alerting.
	OnPanic func(ctx transport.Context, err interface{}, stack []byte)
}

// DefaultRecoveryConfig is the default Recovery middleware config.
var DefaultRecoveryConfig = RecoveryConfig{
	EnableStackTrace: false,
	OnPanic:          nil,
}

// Recovery returns a middleware that recovers from panics.
// It converts panics to JSON error responses using the error code system.
func Recovery() transport.MiddlewareFunc {
	return RecoveryWithConfig(DefaultRecoveryConfig)
}

// RecoveryWithConfig returns a Recovery middleware with custom config.
func RecoveryWithConfig(config RecoveryConfig) transport.MiddlewareFunc {
	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()

					// Call panic handler if configured
					if config.OnPanic != nil {
						config.OnPanic(c, r, stack)
					}

					// Create error response
					var err *errors.Errno
					if config.EnableStackTrace {
						err = errors.ErrPanic.WithMessage(fmt.Sprintf("panic: %v\n%s", r, string(stack)))
					} else {
						err = errors.ErrPanic.WithMessage(fmt.Sprintf("panic: %v", r))
					}

					response.Fail(c, err)
				}
			}()
			next(c)
		}
	}
}
