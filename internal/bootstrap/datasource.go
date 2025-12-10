package bootstrap

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/component/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/component/redis"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
)

// DatasourceInitializer handles datasource initialization.
type DatasourceInitializer struct {
	mysqlOpts *mysqlopts.Options
	redisOpts *redisopts.Options
	manager   *datasource.Manager
}

// NewDatasourceInitializer creates a new DatasourceInitializer.
func NewDatasourceInitializer(mysqlOpts *mysqlopts.Options, redisOpts *redisopts.Options) *DatasourceInitializer {
	return &DatasourceInitializer{
		mysqlOpts: mysqlOpts,
		redisOpts: redisOpts,
	}
}

// Name returns the name of the initializer.
func (di *DatasourceInitializer) Name() string {
	return "datasources"
}

// Initialize initializes all configured datasources.
func (di *DatasourceInitializer) Initialize(ctx context.Context) error {
	di.manager = datasource.NewManager()
	datasource.SetGlobal(di.manager)

	if di.mysqlOpts.Host != "" {
		if err := di.manager.RegisterMySQL("primary", di.mysqlOpts); err != nil {
			return fmt.Errorf("failed to register mysql: %w", err)
		}
	}

	if di.redisOpts.Host != "" {
		if err := di.manager.RegisterRedis("cache", di.redisOpts); err != nil {
			return fmt.Errorf("failed to register redis: %w", err)
		}
	}

	if err := di.manager.InitAll(ctx); err != nil {
		return fmt.Errorf("failed to initialize datasources: %w", err)
	}

	logger.Info("Datasources initialized successfully")
	return nil
}

// Shutdown performs graceful shutdown of all datasources.
func (di *DatasourceInitializer) Shutdown(ctx context.Context) error {
	if di.manager != nil {
		if err := di.manager.CloseAll(); err != nil {
			logger.Errorw("Failed to close datasources during shutdown", "error", err)
			return fmt.Errorf("failed to close datasources: %w", err)
		}
		logger.Info("All datasources closed successfully")
	}
	return nil
}

// GetManager returns the datasource manager.
func (di *DatasourceInitializer) GetManager() *datasource.Manager {
	return di.manager
}
