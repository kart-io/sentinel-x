// Package usercenter provides the User Center Service server implementation.
package usercenter

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/internal/user-center/handler"
	"github.com/kart-io/sentinel-x/internal/user-center/router"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
)

// Name is the name of the application.
const Name = "sentinel-user-center"

// Config contains application-related configurations.
type Config struct {
	HTTPOptions      *httpopts.Options
	GRPCOptions      *grpcopts.Options
	LogOptions       *logopts.Options
	JWTOptions       *jwtopts.Options
	MySQLOptions     *mysqlopts.Options
	RedisOptions     *redisopts.Options
	RecoveryOptions  *middlewareopts.RecoveryOptions
	RequestIDOptions *middlewareopts.RequestIDOptions
	LoggerOptions    *middlewareopts.LoggerOptions
	CORSOptions      *middlewareopts.CORSOptions
	TimeoutOptions   *middlewareopts.TimeoutOptions
	HealthOptions    *middlewareopts.HealthOptions
	MetricsOptions   *middlewareopts.MetricsOptions
	PprofOptions     *middlewareopts.PprofOptions
	VersionOptions   *middlewareopts.VersionOptions
	ShutdownTimeout  time.Duration
}

// Server represents the user center server.
type Server struct {
	srv *server.Manager
}

// NewServer initializes and returns a new Server instance.
func (cfg *Config) NewServer(_ context.Context) (*Server, error) {
	printBanner()

	// 1. 初始化日志
	cfg.LogOptions.AddInitialField("service.name", Name)
	cfg.LogOptions.AddInitialField("service.version", app.GetVersion())
	if err := cfg.LogOptions.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Info("Starting user-center service...")

	// 2. 初始化 MySQL 数据库
	mysqlClient, err := mysql.New(cfg.MySQLOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mysql: %w", err)
	}
	db := mysqlClient.DB()

	// 数据库迁移
	if err := db.AutoMigrate(&model.User{}, &model.Role{}, &model.UserRole{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	logger.Info("Database migration completed")

	// 3. 初始化 Store 层
	userStore := store.NewUserStore(db)
	roleStore := store.NewRoleStore(db)
	logger.Info("Store layer initialized")

	// 4. 初始化 JWT 认证
	jwtAuth, err := jwt.New(jwt.WithOptions(cfg.JWTOptions))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize jwt: %w", err)
	}
	logger.Info("JWT authentication initialized")

	// 5. 初始化 Biz 层
	userService := biz.NewUserService(userStore)
	roleService := biz.NewRoleService(roleStore, userStore)
	authService := biz.NewAuthService(jwtAuth, userStore)
	logger.Info("Business layer initialized")

	// 6. 初始化 Handler 层
	userHandler := handler.NewUserHandler(userService, roleService, authService)
	roleHandler := handler.NewRoleHandler(roleService)
	authHandler := handler.NewAuthHandler(authService)
	logger.Info("Handler layer initialized")

	// 7. 初始化服务器
	serverManager := server.NewManager(
		server.WithHTTPOptions(cfg.HTTPOptions),
		server.WithGRPCOptions(cfg.GRPCOptions),
		server.WithMiddleware(cfg.GetMiddlewareOptions()),
		server.WithShutdownTimeout(cfg.ShutdownTimeout),
	)

	// 8. 注册路由
	if err := router.Register(serverManager, jwtAuth, userHandler, roleHandler, authHandler); err != nil {
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	logger.Info("User center service is ready")
	return &Server{srv: serverManager}, nil
}

// Run starts the server and listens for termination signals.
func (s *Server) Run(_ context.Context) error {
	return s.srv.Run()
}

// GetMiddlewareOptions builds middleware options from individual configurations.
func (cfg *Config) GetMiddlewareOptions() *middlewareopts.Options {
	return &middlewareopts.Options{
		Recovery:  cfg.RecoveryOptions,
		RequestID: cfg.RequestIDOptions,
		Logger:    cfg.LoggerOptions,
		CORS:      cfg.CORSOptions,
		Timeout:   cfg.TimeoutOptions,
		Health:    cfg.HealthOptions,
		Metrics:   cfg.MetricsOptions,
		Pprof:     cfg.PprofOptions,
		Version:   cfg.VersionOptions,
	}
}

func printBanner() {
	fmt.Printf("Starting %s...\n", Name)
}
