package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	"github.com/kart-io/sentinel-x/pkg/infra/server/service"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	options "github.com/kart-io/sentinel-x/pkg/options/server/http"
	apierrors "github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
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
	adapter  Adapter
	server   *http.Server
	handlers []registeredHandler
}

type registeredHandler struct {
	svc     service.Service
	handler transport.HTTPHandler
}

// NewServer creates a new HTTP server with the given options.
func NewServer(serverOpts *options.Options, middlewareOpts *mwopts.Options) *Server {
	if serverOpts == nil {
		serverOpts = options.NewOptions()
	}
	if middlewareOpts == nil {
		middlewareOpts = mwopts.NewOptions()
	}

	adapter := GetAdapter(serverOpts.Adapter)
	if adapter == nil {
		// Default to gin if no adapter is registered
		adapter = GetAdapter(options.AdapterGin)
	}

	// Note: adapter may still be nil if no adapters are registered
	// This will be checked in Start()

	s := &Server{
		opts:     serverOpts,
		mwOpts:   middlewareOpts,
		adapter:  adapter,
		handlers: make([]registeredHandler, 0),
	}

	// 关键：在创建 Server 时就应用中间件
	// 这样所有后续创建的路由组都会继承这些中间件
	if adapter != nil {
		s.applyMiddleware(adapter.Router(), middlewareOpts)
	}

	return s
}

// Name returns the server name.
func (s *Server) Name() string {
	if s.adapter == nil {
		return "http[uninitialized]"
	}
	return fmt.Sprintf("http[%s]", s.adapter.Name())
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

// Router returns the HTTP router.
func (s *Server) Router() transport.Router {
	if s.adapter == nil {
		return nil
	}
	return s.adapter.Router()
}

// SetValidator sets the global validator for the server.
func (s *Server) SetValidator(v transport.Validator) {
	if s.adapter != nil {
		s.adapter.SetValidator(v)
	}
}

// Adapter returns the underlying adapter.
func (s *Server) Adapter() Adapter {
	return s.adapter
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	// Check if adapter is initialized
	if s.adapter == nil {
		return errors.New("HTTP server adapter not initialized: no adapter registered for the configured type. " +
			"Make sure to import the adapter package (e.g., _ \"github.com/kart-io/sentinel-x/pkg/infra/server/transport/http/gin\")")
	}

	// Set default 404 handler with JSON response
	s.adapter.SetNotFoundHandler(func(c transport.Context) {
		response.Fail(c, apierrors.ErrRouteNotFound)
	})

	router := s.adapter.Router()
	mwOpts := s.mwOpts

	// 注意：中间件已在 NewServer 时应用，这里不再重复应用
	// 这是因为 Gin 的 RouterGroup 在创建子组时会复制当前的 handlers
	// 如果中间件在路由注册之后才应用，则不会被子组继承

	// Register health endpoints
	if mwOpts.IsEnabled(mwopts.MiddlewareHealth) {
		middleware.RegisterHealthRoutesWithOptions(router, *mwOpts.Health, nil)
	}

	// Register metrics endpoint
	if mwOpts.IsEnabled(mwopts.MiddlewareMetrics) {
		middleware.RegisterMetricsRoutesWithOptions(router, *mwOpts.Metrics)
	}

	// Register pprof endpoints
	if mwOpts.IsEnabled(mwopts.MiddlewarePprof) {
		middleware.RegisterPprofRoutesWithOptions(router, *mwOpts.Pprof)
	}

	// Register version endpoint
	if mwOpts.IsEnabled(mwopts.MiddlewareVersion) {
		middleware.RegisterVersionRoutes(router, *mwOpts.Version)
	}

	// Register all handlers
	for _, h := range s.handlers {
		h.handler.RegisterRoutes(router)
	}

	s.server = &http.Server{
		Addr:         s.opts.Addr,
		Handler:      s.adapter.Handler(),
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

// applyMiddleware applies configured middleware to the router.
// 使用优先级注册器自动管理中间件执行顺序。
func (s *Server) applyMiddleware(router transport.Router, opts *mwopts.Options) {
	// Ensure all sub-options are initialized with defaults
	_ = opts.Complete()

	// 创建中间件注册器
	registrar := middleware.NewRegistrar()

	// Recovery middleware (enabled by default, 最高优先级)
	registrar.RegisterIf(
		opts.IsEnabled(mwopts.MiddlewareRecovery),
		"recovery",
		middleware.PriorityRecovery,
		resilience.RecoveryWithOptions(*opts.Recovery, nil),
	)

	// RequestID middleware (enabled by default, 为其他中间件提供 RequestID)
	registrar.RegisterIf(
		opts.IsEnabled(mwopts.MiddlewareRequestID),
		"request-id",
		middleware.PriorityRequestID,
		middleware.RequestIDWithOptions(*opts.RequestID, nil),
	)

	// Logger middleware (enabled by default, 依赖 RequestID)
	registrar.RegisterIf(
		opts.IsEnabled(mwopts.MiddlewareLogger),
		"logger",
		middleware.PriorityLogger,
		observability.LoggerWithOptions(*opts.Logger, nil),
	)

	// Metrics middleware (disabled by default)
	registrar.RegisterIf(
		opts.IsEnabled(mwopts.MiddlewareMetrics),
		"metrics",
		middleware.PriorityMetrics,
		middleware.MetricsMiddlewareWithOptions(*opts.Metrics),
	)

	// CORS middleware (disabled by default)
	registrar.RegisterIf(
		opts.IsEnabled(mwopts.MiddlewareCORS),
		"cors",
		middleware.PriorityCORS,
		middleware.CORSWithOptions(*opts.CORS),
	)

	// Timeout middleware (disabled by default)
	registrar.RegisterIf(
		opts.IsEnabled(mwopts.MiddlewareTimeout),
		"timeout",
		middleware.PriorityTimeout,
		middleware.TimeoutWithOptions(*opts.Timeout),
	)

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
	// registrar.RegisterIf(
	//   opts.IsEnabled(mwopts.MiddlewareAuth),
	//   "auth",
	//   middleware.PriorityAuth,
	//   ... // 需要用户手动注入
	// )

	// 按优先级顺序应用所有中间件
	registrar.Apply(router)
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
