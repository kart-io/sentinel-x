package bootstrap

import (
	"context"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	serveropts "github.com/kart-io/sentinel-x/pkg/infra/server"
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

// Initialize creates and configures the server manager.
func (si *ServerInitializer) Initialize(ctx context.Context) error {
	si.manager = server.NewManager(
		server.WithMode(si.serverOpts.Mode),
		server.WithHTTPOptions(si.serverOpts.HTTP),
		server.WithGRPCOptions(si.serverOpts.GRPC),
		server.WithShutdownTimeout(si.serverOpts.ShutdownTimeout),
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
