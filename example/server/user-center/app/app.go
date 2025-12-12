package app

import (
	"context"
	"fmt"
	"os"

	drivermysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/example/server/user-center/handler"
	"github.com/kart-io/sentinel-x/example/server/user-center/service/userservice"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	// Import bridge to register gin adapter
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/security/authz/casbin"
	"github.com/kart-io/sentinel-x/pkg/security/authz/casbin/infrastructure/mysql"
)

const (
	appName = "user-center"
)

func NewApp() *app.App {
	opts := NewOptions()

	return app.NewApp(
		app.WithName(appName),
		app.WithOptions(opts),
		app.WithRunFunc(func() error {
			return Run(opts)
		}),
	)
}

func Run(opts *Options) error {
	if err := opts.Log.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		_ = logger.Flush()
	}()

	// 1. Initialize JWT
	jwtAuth, err := jwt.New(
		jwt.WithOptions(opts.JWT),
	)
	if err != nil {
		return fmt.Errorf("failed to init jwt: %w", err)
	}

	// 2. Initialize Casbin
	var repo casbin.Repository
	if os.Getenv("USE_MYSQL") == "true" {
		dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
		db, err := gorm.Open(drivermysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to connect db: %w", err)
		}
		repo, err = mysql.NewRepository(db)
		if err != nil {
			return fmt.Errorf("failed to init repo: %w", err)
		}
	} else {
		repo = &MockRepository{}
	}

	if err := createModelConf(); err != nil {
		return err
	}
	defer func() {
		_ = os.Remove("model.conf")
	}()

	permSvc, err := casbin.NewPermissionService("model.conf", repo)
	if err != nil {
		return fmt.Errorf("failed to init permission service: %w", err)
	}

	setupPolicies(permSvc)

	// 3. Initialize Service and Handlers
	svc := userservice.NewService(jwtAuth)
	authHandler := handler.NewAuthHandler(svc)
	userHandler := handler.NewUserHandler(svc)

	// 4. Create Server Manager
	mgr := server.NewManager(
		serveropts.WithMode(opts.Server.Mode),
		serveropts.WithHTTPOptions(opts.Server.HTTP),
		serveropts.WithGRPCOptions(opts.Server.GRPC),
	)

	// 5. Register Service
	router := &Router{
		authHandler: authHandler,
		userHandler: userHandler,
		jwtAuth:     jwtAuth,
		permSvc:     permSvc,
	}

	if err := mgr.RegisterHTTP(svc, router); err != nil {
		return err
	}

	// Register gRPC Service
	if err := mgr.RegisterGRPC(svc, &transport.GRPCServiceDesc{
		ServiceDesc: &v1.UserService_ServiceDesc,
		ServiceImpl: userHandler,
	}); err != nil {
		return err
	}

	return mgr.Run()
}

type Router struct {
	authHandler *handler.AuthHandler
	userHandler *handler.UserHandler
	jwtAuth     *jwt.JWT
	permSvc     casbin.PermissionService
}

func (r *Router) RegisterRoutes(router transport.Router) {
	// Public routes (no auth required)
	auth := router.Group("/api/v1/auth")
	auth.Handle("POST", "/login", r.authHandler.Login)
	auth.Handle("POST", "/register", r.authHandler.Register)

	// Protected routes
	api := router.Group("/api/v1")
	api.Use(authMiddleware(r.jwtAuth))
	api.Use(casbinMiddleware(r.permSvc, r.jwtAuth))

	// Auth routes (require authentication)
	api.Handle("POST", "/auth/change-password", r.authHandler.ChangePassword)

	// User routes
	api.Handle("GET", "/users", r.userHandler.ListUsers)
	api.Handle("POST", "/users", r.userHandler.CreateUser)
	api.Handle("GET", "/users/:id", r.userHandler.GetProfile)
	api.Handle("PUT", "/users/:id", r.userHandler.UpdateUser)
	api.Handle("DELETE", "/users/:id", r.userHandler.DeleteUser)
	api.Handle("POST", "/users/batch-delete", r.userHandler.BatchDelete)

	// Admin routes
	api.Handle("POST", "/admin/action", r.userHandler.AdminAction)
}

func authMiddleware(jwtAuth *jwt.JWT) transport.MiddlewareFunc {
	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			token := c.Header("Authorization")
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
				_, err := jwtAuth.Verify(c.Request(), token)
				if err != nil {
					c.JSON(401, map[string]string{"error": "invalid token"})
					return
				}
			} else {
				c.JSON(401, map[string]string{"error": "missing token"})
				return
			}
			next(c)
		}
	}
}

func casbinMiddleware(permSvc casbin.PermissionService, jwtAuth *jwt.JWT) transport.MiddlewareFunc {
	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			path := c.HTTPRequest().URL.Path
			method := c.HTTPRequest().Method

			token := c.Header("Authorization")
			role := "anonymous"
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
				claims, err := jwtAuth.Verify(c.Request(), token)
				if err == nil {
					if claims.Subject == "1" {
						role = "admin"
					} else {
						role = "user"
					}
				}
			}

			allowed, err := permSvc.Enforce(role, path, method)
			if err != nil {
				c.JSON(500, map[string]string{"error": "authorization error"})
				return
			}
			if !allowed {
				c.JSON(403, map[string]string{"error": "forbidden"})
				return
			}

			next(c)
		}
	}
}

func createModelConf() error {
	conf := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act
`
	return os.WriteFile("model.conf", []byte(conf), 0o644)
}

func setupPolicies(svc casbin.PermissionService) {
	// Admin has full access to all user operations
	if _, err := svc.AddPolicy("admin", "/api/v1/users", "GET"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("admin", "/api/v1/users", "POST"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("admin", "/api/v1/users/:id", "GET"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("admin", "/api/v1/users/:id", "PUT"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("admin", "/api/v1/users/:id", "DELETE"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("admin", "/api/v1/users/batch-delete", "POST"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("admin", "/api/v1/admin/action", "POST"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("admin", "/api/v1/auth/change-password", "POST"); err != nil {
		panic(err)
	}

	// Regular users can only read user list and their own profile
	if _, err := svc.AddPolicy("user", "/api/v1/users", "GET"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("user", "/api/v1/users/:id", "GET"); err != nil {
		panic(err)
	}
	if _, err := svc.AddPolicy("user", "/api/v1/auth/change-password", "POST"); err != nil {
		panic(err)
	}
}

type MockRepository struct {
	policies []*casbin.Policy
}

func (m *MockRepository) LoadPolicies(ctx context.Context) ([]*casbin.Policy, error) {
	return m.policies, nil
}

func (m *MockRepository) SavePolicies(ctx context.Context, policies []*casbin.Policy) error {
	m.policies = policies
	return nil
}

func (m *MockRepository) AddPolicy(ctx context.Context, p *casbin.Policy) error {
	m.policies = append(m.policies, p)
	return nil
}

func (m *MockRepository) RemovePolicy(ctx context.Context, p *casbin.Policy) error {
	return nil
}

func (m *MockRepository) RemoveFilteredPolicy(ctx context.Context, ptype string, fieldIndex int, fieldValues ...string) error {
	return nil
}
