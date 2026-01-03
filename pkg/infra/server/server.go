package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/server/service"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport/grpc"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport/http"
	options "github.com/kart-io/sentinel-x/pkg/options/server"
)

// Options is re-exported from pkg/options/server for convenience.
type Options = options.Options

// Option is re-exported from pkg/options/server for convenience.
type Option = options.Option

// Mode is re-exported from pkg/options/server for convenience.
type Mode = options.Mode

// Re-export mode constants.
const (
	ModeHTTPOnly = options.ModeHTTPOnly
	ModeGRPCOnly = options.ModeGRPCOnly
	ModeBoth     = options.ModeBoth
)

// NewOptions is re-exported from pkg/options/server for convenience.
var NewOptions = options.NewOptions

// Re-export option functions.
var (
	WithMode            = options.WithMode
	WithHTTPOptions     = options.WithHTTPOptions
	WithGRPCOptions     = options.WithGRPCOptions
	WithMiddleware      = options.WithMiddleware
	WithShutdownTimeout = options.WithShutdownTimeout
)

// Manager manages multiple servers (HTTP, gRPC) with unified lifecycle.
type Manager struct {
	opts       *options.Options
	registry   *Registry
	httpServer *http.Server
	grpcServer *grpc.Server
	servers    []Runnable
	mu         sync.Mutex
	started    bool
}

// NewManager creates a new server manager with the given options.
func NewManager(opts ...options.Option) *Manager {
	serverOpts := options.NewOptions()
	for _, opt := range opts {
		opt(serverOpts)
	}

	m := &Manager{
		opts:     serverOpts,
		registry: NewRegistry(),
		servers:  make([]Runnable, 0),
	}

	// Create HTTP server if enabled
	if serverOpts.EnableHTTP() && serverOpts.HTTP != nil {
		m.httpServer = http.NewServer(serverOpts.HTTP, serverOpts.Middleware)
	}

	// Create gRPC server if enabled
	if serverOpts.EnableGRPC() && serverOpts.GRPC != nil {
		m.grpcServer = grpc.NewServer(
			grpc.WithAddr(serverOpts.GRPC.Addr),
			grpc.WithTimeout(serverOpts.GRPC.Timeout),
			grpc.WithMaxRecvMsgSize(serverOpts.GRPC.MaxRecvMsgSize),
			grpc.WithMaxSendMsgSize(serverOpts.GRPC.MaxSendMsgSize),
			grpc.WithReflection(serverOpts.GRPC.EnableReflection),
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

// Wait waits for all servers to be ready with a context timeout.
// This method ensures servers have been started and are accepting connections.
// Note: The actual readiness check is lightweight as servers start immediately
// once Start() completes successfully.
func (m *Manager) Wait(_ context.Context) error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return fmt.Errorf("server manager not started")
	}

	// Check if there are any servers to wait for
	if m.httpServer == nil && m.grpcServer == nil && len(m.servers) == 0 {
		m.mu.Unlock()
		return fmt.Errorf("no servers configured")
	}
	m.mu.Unlock()

	// Servers are considered ready immediately after Start() completes successfully.
	// The Start() method already ensures servers are listening before returning.
	// This Wait() method can be used for additional readiness checks in the future.

	logger.Info("All servers ready")
	return nil
}
