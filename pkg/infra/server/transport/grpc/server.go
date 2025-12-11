package grpc

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
)

// Re-export types from options package for convenience
type (
	// Options contains gRPC server configuration.
	Options = grpcopts.Options
	// Option is a function that configures Options.
	Option = grpcopts.Option
)

// Re-export option functions
var (
	NewOptions         = grpcopts.NewOptions
	WithAddr           = grpcopts.WithAddr
	WithTimeout        = grpcopts.WithTimeout
	WithMaxRecvMsgSize = grpcopts.WithMaxRecvMsgSize
	WithMaxSendMsgSize = grpcopts.WithMaxSendMsgSize
	WithReflection     = grpcopts.WithReflection
)

// Server is the gRPC server implementation.
// It is designed to be compatible with Kratos lifecycle management.
type Server struct {
	opts     *grpcopts.Options
	server   *grpc.Server
	listener net.Listener
	services []*transport.GRPCServiceDesc
}

// NewServer creates a new gRPC server with the given options.
func NewServer(opts ...grpcopts.Option) *Server {
	options := grpcopts.NewOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Create gRPC server options
	grpcOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(options.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(options.MaxSendMsgSize),
	}

	return &Server{
		opts:     options,
		server:   grpc.NewServer(grpcOpts...),
		services: make([]*transport.GRPCServiceDesc, 0),
	}
}

// Name returns the server name.
func (s *Server) Name() string {
	return "grpc"
}

// RegisterGRPCService registers a gRPC service.
func (s *Server) RegisterGRPCService(desc *transport.GRPCServiceDesc) error {
	s.services = append(s.services, desc)
	return nil
}

// RegisterService registers a gRPC service using the grpc.ServiceDesc.
// This method provides direct access to the underlying grpc.Server registration.
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
}

// Server returns the underlying grpc.Server.
func (s *Server) Server() *grpc.Server {
	return s.server
}

// Start starts the gRPC server.
func (s *Server) Start(ctx context.Context) error {
	// Register all services
	for _, svc := range s.services {
		if desc, ok := svc.ServiceDesc.(*grpc.ServiceDesc); ok {
			s.server.RegisterService(desc, svc.ServiceImpl)
		}
	}

	// Enable reflection if configured
	if s.opts.EnableReflection {
		reflection.Register(s.server)
	}

	// Create listener
	var err error
	s.listener, err = net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return err
	}

	// Start serving in a goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(s.listener); err != nil {
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

// Stop stops the gRPC server gracefully.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	// Graceful stop with context
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-ctx.Done():
		s.server.Stop() // Force stop if context is cancelled
		return ctx.Err()
	case <-done:
		return nil
	}
}

// Ensure Server implements the required interfaces.
var (
	_ transport.Transport     = (*Server)(nil)
	_ transport.GRPCRegistrar = (*Server)(nil)
)
