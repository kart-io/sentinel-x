// Package handler provides HTTP handlers for the example service.
// These handlers are thin wrappers that delegate to the service layer.
// NO business logic should exist here.
package handler

import (
	"github.com/kart-io/sentinel-x/example/server/example/service/helloservice"
	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/response"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// HelloHTTPHandler handles HTTP requests for HelloService.
// It delegates all business logic to the underlying service.
type HelloHTTPHandler struct {
	svc *helloservice.Service
}

// NewHelloHTTPHandler creates a new HTTP handler.
func NewHelloHTTPHandler(svc *helloservice.Service) *HelloHTTPHandler {
	return &HelloHTTPHandler{svc: svc}
}

// RegisterRoutes registers HTTP routes.
// This implements the transport.HTTPHandler interface.
func (h *HelloHTTPHandler) RegisterRoutes(router transport.Router) {
	// Create a group for /api/v1
	api := router.Group("/api/v1")

	// Register routes
	api.Handle("GET", "/hello", h.SayHello)
	api.Handle("POST", "/hello", h.SayHelloPost)
	api.Handle("GET", "/hello/:name", h.SayHelloPath)
}

// HelloRequest is the HTTP request body for POST /hello.
type HelloRequest struct {
	Name string `json:"name" form:"name"`
}

// HelloData is the response data for hello endpoint.
type HelloData struct {
	Message string `json:"message"`
}

// SayHello handles GET /api/v1/hello?name=xxx
func (h *HelloHTTPHandler) SayHello(c transport.Context) {
	name := c.Query("name")
	// Delegate to service layer - NO business logic here
	message, err := h.svc.SayHello(c.Request(), name)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.OK(c, &HelloData{Message: message})
}

// SayHelloPost handles POST /api/v1/hello
func (h *HelloHTTPHandler) SayHelloPost(c transport.Context) {
	var req HelloRequest
	if err := c.Bind(&req); err != nil {
		response.Fail(c, errors.ErrInvalidParam.WithMessage("invalid request body"))
		return
	}

	if req.Name == "" {
		req.Name = "World"
	}

	// Delegate to service layer - NO business logic here
	message, err := h.svc.SayHello(c.Request(), req.Name)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.OK(c, &HelloData{Message: message})
}

// SayHelloPath handles GET /api/v1/hello/:name
func (h *HelloHTTPHandler) SayHelloPath(c transport.Context) {
	name := c.Param("name")

	// Delegate to service layer - NO business logic here
	message, err := h.svc.SayHello(c.Request(), name)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.OK(c, &HelloData{Message: message})
}

// Ensure HelloHTTPHandler implements transport.HTTPHandler.
var _ transport.HTTPHandler = (*HelloHTTPHandler)(nil)
