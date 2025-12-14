package bootstrap

import (
	"context"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	server "github.com/kart-io/sentinel-x/pkg/infra/server"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
)

// ServerInitializer handles server manager initialization.
type ServerInitializer struct {
	serverOpts   *serveropts.Options
	registerFunc func(*server.Manager, *jwt.JWT, *datasource.Manager) error
	manager      *server.Manager

	// Dependencies
	jwtAuth   *jwt.JWT
	dsManager *datasource.Manager
}

// NewServerInitializer creates a new ServerInitializer.
func NewServerInitializer(serverOpts *serveropts.Options, registerFunc func(*server.Manager, *jwt.JWT, *datasource.Manager) error) *ServerInitializer {
	return &ServerInitializer{
		serverOpts:   serverOpts,
		registerFunc: registerFunc,
	}
}

// SetDependencies sets the dependencies needed for registration.
func (si *ServerInitializer) SetDependencies(jwtAuth *jwt.JWT, dsManager *datasource.Manager) {
	si.jwtAuth = jwtAuth
	si.dsManager = dsManager
}

// Name returns the name of the initializer.
func (si *ServerInitializer) Name() string {
	return "server"
}

// Dependencies returns the names of initializers this one depends on.
// Server depends on middleware for proper middleware configuration.
func (si *ServerInitializer) Dependencies() []string {
	return []string{"middleware"}
}

// Initialize creates and configures the server manager.
func (si *ServerInitializer) Initialize(_ context.Context) error {
	si.manager = server.NewManager(
		serveropts.WithMode(si.serverOpts.Mode),
		serveropts.WithHTTPOptions(si.serverOpts.HTTP),
		serveropts.WithGRPCOptions(si.serverOpts.GRPC),
		serveropts.WithShutdownTimeout(si.serverOpts.ShutdownTimeout),
	)

	if si.registerFunc != nil {
		if err := si.registerFunc(si.manager, si.jwtAuth, si.dsManager); err != nil {
			return err
		}
	}

	logger.Info("All services registered successfully")
	return nil
}

// GetManager returns the server manager.
func (si *ServerInitializer) GetManager() *server.Manager {
	return si.manager
}

// Run starts the server manager.
func (si *ServerInitializer) Run() error {
	if si.manager == nil {
		logger.Error("Server manager not initialized")
		return nil
	}

	logger.Info("Starting server manager...")
	return si.manager.Run()
}
