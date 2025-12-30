// Package app provides the User Center Service application.
package app

import (
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/internal/user-center/handler"
	"github.com/kart-io/sentinel-x/internal/user-center/router"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	// Register adapters
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
)

const (
	appName        = "sentinel-user-center"
	appDescription = `Sentinel-X User Center Service

The user center service for Sentinel-X platform.

This server provides:
  - User management
  - Authentication & Authorization
  - Profile management`
)

// NewApp creates a new application instance.
func NewApp() *app.App {
	opts := NewOptions()

	return app.NewApp(
		app.WithName(appName),
		app.WithDescription(appDescription),
		app.WithOptions(opts),
		app.WithRunFunc(func() error {
			return Run(opts)
		}),
	)
}

// Run runs the User Center Service with the given options.
func Run(opts *Options) error {
	printBanner(opts)

	// 1. 初始化日志
	opts.Log.AddInitialField("service.name", appName)
	opts.Log.AddInitialField("service.version", app.GetVersion())
	if err := opts.Log.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Info("Starting user-center service...")

	// 2. 初始化 MySQL 数据库
	mysqlClient, err := mysql.New(opts.MySQL)
	if err != nil {
		return fmt.Errorf("failed to initialize mysql: %w", err)
	}
	db := mysqlClient.DB()

	// 数据库迁移
	if err := db.AutoMigrate(&model.User{}, &model.Role{}, &model.UserRole{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	logger.Info("Database migration completed")

	// 3. 初始化 Store 层
	userStore := store.NewUserStore(db)
	roleStore := store.NewRoleStore(db)
	logger.Info("Store layer initialized")

	// 4. 初始化 JWT 认证
	jwtAuth, err := jwt.New(jwt.WithOptions(opts.JWT))
	if err != nil {
		return fmt.Errorf("failed to initialize jwt: %w", err)
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
		server.WithMode(opts.Server.Mode),
		server.WithHTTPOptions(opts.Server.HTTP),
		server.WithGRPCOptions(opts.Server.GRPC),
		server.WithShutdownTimeout(opts.Server.ShutdownTimeout),
	)

	// 8. 注册路由
	if err := router.Register(serverManager, jwtAuth, userHandler, roleHandler, authHandler); err != nil {
		return fmt.Errorf("failed to register routes: %w", err)
	}

	// 9. 启动服务器
	logger.Info("User center service is ready")
	return serverManager.Run()
}

func printBanner(_ *Options) {
	fmt.Printf("Starting %s...\n", appName)
	// Simplified banner for now
}
