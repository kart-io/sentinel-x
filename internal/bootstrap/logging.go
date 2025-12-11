package bootstrap

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
)

// LoggingInitializer handles logging system initialization.
type LoggingInitializer struct {
	opts       *logopts.Options
	appName    string
	appVersion string
	serverMode string
}

// NewLoggingInitializer creates a new LoggingInitializer.
func NewLoggingInitializer(opts *logopts.Options, appName, appVersion, serverMode string) *LoggingInitializer {
	return &LoggingInitializer{
		opts:       opts,
		appName:    appName,
		appVersion: appVersion,
		serverMode: serverMode,
	}
}

// Name returns the name of the initializer.
func (li *LoggingInitializer) Name() string {
	return "logging"
}

// Dependencies returns the names of initializers this one depends on.
// Logging has no dependencies - it should be initialized first.
func (li *LoggingInitializer) Dependencies() []string {
	return nil
}

// Initialize initializes the logging system.
func (li *LoggingInitializer) Initialize(ctx context.Context) error {
	// Inject service metadata into logger options
	li.opts.AddInitialField("service.name", li.appName)
	li.opts.AddInitialField("service.version", li.appVersion)

	if err := li.opts.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger.Infow("Starting Sentinel-X API server",
		"app", li.appName,
		"version", li.appVersion,
		"mode", li.serverMode,
	)

	return nil
}
