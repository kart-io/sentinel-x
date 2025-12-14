package factory

import (
	"context"
	"fmt"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/engines/slog"
	"github.com/kart-io/logger/engines/zap"
	"github.com/kart-io/logger/errors"
	"github.com/kart-io/logger/option"
)

// LoggerFactory creates logger instances based on configuration.
type LoggerFactory struct {
	option       *option.LogOption
	errorHandler *errors.ErrorHandler
}

// NewLoggerFactory creates a new logger factory with the provided configuration.
func NewLoggerFactory(opt *option.LogOption) *LoggerFactory {
	return &LoggerFactory{
		option:       opt,
		errorHandler: errors.NewErrorHandler(nil), // Use default retry policy
	}
}

// NewLoggerFactoryWithErrorHandler creates a new logger factory with custom error handling.
func NewLoggerFactoryWithErrorHandler(opt *option.LogOption, errorHandler *errors.ErrorHandler) *LoggerFactory {
	return &LoggerFactory{
		option:       opt,
		errorHandler: errorHandler,
	}
}

// CreateLogger creates a logger instance based on the configured engine.
func (f *LoggerFactory) CreateLogger() (core.Logger, error) {
	return f.CreateLoggerWithContext(context.Background())
}

// CreateLoggerWithContext creates a logger instance with context for error handling.
func (f *LoggerFactory) CreateLoggerWithContext(ctx context.Context) (core.Logger, error) {
	// Validate configuration
	if err := f.option.Validate(); err != nil {
		configErr := errors.NewError(errors.ConfigError, "factory", "invalid configuration", err)
		f.errorHandler.HandleError(configErr)
		return nil, configErr
	}

	// Validate engine before attempting creation
	if f.option.Engine != "zap" && f.option.Engine != "slog" {
		engineErr := errors.NewError(errors.ConfigError, "factory", fmt.Sprintf("unsupported logger engine: %s", f.option.Engine), nil)
		f.errorHandler.HandleError(engineErr)
		return nil, engineErr
	}

	// Try to create logger with retry logic and graceful degradation
	var logger core.Logger
	var createErr error

	err := f.errorHandler.ExecuteWithRetry(ctx, "factory", func() error {
		switch f.option.Engine {
		case "zap":
			if l, err := f.createZapLogger(); err == nil {
				logger = l
				return nil
			} else {
				// Try fallback to slog
				if fallbackLogger, fallbackErr := f.createSlogLogger(); fallbackErr == nil {
					logger = fallbackLogger
					return nil
				}
				return errors.NewError(errors.EngineError, "factory", "both zap and slog engines failed", err)
			}
		case "slog":
			if l, err := f.createSlogLogger(); err == nil {
				logger = l
				return nil
			} else {
				// Try fallback to zap
				if fallbackLogger, fallbackErr := f.createZapLogger(); fallbackErr == nil {
					logger = fallbackLogger
					return nil
				}
				return errors.NewError(errors.EngineError, "factory", "both slog and zap engines failed", err)
			}
		default:
			return errors.NewError(errors.ConfigError, "factory", fmt.Sprintf("unsupported logger engine: %s", f.option.Engine), nil)
		}
	})
	if err != nil {
		createErr = err
		// As a last resort, return the fallback logger from error handler
		logger = f.errorHandler.GetFallbackLogger()
	}

	return logger, createErr
}

// createZapLogger creates a Zap-based logger implementation.
func (f *LoggerFactory) createZapLogger() (core.Logger, error) {
	return zap.NewZapLogger(f.option)
}

// createSlogLogger creates a Slog-based logger implementation.
func (f *LoggerFactory) createSlogLogger() (core.Logger, error) {
	return slog.NewSlogLogger(f.option)
}

// GetOption returns the current configuration.
func (f *LoggerFactory) GetOption() *option.LogOption {
	return f.option
}

// UpdateOption updates the factory configuration and can be used for dynamic reconfiguration.
func (f *LoggerFactory) UpdateOption(opt *option.LogOption) error {
	if err := opt.Validate(); err != nil {
		configErr := errors.NewError(errors.ConfigError, "factory", "invalid configuration update", err)
		f.errorHandler.HandleError(configErr)
		return configErr
	}
	f.option = opt
	return nil
}

// GetErrorHandler returns the error handler for inspection or configuration.
func (f *LoggerFactory) GetErrorHandler() *errors.ErrorHandler {
	return f.errorHandler
}

// GetErrorStats returns error statistics from the factory's error handler.
func (f *LoggerFactory) GetErrorStats() map[string]int {
	return f.errorHandler.GetErrorStats()
}

// GetLastErrors returns the most recent errors from the factory's error handler.
func (f *LoggerFactory) GetLastErrors() map[string]*errors.LoggerError {
	return f.errorHandler.GetLastErrors()
}

// ResetErrors clears all error statistics and cached errors.
func (f *LoggerFactory) ResetErrors() {
	f.errorHandler.Reset()
}

// SetErrorCallback sets a callback function to be called when errors occur.
func (f *LoggerFactory) SetErrorCallback(callback func(*errors.LoggerError)) {
	f.errorHandler.SetErrorCallback(callback)
}

// SetFallbackLogger sets the fallback logger to use when all engines fail.
func (f *LoggerFactory) SetFallbackLogger(logger core.Logger) {
	f.errorHandler.SetFallbackLogger(logger)
}
