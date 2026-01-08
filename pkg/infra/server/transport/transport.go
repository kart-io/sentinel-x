// Package transport provides transport layer interfaces.
package transport

import (
	"context"
)

// Transport represents a transport protocol server.
type Transport interface {
	// Start starts the transport server.
	Start(ctx context.Context) error
	// Stop stops the transport server gracefully.
	Stop(ctx context.Context) error
	// Name returns the transport name (e.g., "http", "grpc").
	Name() string
}

// GRPCRegistrar is the interface for registering gRPC services.
type GRPCRegistrar interface {
	// RegisterGRPCService registers a gRPC service.
	RegisterGRPCService(desc *GRPCServiceDesc) error
}

// GRPCServiceDesc describes a gRPC service for registration.
type GRPCServiceDesc struct {
	// ServiceDesc is the gRPC service descriptor.
	ServiceDesc interface{}
	// ServiceImpl is the gRPC service implementation.
	ServiceImpl interface{}
}
