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
	"github.com/kart-io/sentinel-x/pkg/component/etcd"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	discoveryetcd "github.com/kart-io/sentinel-x/pkg/infra/discovery/etcd"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	etcdopts "github.com/kart-io/sentinel-x/pkg/options/etcd"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/security/authz/casbin"
)

// Name is the name of the application.
const Name = "sentinel-user-center"

// Config contains application-related configurations.
type Config struct {
	HTTPOptions      *httpopts.Options
	GRPCOptions      *grpcopts.Options
	EtcdOptions      *etcdopts.Options
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
	srv       *server.Manager
	registrar *discoveryetcd.Registrar
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
	if err := db.AutoMigrate(&model.User{}, &model.Role{}, &model.UserRole{}, &model.LoginLog{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	logger.Info("Database migration completed")

	// 3. 初始化 Store 层
	userStore := store.NewUserStore(db)
	roleStore := store.NewRoleStore(db)
	logStore := store.NewLogStore(db)
	logger.Info("Store layer initialized")

	// 4. 初始化 Redis
	redisClient, err := redis.New(cfg.RedisOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis: %w", err)
	}

	// 5. 初始化 JWT 认证
	jwtStore := jwt.NewRedisStore(redisClient, "jwt:blacklist:")
	jwtAuth, err := jwt.New(jwt.WithOptions(cfg.JWTOptions), jwt.WithStore(jwtStore))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize jwt: %w", err)
	}
	logger.Info("JWT authentication initialized")

	// 5.5. 初始化 Casbin 权限服务
	modelPath := "configs/casbin/rbac_model.conf"
	permissionService, err := casbin.NewServiceWithGorm(db, modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize casbin permission service: %w", err)
	}
	logger.Info("Casbin permission service initialized")

	// 6. 初始化 Biz 层
	userService := biz.NewUserService(userStore)
	roleService := biz.NewRoleService(roleStore, userStore, permissionService)
	authService := biz.NewAuthService(jwtAuth, userStore, logStore, redisClient)
	logger.Info("Business layer initialized")

	// 7. 初始化 Handler 层
	userHandler := handler.NewUserHandler(userService, roleService, authService)
	roleHandler := handler.NewRoleHandler(roleService)
	authHandler := handler.NewAuthHandler(authService)
	logger.Info("Handler layer initialized")

	// 8. 初始化服务器
	serverManager := server.NewManager(
		server.WithHTTPOptions(cfg.HTTPOptions),
		server.WithGRPCOptions(cfg.GRPCOptions),
		server.WithMiddleware(cfg.GetMiddlewareOptions()),
		server.WithShutdownTimeout(cfg.ShutdownTimeout),
	)

	// 9. 注册路由
	if err := router.Register(serverManager, jwtAuth, userHandler, roleHandler, authHandler); err != nil {
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	// 10. 初始化 Etcd 注册中心
	var registrar *discoveryetcd.Registrar
	if cfg.EtcdOptions != nil && len(cfg.EtcdOptions.Endpoints) > 0 {
		etcdClient, err := etcd.New(cfg.EtcdOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize etcd client: %w", err)
		}

		// 构造注册信息
		// TODO: 获取真实的对外 IP，目前使用配置的地址
		// Traefik 路由规则：PathPrefix("/v1")
		registrar = discoveryetcd.NewRegistrar(etcdClient.Raw(), Name, cfg.HTTPOptions.Addr, "PathPrefix(`/v1`)")
		logger.Info("Etcd registrar initialized")
	}

	logger.Info("User center service is ready")
	return &Server{
		srv:       serverManager,
		registrar: registrar,
	}, nil
}

// Run starts the server and listens for termination signals.
func (s *Server) Run(ctx context.Context) error {
	if s.registrar != nil {
		if err := s.registrar.Register(ctx); err != nil {
			return fmt.Errorf("failed to register service: %w", err)
		}
		defer s.registrar.Close()
	}

	return s.srv.Run()
}

// GetMiddlewareOptions 从各个配置构建中间件选项。
func (cfg *Config) GetMiddlewareOptions() *middlewareopts.Options {
	opts := middlewareopts.NewOptions()

	if cfg.RecoveryOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareRecovery, cfg.RecoveryOptions)
	}
	if cfg.RequestIDOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareRequestID, cfg.RequestIDOptions)
	}
	if cfg.LoggerOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareLogger, cfg.LoggerOptions)
	}
	if cfg.CORSOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareCORS, cfg.CORSOptions)
	}
	if cfg.TimeoutOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareTimeout, cfg.TimeoutOptions)
	}
	if cfg.HealthOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareHealth, cfg.HealthOptions)
	}
	if cfg.MetricsOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareMetrics, cfg.MetricsOptions)
	}
	if cfg.PprofOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewarePprof, cfg.PprofOptions)
	}
	if cfg.VersionOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareVersion, cfg.VersionOptions)
	}

	return opts
}

func printBanner() {
	fmt.Printf("Starting %s...\n", Name)
}
