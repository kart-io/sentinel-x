package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/kart-io/logger"

	// 导入 middleware 包以触发 init() 注册工厂
	_ "github.com/kart-io/sentinel-x/pkg/infra/middleware"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	options "github.com/kart-io/sentinel-x/pkg/options/server/http"
	apierrors "github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// Validator is the interface for request validation.
type Validator interface {
	// Validate validates the given struct.
	Validate(i interface{}) error
}

// Re-export types from options package for convenience
type (
	// Options contains HTTP server configuration.
	Options = options.Options
	// Option is a function that configures Options.
	Option = options.Option
	// AdapterType represents the HTTP framework adapter type.
	AdapterType = options.AdapterType
)

// Re-export constants
const (
	AdapterGin  = options.AdapterGin
	AdapterEcho = options.AdapterEcho
)

// Re-export option functions
var (
	NewOptions       = options.NewOptions
	WithAddr         = options.WithAddr
	WithReadTimeout  = options.WithReadTimeout
	WithWriteTimeout = options.WithWriteTimeout
	WithIdleTimeout  = options.WithIdleTimeout
	WithAdapter      = options.WithAdapter
)

// Server is the HTTP server implementation.
type Server struct {
	opts   *options.Options
	mwOpts *mwopts.Options
	engine *gin.Engine
	server *http.Server
}

// ginValidator wraps Validator for gin binding.
type ginValidator struct {
	validator Validator
}

func (v *ginValidator) ValidateStruct(obj interface{}) error {
	return v.validator.Validate(obj)
}

func (v *ginValidator) Engine() interface{} {
	return nil
}

// NewServer creates a new HTTP server with the given options.
func NewServer(serverOpts *options.Options, middlewareOpts *mwopts.Options) *Server {
	if serverOpts == nil {
		serverOpts = options.NewOptions()
	}
	if middlewareOpts == nil {
		middlewareOpts = mwopts.NewOptions()
	}

	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 创建 Gin 引擎（不使用默认中间件）
	engine := gin.New()

	s := &Server{
		opts:   serverOpts,
		mwOpts: middlewareOpts,
		engine: engine,
	}

	// 在创建 Server 时就应用中间件
	// 这样所有后续创建的路由组都会继承这些中间件
	s.applyMiddleware(middlewareOpts)

	return s
}

// Name returns the server name.
func (s *Server) Name() string {
	return "http[gin]"
}

// Engine returns the underlying gin.Engine.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// SetValidator sets the global validator for the server.
func (s *Server) SetValidator(v Validator) {
	binding.Validator = &ginValidator{validator: v}
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	// Set default 404 handler with JSON response
	s.engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    apierrors.ErrRouteNotFound.Code,
			"message": apierrors.ErrRouteNotFound.Message,
		})
	})

	// 注意：中间件已在 NewServer 时应用，这里不再重复应用
	// 这是因为 Gin 的 RouterGroup 在创建子组时会复制当前的 handlers
	// 如果中间件在路由注册之后才应用，则不会被子组继承

	// 使用注册机制注册路由
	s.registerRoutes()

	s.server = &http.Server{
		Addr:         s.opts.Addr,
		Handler:      s.engine,
		ReadTimeout:  s.opts.ReadTimeout,
		WriteTimeout: s.opts.WriteTimeout,
		IdleTimeout:  s.opts.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// registerRoutes 使用注册机制注册中间件路由。
func (s *Server) registerRoutes() {
	// 需要注册路由的中间件列表
	routeMiddlewares := []string{
		mwopts.MiddlewareHealth,
		mwopts.MiddlewareMetrics,
		mwopts.MiddlewarePprof,
		mwopts.MiddlewareVersion,
	}

	for _, name := range routeMiddlewares {
		if !s.mwOpts.IsEnabled(name) {
			continue
		}

		registrar, ok := mwopts.GetRouteRegistrar(name)
		if !ok {
			continue
		}

		cfg := s.mwOpts.GetConfig(name)
		if cfg == nil {
			continue
		}

		if err := registrar.RegisterRoutes(s.engine, cfg); err != nil {
			logger.Warnw("failed to register routes",
				"middleware", name,
				"error", err.Error(),
			)
		}
	}
}

// applyMiddleware applies configured middleware to the engine.
// 使用注册机制动态应用中间件，支持灵活的中间件顺序配置。
func (s *Server) applyMiddleware(opts *mwopts.Options) {
	// Ensure all sub-options are initialized with defaults
	_ = opts.Complete()

	// 获取中间件应用顺序（如果未配置则使用默认顺序）
	middlewareOrder := opts.GetMiddlewareOrder()

	// 按照配置的顺序应用中间件
	for _, name := range middlewareOrder {
		if !opts.IsEnabled(name) {
			continue
		}

		// 从注册表获取工厂
		factory, ok := mwopts.GetFactory(name)
		if !ok {
			logger.Debugw("middleware factory not registered, skipping",
				"middleware", name,
			)
			continue
		}

		// 检查是否需要运行时依赖
		if factory.NeedsRuntime() {
			logger.Debugw("middleware requires runtime dependencies, skipping auto-load",
				"middleware", name,
			)
			continue
		}

		// 获取配置
		cfg := opts.GetConfig(name)
		if cfg == nil {
			logger.Debugw("middleware config not found, skipping",
				"middleware", name,
			)
			continue
		}

		// 创建中间件
		handler, err := factory.Create(cfg)
		if err != nil {
			logger.Warnw("failed to create middleware",
				"middleware", name,
				"error", err.Error(),
			)
			continue
		}

		// 应用中间件
		s.engine.Use(handler)
		logger.Debugw("middleware applied",
			"middleware", name,
		)
	}
}

// Stop stops the HTTP server gracefully.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
