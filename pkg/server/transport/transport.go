// Package transport provides transport layer interfaces for HTTP and gRPC.
package transport

import (
	"context"
	"io"
	"net/http"

	"github.com/kart-io/sentinel-x/pkg/server/service"
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
type HTTPHandler interface {
	// RegisterRoutes registers HTTP routes to the given router.
	RegisterRoutes(router Router)
}

// Router is an abstraction over HTTP routers (Gin, Echo, etc.)
type Router interface {
	// Handle registers a handler for the given method and path.
	Handle(method, path string, handler HandlerFunc)
	// Group creates a sub-router with the given prefix.
	Group(prefix string) Router
	// Use adds middleware to the router.
	Use(middleware ...MiddlewareFunc)
}

// HandlerFunc is the HTTP handler function signature.
type HandlerFunc func(Context)

// MiddlewareFunc is the middleware function signature.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Context represents the HTTP request context.
// This interface is framework-agnostic and can be implemented by any HTTP framework.
type Context interface {
	// Request returns the underlying request context.
	Request() context.Context
	// SetRequest sets the request context.
	SetRequest(ctx context.Context)

	// HTTPRequest returns the underlying *http.Request.
	HTTPRequest() *http.Request
	// ResponseWriter returns the underlying http.ResponseWriter.
	ResponseWriter() http.ResponseWriter
	// Body returns the request body reader.
	Body() io.ReadCloser

	// Param returns the URL path parameter value.
	Param(key string) string
	// Query returns the query parameter value.
	Query(key string) string
	// Header returns the request header value.
	Header(key string) string
	// SetHeader sets a response header.
	SetHeader(key, value string)

	// Bind binds the request body to the given struct.
	Bind(v interface{}) error
	// JSON sends a JSON response.
	JSON(code int, v interface{})
	// String sends a string response.
	String(code int, s string)
	// Error sends an error response.
	Error(code int, err error)

	// GetRawContext returns the underlying framework context (gin.Context, echo.Context).
	// This should only be used when framework-specific features are needed.
	GetRawContext() interface{}
}

// GRPCServiceDesc describes a gRPC service for registration.
type GRPCServiceDesc struct {
	// ServiceDesc is the gRPC service descriptor.
	ServiceDesc interface{}
	// ServiceImpl is the gRPC service implementation.
	ServiceImpl interface{}
}
