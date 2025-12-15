package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	authmw "github.com/kart-io/sentinel-x/pkg/infra/middleware/auth"
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
	WithMiddleware   = options.WithMiddleware
)

// Server is the HTTP server implementation.
type Server struct {
	opts     *options.Options
	adapter  Adapter
	server   *http.Server
	handlers []registeredHandler
}

type registeredHandler struct {
	svc     service.Service
	handler transport.HTTPHandler
}

// NewServer creates a new HTTP server with the given options.
func NewServer(opts ...options.Option) *Server {
	serverOpts := options.NewOptions()
	for _, opt := range opts {
		opt(serverOpts)
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
		adapter:  adapter,
		handlers: make([]registeredHandler, 0),
	}

	// 关键：在创建 Server 时就应用中间件
	// 这样所有后续创建的路由组都会继承这些中间件
	if adapter != nil {
		s.applyMiddleware(adapter.Router(), serverOpts.Middleware)
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
	mwOpts := s.opts.Middleware

	// 注意：中间件已在 NewServer 时应用，这里不再重复应用
	// 这是因为 Gin 的 RouterGroup 在创建子组时会复制当前的 handlers
	// 如果中间件在路由注册之后才应用，则不会被子组继承

	// Register health endpoints
	if !mwOpts.DisableHealth {
		middleware.RegisterHealthRoutes(router, mwOpts.Health)
	}

	// Register metrics endpoint
	if !mwOpts.DisableMetrics {
		middleware.RegisterMetricsRoutesWithOptions(router, mwOpts.Metrics)
	}

	// Register pprof endpoints
	if !mwOpts.DisablePprof {
		middleware.RegisterPprofRoutes(router, mwOpts.Pprof)
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
func (s *Server) applyMiddleware(router transport.Router, opts *mwopts.Options) {
	// Recovery middleware (enabled by default)
	if !opts.DisableRecovery {
		router.Use(middleware.RecoveryWithConfig(middleware.RecoveryConfig{
			EnableStackTrace: opts.Recovery.EnableStackTrace,
			OnPanic:          opts.Recovery.OnPanic,
		}))
	}

	// RequestID middleware (enabled by default)
	if !opts.DisableRequestID {
		config := middleware.RequestIDConfig{
			Header:    opts.RequestID.Header,
			Generator: opts.RequestID.Generator,
		}
		if config.Header == "" {
			config.Header = middleware.HeaderXRequestID
		}
		router.Use(middleware.RequestIDWithConfig(config))
	}

	// Logger middleware (enabled by default)
	if !opts.DisableLogger {
		router.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			SkipPaths:           opts.Logger.SkipPaths,
			Output:              opts.Logger.Output,
			UseStructuredLogger: opts.Logger.UseStructuredLogger,
		}))
	}

	// CORS middleware (disabled by default)
	if !opts.DisableCORS {
		router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     opts.CORS.AllowOrigins,
			AllowMethods:     opts.CORS.AllowMethods,
			AllowHeaders:     opts.CORS.AllowHeaders,
			ExposeHeaders:    opts.CORS.ExposeHeaders,
			AllowCredentials: opts.CORS.AllowCredentials,
			MaxAge:           opts.CORS.MaxAge,
		}))
	}

	// Timeout middleware (disabled by default)
	if !opts.DisableTimeout {
		router.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
			Timeout:   opts.Timeout.Timeout,
			SkipPaths: opts.Timeout.SkipPaths,
		}))
	}

	// Metrics middleware (disabled by default, but endpoint is registered separately)
	if !opts.DisableMetrics {
		router.Use(middleware.MetricsMiddlewareWithOptions(opts.Metrics))
	}

	// Auth middleware (JWT authentication)
	if !opts.DisableAuth && opts.Auth.Authenticator != nil {
		router.Use(authmw.Auth(
			authmw.WithAuthenticator(opts.Auth.Authenticator),
			authmw.WithTokenLookup(opts.Auth.TokenLookup),
			authmw.WithAuthScheme(opts.Auth.AuthScheme),
			authmw.WithSkipPaths(opts.Auth.SkipPaths...),
			authmw.WithSkipPathPrefixes(opts.Auth.SkipPathPrefixes...),
			authmw.WithErrorHandler(opts.Auth.ErrorHandler),
			authmw.WithSuccessHandler(opts.Auth.SuccessHandler),
		))
	}
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
