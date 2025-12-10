// Package http provides HTTP server transport implementation.
package http

import (
	"context"
	"io"
	"net/http"
)

// FrameworkBridge abstracts HTTP framework-specific operations.
// Each framework (Gin, Echo, Chi, etc.) implements this interface.
// This design minimizes framework coupling - when upgrading frameworks,
// only the bridge implementation needs to change.
type FrameworkBridge interface {
	// Name returns the framework name (e.g., "gin", "echo").
	Name() string

	// Handler returns the http.Handler for use with net/http server.
	Handler() http.Handler

	// AddRoute adds a route handler for the given method and path.
	AddRoute(method, path string, handler BridgeHandler)

	// AddRouteGroup creates a sub-group with the given prefix.
	AddRouteGroup(prefix string) RouteGroup

	// AddMiddleware adds a middleware to the root router.
	AddMiddleware(middleware BridgeMiddleware)

	// SetNotFoundHandler sets the handler for 404 responses.
	SetNotFoundHandler(handler BridgeHandler)

	// SetErrorHandler sets the global error handler.
	SetErrorHandler(handler BridgeErrorHandler)
}

// RouteGroup represents a route group within the framework.
type RouteGroup interface {
	// AddRoute adds a route handler within this group.
	AddRoute(method, path string, handler BridgeHandler)

	// AddRouteGroup creates a nested sub-group.
	AddRouteGroup(prefix string) RouteGroup

	// AddMiddleware adds a middleware to this group.
	AddMiddleware(middleware BridgeMiddleware)
}

// BridgeHandler is the handler function signature for the bridge.
// It receives a framework-agnostic RequestContext.
type BridgeHandler func(ctx *RequestContext)

// BridgeMiddleware is the middleware function signature.
type BridgeMiddleware func(next BridgeHandler) BridgeHandler

// BridgeErrorHandler handles errors in the bridge.
type BridgeErrorHandler func(err error, ctx *RequestContext)

// RequestContext provides a framework-agnostic HTTP context.
// This abstracts away framework-specific context implementations.
type RequestContext struct {
	// request holds the standard HTTP request.
	request *http.Request

	// writer holds the response writer.
	writer http.ResponseWriter

	// params holds URL path parameters.
	params map[string]string

	// rawContext holds the underlying framework context for advanced use.
	rawContext interface{}

	// written indicates if response has been written.
	written bool

	// statusCode holds the response status code.
	statusCode int

	// lang holds the language preference for i18n.
	lang string
}

// NewRequestContext creates a new RequestContext.
func NewRequestContext(r *http.Request, w http.ResponseWriter) *RequestContext {
	return &RequestContext{
		request:    r,
		writer:     w,
		params:     make(map[string]string),
		statusCode: http.StatusOK,
	}
}

// Request returns the underlying context.Context.
func (c *RequestContext) Request() context.Context {
	return c.request.Context()
}

// SetRequest updates the request context.
func (c *RequestContext) SetRequest(ctx context.Context) {
	c.request = c.request.WithContext(ctx)
}

// HTTPRequest returns the underlying *http.Request.
func (c *RequestContext) HTTPRequest() *http.Request {
	return c.request
}

// ResponseWriter returns the underlying http.ResponseWriter.
func (c *RequestContext) ResponseWriter() http.ResponseWriter {
	return c.writer
}

// Param returns the URL path parameter value.
func (c *RequestContext) Param(key string) string {
	return c.params[key]
}

// SetParam sets a URL path parameter (used by bridge implementations).
func (c *RequestContext) SetParam(key, value string) {
	c.params[key] = value
}

// SetParams sets all URL path parameters at once.
func (c *RequestContext) SetParams(params map[string]string) {
	c.params = params
}

// Query returns the query parameter value.
func (c *RequestContext) Query(key string) string {
	return c.request.URL.Query().Get(key)
}

// QueryDefault returns the query parameter value or a default.
func (c *RequestContext) QueryDefault(key, defaultValue string) string {
	if v := c.Query(key); v != "" {
		return v
	}
	return defaultValue
}

// Header returns the request header value.
func (c *RequestContext) Header(key string) string {
	return c.request.Header.Get(key)
}

// SetHeader sets a response header.
func (c *RequestContext) SetHeader(key, value string) {
	c.writer.Header().Set(key, value)
}

// Body returns the request body reader.
func (c *RequestContext) Body() io.ReadCloser {
	return c.request.Body
}

// Status sets the response status code.
func (c *RequestContext) Status(code int) {
	c.statusCode = code
}

// Written returns whether the response has been written.
func (c *RequestContext) Written() bool {
	return c.written
}

// SetRawContext sets the underlying framework context.
func (c *RequestContext) SetRawContext(raw interface{}) {
	c.rawContext = raw
}

// GetRawContext returns the underlying framework context.
func (c *RequestContext) GetRawContext() interface{} {
	return c.rawContext
}

// BridgeFactory is a factory function for creating FrameworkBridge instances.
type BridgeFactory func() FrameworkBridge
