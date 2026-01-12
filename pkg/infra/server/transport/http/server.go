package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/performance"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
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

	// Register health endpoints
	if s.mwOpts.IsEnabled(mwopts.MiddlewareHealth) {
		middleware.RegisterHealthRoutesWithOptions(s.engine, *s.mwOpts.Health, nil)
	}

	// Register metrics endpoint
	if s.mwOpts.IsEnabled(mwopts.MiddlewareMetrics) {
		middleware.RegisterMetricsRoutesWithOptions(s.engine, *s.mwOpts.Metrics)
	}

	// Register pprof endpoints
	if s.mwOpts.IsEnabled(mwopts.MiddlewarePprof) {
		middleware.RegisterPprofRoutesWithOptions(s.engine, *s.mwOpts.Pprof)
	}

	// Register version endpoint
	if s.mwOpts.IsEnabled(mwopts.MiddlewareVersion) {
		middleware.RegisterVersionRoutes(s.engine, *s.mwOpts.Version)
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
// 根据配置的 middleware 字段动态应用中间件，支持灵活的中间件顺序配置。
func (s *Server) applyMiddleware(opts *mwopts.Options) {
	// Ensure all sub-options are initialized with defaults
	_ = opts.Complete()

	// 获取中间件应用顺序（如果未配置则使用默认顺序）
	middlewareOrder := opts.GetMiddlewareOrder()

	// 中间件构造函数映射表
	middlewareFactories := map[string]func(){
		mwopts.MiddlewareRecovery: func() {
			if opts.IsEnabled(mwopts.MiddlewareRecovery) {
				s.engine.Use(resilience.RecoveryWithOptions(*opts.Recovery, nil))
			}
		},
		mwopts.MiddlewareRequestID: func() {
			if opts.IsEnabled(mwopts.MiddlewareRequestID) {
				s.engine.Use(middleware.RequestIDWithOptions(*opts.RequestID, nil))
			}
		},
		mwopts.MiddlewareLogger: func() {
			if opts.IsEnabled(mwopts.MiddlewareLogger) {
				s.engine.Use(observability.LoggerWithOptions(*opts.Logger, nil))
			}
		},
		mwopts.MiddlewareMetrics: func() {
			if opts.IsEnabled(mwopts.MiddlewareMetrics) {
				s.engine.Use(middleware.MetricsMiddlewareWithOptions(*opts.Metrics))
			}
		},
		mwopts.MiddlewareCORS: func() {
			if opts.IsEnabled(mwopts.MiddlewareCORS) {
				s.engine.Use(middleware.CORSWithOptions(*opts.CORS))
			}
		},
		mwopts.MiddlewareTimeout: func() {
			if opts.IsEnabled(mwopts.MiddlewareTimeout) {
				s.engine.Use(middleware.TimeoutWithOptions(*opts.Timeout))
			}
		},
		mwopts.MiddlewareBodyLimit: func() {
			if opts.IsEnabled(mwopts.MiddlewareBodyLimit) {
				s.engine.Use(resilience.BodyLimitWithOptions(*opts.BodyLimit))
			}
		},
		mwopts.MiddlewareCircuitBreaker: func() {
			if opts.IsEnabled(mwopts.MiddlewareCircuitBreaker) {
				s.engine.Use(resilience.CircuitBreakerWithOptions(*opts.CircuitBreaker))
			}
		},
		mwopts.MiddlewareSecurityHeaders: func() {
			if opts.IsEnabled(mwopts.MiddlewareSecurityHeaders) {
				s.engine.Use(middleware.SecurityHeadersWithOptions(*opts.SecurityHeaders))
			}
		},
		mwopts.MiddlewareCompression: func() {
			if opts.IsEnabled(mwopts.MiddlewareCompression) {
				s.engine.Use(performance.CompressionWithOptions(*opts.Compression))
			}
		},
		// 注意：Auth, Authz, RateLimit 等中间件需要运行时依赖，不能从配置文件直接加载
		// 用户需要在业务代码中手动添加
	}

	// 按照配置的顺序应用中间件
	for _, name := range middlewareOrder {
		if factory, exists := middlewareFactories[name]; exists {
			factory()
		}
	}
}

// Stop stops the HTTP server gracefully.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
