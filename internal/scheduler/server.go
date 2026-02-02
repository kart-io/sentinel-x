package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/scheduler/biz"
	"github.com/kart-io/sentinel-x/internal/scheduler/handler"
	"github.com/kart-io/sentinel-x/internal/scheduler/router"
	"github.com/kart-io/sentinel-x/internal/scheduler/store"
	"github.com/kart-io/sentinel-x/pkg/component/etcd"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	etcdopts "github.com/kart-io/sentinel-x/pkg/options/etcd"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
	"gorm.io/gorm"
)

// Name is the name of the application.
const Name = "sentinel-scheduler"

// Config contains application-related configurations.
type Config struct {
	HTTPOptions     *httpopts.Options
	GRPCOptions     *grpcopts.Options
	MySQLOptions    *mysqlopts.Options
	RedisOptions    *redisopts.Options
	EtcdOptions     *etcdopts.Options
	LogOptions      *logopts.Options
	RecoveryOptions *middlewareopts.RecoveryOptions
	ShutdownTimeout time.Duration
}

// Server represents the Scheduler server.
type Server struct {
	srv       *server.Manager
	scheduler *biz.Scheduler
	db        *gorm.DB
}

// NewServer initializes and returns a new Server instance.
func (cfg *Config) NewServer(_ context.Context) (*Server, error) {
	// 1. Init Logger
	cfg.LogOptions.AddInitialField("service.name", Name)
	cfg.LogOptions.AddInitialField("service.version", app.GetVersion())
	if err := cfg.LogOptions.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Info("Starting Scheduler service...")

	// 2. Init MySQL
	db, err := mysql.New(cfg.MySQLOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mysql: %w", err)
	}
	logger.Info("MySQL initialized")

	// 3. Init Redis
	redisClient, err := redis.New(cfg.RedisOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis: %w", err)
	}
	logger.Info("Redis initialized")

	// 4. Init Etcd
	etcdClient, err := etcd.New(cfg.EtcdOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize etcd: %w", err)
	}
	logger.Info("Etcd initialized")

	// 5. Init Store (Use db.DB() to get *gorm.DB)
	// AutoMigrate
	if err := db.DB().AutoMigrate(&model.Job{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate job model: %w", err)
	}
	jobStore := store.NewJobStore(db.DB())

	// 6. Init Biz
	jobManager := biz.NewJobManager(jobStore)
	schedulerEngine := biz.NewScheduler(jobStore, redisClient, etcdClient)

	// 7. Init Handler
	schedulerHandler := handler.NewSchedulerHandler(jobManager)

	// 8. Init Server Manager
	serverManager := server.NewManager(
		server.WithHTTPOptions(cfg.HTTPOptions),
		server.WithGRPCOptions(cfg.GRPCOptions),
		// server.WithMiddleware(cfg.GetMiddlewareOptions()), // Simplified for MVP
		server.WithShutdownTimeout(cfg.ShutdownTimeout),
	)

	// 7. Register Routes
	if err := router.Register(serverManager, schedulerHandler); err != nil {
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	return &Server{
		srv:       serverManager,
		scheduler: schedulerEngine,
		db:        db.DB(),
	}, nil
}

// Run starts the server and the scheduler engine.
func (s *Server) Run(ctx context.Context) error {
	// Start Scheduler Engine
	s.scheduler.Start(ctx)
	defer s.scheduler.Stop()

	return s.srv.Run()
}
