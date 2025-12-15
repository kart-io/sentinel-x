// Package transport provides transport layer interfaces for HTTP and gRPC.
package transport

import (
	"context"
	"io"
	"net/http"

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
	// Static serves static files from the given root directory.
	Static(prefix, root string)
	// Mount mounts an http.Handler to the given prefix.
	Mount(prefix string, handler http.Handler)
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
	// Validate validates the given struct using the global validator.
	// Returns nil if validation passes, or *validator.ValidationErrors if validation fails.
	Validate(v interface{}) error
	// ShouldBindAndValidate binds and validates the request body.
	// Returns nil if both binding and validation pass.
	ShouldBindAndValidate(v interface{}) error
	// MustBindAndValidate binds and validates, returning first error message if failed.
	MustBindAndValidate(v interface{}) (string, bool)

	// JSON sends a JSON response.
	JSON(code int, v interface{})
	// String sends a string response.
	String(code int, s string)
	// Error sends an error response.
	Error(code int, err error)

	// GetRawContext returns the underlying framework context (gin.Context, echo.Context).
	// This should only be used when framework-specific features are needed.
	GetRawContext() interface{}

	// Lang returns the language preference from Accept-Language header or query param.
	Lang() string
	// SetLang sets the language for this request context.
	SetLang(lang string)
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
