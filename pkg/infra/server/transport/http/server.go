package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	"github.com/kart-io/sentinel-x/pkg/infra/server/service"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	options "github.com/kart-io/sentinel-x/pkg/options/server/http"
	apierrors "github.com/kart-io/sentinel-x/pkg/utils/errors"
)

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
	opts     *options.Options
	mwOpts   *mwopts.Options
	engine   *gin.Engine
	server   *http.Server
	handlers []registeredHandler
}

type registeredHandler struct {
	svc     service.Service
	handler transport.HTTPHandler
}

// ginValidator wraps transport.Validator for gin binding.
type ginValidator struct {
	validator transport.Validator
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
		opts:     serverOpts,
		mwOpts:   middlewareOpts,
		engine:   engine,
		handlers: make([]registeredHandler, 0),
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

// RegisterHTTPHandler registers an HTTP handler for a service.
func (s *Server) RegisterHTTPHandler(svc service.Service, handler transport.HTTPHandler) error {
	if handler == nil {
		return errors.New("handler cannot be nil")
	}
	s.handlers = append(s.handlers, registeredHandler{
		svc:     svc,
		handler: handler,
	})
	return nil
}

// Engine returns the underlying gin.Engine.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Router returns the HTTP router (deprecated, use Engine instead).
// Deprecated: Use Engine() to get *gin.Engine directly.
func (s *Server) Router() transport.Router {
	// 返回 nil 或者可以考虑返回一个兼容层
	// 为了保持向后兼容，这里暂时保留方法但返回 nil
	return nil
}

// SetValidator sets the global validator for the server.
func (s *Server) SetValidator(v transport.Validator) {
	binding.Validator = &ginValidator{validator: v}
}

// Adapter returns the underlying adapter (deprecated).
// Deprecated: Adapter abstraction has been removed.
func (s *Server) Adapter() Adapter {
	return nil
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

	// TODO: 这些端点注册函数需要重构为直接接受 *gin.Engine 而非 transport.Router
	// 暂时注释掉，等待中间件层重构完成后再启用

	// Register health endpoints
	// if s.mwOpts.IsEnabled(mwopts.MiddlewareHealth) {
	// 	middleware.RegisterHealthRoutesWithGin(s.engine, *s.mwOpts.Health, nil)
	// }

	// Register metrics endpoint
	// if s.mwOpts.IsEnabled(mwopts.MiddlewareMetrics) {
	// 	middleware.RegisterMetricsRoutesWithGin(s.engine, *s.mwOpts.Metrics)
	// }

	// Register pprof endpoints
	// if s.mwOpts.IsEnabled(mwopts.MiddlewarePprof) {
	// 	middleware.RegisterPprofRoutesWithGin(s.engine, *s.mwOpts.Pprof)
	// }

	// Register version endpoint
	// if s.mwOpts.IsEnabled(mwopts.MiddlewareVersion) {
	// 	middleware.RegisterVersionRoutesWithGin(s.engine, *s.mwOpts.Version)
	// }

	// Register all handlers
	for _, h := range s.handlers {
		// 注意：HTTPHandler接口需要适配为直接使用gin.Engine
		// 这里暂时保留，后续可能需要调整HTTPHandler接口
		if h.handler != nil {
			// 由于移除了Router接口，这里需要跳过或者重新设计
			// 暂时注释掉，因为handler.RegisterRoutes期望transport.Router
			// h.handler.RegisterRoutes(???)
		}
	}

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

// applyMiddleware applies configured middleware to the engine.
// 使用Gin的中间件机制直接注册中间件。
func (s *Server) applyMiddleware(opts *mwopts.Options) {
	// Ensure all sub-options are initialized with defaults
	_ = opts.Complete()

	// Recovery middleware (enabled by default, 最高优先级)
	if opts.IsEnabled(mwopts.MiddlewareRecovery) {
		s.engine.Use(resilience.RecoveryWithOptions(*opts.Recovery, nil))
	}

	// RequestID middleware (enabled by default, 为其他中间件提供 RequestID)
	if opts.IsEnabled(mwopts.MiddlewareRequestID) {
		s.engine.Use(middleware.RequestIDWithOptions(*opts.RequestID, nil))
	}

	// Logger middleware (enabled by default, 依赖 RequestID)
	if opts.IsEnabled(mwopts.MiddlewareLogger) {
		s.engine.Use(observability.LoggerWithOptions(*opts.Logger, nil))
	}

	// Metrics middleware (disabled by default)
	if opts.IsEnabled(mwopts.MiddlewareMetrics) {
		s.engine.Use(middleware.MetricsMiddlewareWithOptions(*opts.Metrics))
	}

	// CORS middleware (disabled by default)
	if opts.IsEnabled(mwopts.MiddlewareCORS) {
		s.engine.Use(middleware.CORSWithOptions(*opts.CORS))
	}

	// Timeout middleware (disabled by default)
	if opts.IsEnabled(mwopts.MiddlewareTimeout) {
		s.engine.Use(middleware.TimeoutWithOptions(*opts.Timeout))
	}

	// Auth middleware (JWT authentication, 在业务逻辑前执行)
	// 注意：Auth 中间件需要运行时注入 Authenticator，不能从配置文件加载
	// 用户需要在自己的代码中手动添加 Auth 中间件
	// 示例：
	//   authmw.AuthWithOptions(
	//       *opts.Auth,
	//       myAuthenticator,  // 运行时依赖
	//       nil,  // errorHandler
	//       nil,  // successHandler
	//   )
}

// Stop stops the HTTP server gracefully.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// Ensure Server implements the required interfaces.
var (
	_ transport.Transport     = (*Server)(nil)
	_ transport.HTTPRegistrar = (*Server)(nil)
)
