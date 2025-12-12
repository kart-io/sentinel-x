package bootstrap

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	server "github.com/kart-io/sentinel-x/pkg/infra/server"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	jwt "github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
)

// AppBootstrapper composes multiple initializers to bootstrap the entire application.
// It follows the Single Responsibility Principle by delegating specific initialization
// tasks to dedicated initializer components.
type AppBootstrapper struct {
	// Ordered list of initializers
	initializers []Initializer

	// Components that need graceful shutdown
	shutdowners []Shutdowner

	// Individual initializers for access
	loggingInit    *LoggingInitializer
	datasourceInit *DatasourceInitializer
	authInit       *AuthInitializer
	middlewareInit *MiddlewareInitializer
	serverInit     *ServerInitializer
}

// BootstrapOptions contains all the configuration needed for bootstrapping.
type BootstrapOptions struct {
	AppName    string
	AppVersion string
	ServerMode string

	LogOpts    *logopts.Options
	ServerOpts *serveropts.Options
	JWTOpts    *jwtopts.Options
	MySQLOpts  *mysqlopts.Options
	RedisOpts  *redisopts.Options

	RegisterFunc func(*server.Manager, *jwt.JWT, *datasource.Manager) error
}

// NewAppBootstrapper creates a new AppBootstrapper with all initializers configured.
func NewAppBootstrapper(opts *BootstrapOptions) *AppBootstrapper {
	b := &AppBootstrapper{}

	// Create individual initializers
	b.loggingInit = NewLoggingInitializer(opts.LogOpts, opts.AppName, opts.AppVersion, opts.ServerMode)
	b.datasourceInit = NewDatasourceInitializer(opts.MySQLOpts, opts.RedisOpts)
	b.authInit = NewAuthInitializer(opts.JWTOpts)
	b.middlewareInit = NewMiddlewareInitializer(
		opts.ServerOpts,
		b.authInit,
		nil, // Will be set after datasource initialization
		opts.AppVersion,
	)
	b.serverInit = NewServerInitializer(opts.ServerOpts, opts.RegisterFunc)

	// Note: middleware init will need datasource manager after datasource init completes
	// This will be handled in the Initialize method

	// Define initialization order (dependencies matter)
	b.initializers = []Initializer{
		b.loggingInit,
		b.datasourceInit,
		b.authInit,
		// middleware init will be added dynamically after datasource init
		// server init will be added last
	}

	// Register components that need graceful shutdown
	b.shutdowners = []Shutdowner{
		b.datasourceInit,
	}

	return b
}

// Initialize runs all initializers in order.
func (b *AppBootstrapper) Initialize(ctx context.Context) error {
	// Run logging initialization first
	if err := b.runInitializer(ctx, b.loggingInit); err != nil {
		return err
	}

	// Run datasource initialization
	if err := b.runInitializer(ctx, b.datasourceInit); err != nil {
		return err
	}

	// Update middleware initializer with datasource manager
	b.middlewareInit.datasourceManager = b.datasourceInit.GetManager()

	// Update auth initializer with datasource manager
	b.authInit.datasourceManager = b.datasourceInit.GetManager()

	// Run auth initialization
	if err := b.runInitializer(ctx, b.authInit); err != nil {
		return err
	}

	// Run middleware initialization
	if err := b.runInitializer(ctx, b.middlewareInit); err != nil {
		return err
	}

	// Run server initialization
	b.serverInit.SetDependencies(b.authInit.GetJWT(), b.datasourceInit.GetManager())
	if err := b.runInitializer(ctx, b.serverInit); err != nil {
		return err
	}

	return nil
}

// runInitializer runs a single initializer with logging.
func (b *AppBootstrapper) runInitializer(ctx context.Context, init Initializer) error {
	logger.Infof("Initializing %s...", init.Name())
	if err := init.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize %s: %w", init.Name(), err)
	}
	return nil
}

// Shutdown gracefully shuts down all components in reverse order.
func (b *AppBootstrapper) Shutdown(ctx context.Context) error {
	var errs []error

	// Shutdown in reverse order of initialization
	for i := len(b.shutdowners) - 1; i >= 0; i-- {
		shutdowner := b.shutdowners[i]
		if err := shutdowner.Shutdown(ctx); err != nil {
			errs = append(errs, err)
			logger.Errorf("Error during shutdown: %v", err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors occurred: %d errors", len(errs))
	}

	return nil
}

// GetServerManager returns the server manager for running the server.
func (b *AppBootstrapper) GetServerManager() *ServerInitializer {
	return b.serverInit
}
