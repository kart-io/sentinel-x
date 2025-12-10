package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	httpopts "github.com/kart-io/sentinel-x/pkg/infra/server/http"
	"github.com/kart-io/sentinel-x/pkg/infra/server/service"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport/grpc"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport/http"
)

// Manager manages multiple servers (HTTP, gRPC) with unified lifecycle.
type Manager struct {
	opts       *Options
	registry   *Registry
	httpServer *http.Server
	grpcServer *grpc.Server
	servers    []Runnable
	mu         sync.Mutex
	started    bool
}

// NewManager creates a new server manager with the given options.
func NewManager(opts ...Option) *Manager {
	options := NewOptions()
	for _, opt := range opts {
		opt(options)
	}

	m := &Manager{
		opts:     options,
		registry: NewRegistry(),
		servers:  make([]Runnable, 0),
	}

	// Create HTTP server if enabled
	if options.EnableHTTP() && options.HTTP != nil {
		m.httpServer = http.NewServer(
			http.WithAddr(options.HTTP.Addr),
			http.WithReadTimeout(options.HTTP.ReadTimeout),
			http.WithWriteTimeout(options.HTTP.WriteTimeout),
			http.WithIdleTimeout(options.HTTP.IdleTimeout),
			http.WithAdapter(httpopts.AdapterType(options.HTTP.Adapter)),
			http.WithMiddleware(func(opts *middleware.Options) {
				*opts = *options.HTTP.Middleware
			}),
		)
	}

	// Create gRPC server if enabled
	if options.EnableGRPC() && options.GRPC != nil {
		m.grpcServer = grpc.NewServer(
			grpc.WithAddr(options.GRPC.Addr),
			grpc.WithTimeout(options.GRPC.Timeout),
			grpc.WithMaxRecvMsgSize(options.GRPC.MaxRecvMsgSize),
			grpc.WithMaxSendMsgSize(options.GRPC.MaxSendMsgSize),
			grpc.WithReflection(options.GRPC.EnableReflection),
		)
	}

	return m
}

// Registry returns the service registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}

// HTTPServer returns the HTTP server (may be nil if not enabled).
func (m *Manager) HTTPServer() *http.Server {
	return m.httpServer
}

// GRPCServer returns the gRPC server (may be nil if not enabled).
func (m *Manager) GRPCServer() *grpc.Server {
	return m.grpcServer
}

// RegisterService registers a service with unified HTTP and gRPC support.
func (m *Manager) RegisterService(svc service.Service, httpHandler transport.HTTPHandler, grpcDesc *transport.GRPCServiceDesc) error {
	return m.registry.RegisterService(svc, httpHandler, grpcDesc)
}

// RegisterHTTP registers an HTTP-only handler.
func (m *Manager) RegisterHTTP(svc service.Service, handler transport.HTTPHandler) error {
	return m.registry.RegisterHTTP(svc, handler)
}

// RegisterGRPC registers a gRPC-only service.
func (m *Manager) RegisterGRPC(svc service.Service, desc *transport.GRPCServiceDesc) error {
	return m.registry.RegisterGRPC(svc, desc)
}

// AddServer adds a custom server to the manager.
func (m *Manager) AddServer(server Runnable) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers = append(m.servers, server)
}

// Start starts all servers.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return fmt.Errorf("server manager already started")
	}
	m.started = true
	m.mu.Unlock()

	// Apply registry to servers
	if m.httpServer != nil {
		if err := m.registry.ApplyToHTTP(m.httpServer); err != nil {
			return fmt.Errorf("failed to apply HTTP handlers: %w", err)
		}
	}

	if m.grpcServer != nil {
		if err := m.registry.ApplyToGRPC(m.grpcServer); err != nil {
			return fmt.Errorf("failed to apply gRPC services: %w", err)
		}
	}

	// Initialize services
	for _, svc := range m.registry.GetAllServices() {
		if init, ok := svc.(service.Initializable); ok {
			if err := init.Init(ctx); err != nil {
				return fmt.Errorf("failed to initialize service %s: %w", svc.ServiceName(), err)
			}
		}
	}

	// Start HTTP server
	if m.httpServer != nil {
		if err := m.httpServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}
		logger.Infow("HTTP server started", "addr", m.opts.HTTP.Addr)
	}

	// Start gRPC server
	if m.grpcServer != nil {
		if err := m.grpcServer.Start(ctx); err != nil {
			if m.httpServer != nil {
				_ = m.httpServer.Stop(ctx)
			}
			return fmt.Errorf("failed to start gRPC server: %w", err)
		}
		logger.Infow("gRPC server started", "addr", m.opts.GRPC.Addr)
	}

	// Start custom servers
	for _, server := range m.servers {
		if err := server.Start(ctx); err != nil {
			if m.grpcServer != nil {
				_ = m.grpcServer.Stop(ctx)
			}
			if m.httpServer != nil {
				_ = m.httpServer.Stop(ctx)
			}
			return fmt.Errorf("failed to start server %s: %w", server.Name(), err)
		}
		logger.Infow("Custom server started", "name", server.Name())
	}

	return nil
}

// Stop stops all servers gracefully.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	var errs []error

	// Stop custom servers first
	for _, server := range m.servers {
		if err := server.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop server %s: %w", server.Name(), err))
		}
	}

	// Stop HTTP server
	if m.httpServer != nil {
		if err := m.httpServer.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop HTTP server: %w", err))
		}
		logger.Info("HTTP server stopped")
	}

	// Stop gRPC server
	if m.grpcServer != nil {
		if err := m.grpcServer.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop gRPC server: %w", err))
		}
		logger.Info("gRPC server stopped")
	}

	// Close services
	for _, svc := range m.registry.GetAllServices() {
		if closer, ok := svc.(service.Closeable); ok {
			if err := closer.Close(ctx); err != nil {
				errs = append(errs, fmt.Errorf("failed to close service %s: %w", svc.ServiceName(), err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}
	return nil
}

// Run starts all servers and waits for shutdown signal.
func (m *Manager) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start servers
	if err := m.Start(ctx); err != nil {
		return err
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), m.opts.ShutdownTimeout)
	defer shutdownCancel()

	return m.Stop(shutdownCtx)
}

// Wait waits for all servers to be ready.
func (m *Manager) Wait(ctx context.Context) error {
	return nil
}
