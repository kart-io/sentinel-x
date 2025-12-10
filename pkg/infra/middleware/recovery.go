package middleware

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// RecoveryConfig defines the config for Recovery middleware.
type RecoveryConfig struct {
	// EnableStackTrace includes stack trace in error response (for development).
	// Note: In production environment, this will be automatically disabled for client responses.
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
	// Check if running in production environment
	isProd := isProductionEnvironment()

	// Validate and adjust config for production environment
	shouldReturnStackToClient := validateStackTraceConfig(config.EnableStackTrace, isProd)

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()

					// Always log full stack trace to logs
					logPanicWithStackTrace(c, r, stack)

					// Call panic handler if configured
					if config.OnPanic != nil {
						config.OnPanic(c, r, stack)
					}

					// Build client error response
					err := buildClientErrorResponse(r, stack, shouldReturnStackToClient)

					response.Fail(c, err)
				}
			}()
			next(c)
		}
	}
}

// isProductionEnvironment checks if the application is running in production environment.
// It checks APP_ENV or GO_ENV environment variables.
func isProductionEnvironment() bool {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}

	// Convert to lowercase for case-insensitive comparison
	switch env {
	case "production", "prod", "PRODUCTION", "PROD":
		return true
	default:
		return false
	}
}

// validateStackTraceConfig validates and adjusts EnableStackTrace config based on environment.
// In production, it enforces disabling stack trace in client responses with a warning.
func validateStackTraceConfig(enableStackTrace bool, isProd bool) bool {
	if isProd && enableStackTrace {
		logger.Warn("Stack trace is enabled but running in production environment. " +
			"Stack trace will NOT be returned to clients for security reasons. " +
			"Full stack trace will still be logged.")
		return false
	}
	return enableStackTrace
}

// logPanicWithStackTrace logs the panic with full stack trace information.
// This ensures complete debugging information is always available in logs.
func logPanicWithStackTrace(c transport.Context, panicValue interface{}, stack []byte) {
	req := c.HTTPRequest()
	logger.Errorw("panic recovered",
		"panic", panicValue,
		"stack_trace", string(stack),
		"path", req.URL.Path,
		"method", req.Method,
	)
}

// buildClientErrorResponse builds the error response to be returned to the client.
// Stack trace is only included in non-production environments when explicitly enabled.
func buildClientErrorResponse(panicValue interface{}, stack []byte, includeStackTrace bool) *errors.Errno {
	if includeStackTrace {
		// Development environment: include full stack trace for debugging
		return errors.ErrPanic.WithMessage(fmt.Sprintf("panic: %v\n%s", panicValue, string(stack)))
	}

	// Production environment: return generic error message only
	return errors.ErrPanic.WithMessage(fmt.Sprintf("panic: %v", panicValue))
}
