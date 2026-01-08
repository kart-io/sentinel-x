// Package transport provides transport layer interfaces for HTTP and gRPC.
package transport

import (
	"context"

	"github.com/kart-io/sentinel-x/pkg/infra/server/service"
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

// HTTPRegistrar is the interface for registering HTTP routes.
// Deprecated: This interface assumes the old adapter pattern. Use *gin.Engine directly.
type HTTPRegistrar interface {
	// RegisterHTTPHandler registers an HTTP handler for the given service.
	RegisterHTTPHandler(svc service.Service, handler HTTPHandler) error
}

// GRPCRegistrar is the interface for registering gRPC services.
type GRPCRegistrar interface {
	// RegisterGRPCService registers a gRPC service.
	RegisterGRPCService(desc *GRPCServiceDesc) error
}

// HTTPHandler represents an HTTP handler that can register routes.
// Deprecated: Use *gin.Engine directly for routing.
type HTTPHandler interface {
	// RegisterRoutes registers HTTP routes to the given router.
	RegisterRoutes(router Router)
}

// Router is an abstraction over HTTP routers (Gin, Echo, etc.)
// Deprecated: Use *gin.Engine directly.
type Router interface {
	// Handle registers a handler for the given method and path.
	Handle(method, path string, handler HandlerFunc)
	// Group creates a sub-router with the given prefix.
	Group(prefix string) Router
	// Use adds middleware to the router.
	Use(middleware ...MiddlewareFunc)
	// Static serves static files from the given root directory.
	Static(prefix, root string)
}

// HandlerFunc is the HTTP handler function signature.
// Deprecated: Use gin.HandlerFunc.
type HandlerFunc func(Context)

// MiddlewareFunc is the middleware function signature.
// Deprecated: Use gin.HandlerFunc.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Context represents the HTTP request context.
// Deprecated: Use *gin.Context directly.
type Context interface {
	// GetRawContext returns the underlying framework context (gin.Context, echo.Context).
	GetRawContext() interface{}
}

// GRPCServiceDesc describes a gRPC service for registration.
type GRPCServiceDesc struct {
	// ServiceDesc is the gRPC service descriptor.
	ServiceDesc interface{}
	// ServiceImpl is the gRPC service implementation.
	ServiceImpl interface{}
}

// Validator is the interface for request validation.
type Validator interface {
	// Validate validates the given struct.
	Validate(i interface{}) error
}
