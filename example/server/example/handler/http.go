// Package handler provides HTTP handlers for the example service.
// These handlers are thin wrappers that delegate to the service layer.
// NO business logic should exist here.
package handler

import (
	"github.com/kart-io/sentinel-x/example/server/example/service/helloservice"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
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

	// Validation example routes
	api.Handle("POST", "/users/register", h.RegisterUser)
	api.Handle("POST", "/products", h.CreateProduct)
	api.Handle("POST", "/orders", h.CreateOrder)
}

// HelloRequest is the HTTP request body for POST /hello.
type HelloRequest struct {
	Name string `json:"name" form:"name" validate:"omitempty,min=1,max=50"`
}

// HelloData is the response data for hello endpoint.
type HelloData struct {
	Message string `json:"message"`
}

// ============================================================================
// Validation Example 1: User Registration
// Demonstrates: username, password, email, mobile validation with i18n
// ============================================================================

// RegisterUserRequest is the request body for user registration.
type RegisterUserRequest struct {
	// Username must start with letter, contain letters/numbers/underscore, 3-32 chars
	Username string `json:"username" validate:"required,username"`
	// Password must be at least 8 chars with letter and number
	Password string `json:"password" validate:"required,password"`
	// Email must be valid email format
	Email string `json:"email" validate:"required,email"`
	// Mobile must be valid Chinese mobile number
	Mobile string `json:"mobile" validate:"omitempty,mobile"`
	// Age must be between 1 and 150
	Age int `json:"age" validate:"omitempty,min=1,max=150"`
}

// RegisterUser handles POST /api/v1/users/register
// This demonstrates comprehensive user registration validation.
func (h *HelloHTTPHandler) RegisterUser(c transport.Context) {
	var req RegisterUserRequest

	// Use ShouldBindAndValidate for combined binding and validation
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	// Business logic would go here (delegate to service layer)
	response.OK(c, map[string]interface{}{
		"message":  "User registered successfully",
		"username": req.Username,
		"email":    req.Email,
	})
}

// ============================================================================
// Validation Example 2: Product Creation
// Demonstrates: numeric validation, string length, required fields
// ============================================================================

// CreateProductRequest is the request body for creating a product.
type CreateProductRequest struct {
	// Name is required, 2-100 characters
	Name string `json:"name" validate:"required,min=2,max=100"`
	// Price must be greater than 0
	Price float64 `json:"price" validate:"required,gt=0"`
	// Description is optional, max 1000 characters
	Description string `json:"description" validate:"omitempty,max=1000"`
	// Category must be one of: electronics, clothing, food, other
	Category string `json:"category" validate:"required,oneof=electronics clothing food other"`
	// Stock must be non-negative
	Stock int `json:"stock" validate:"min=0"`
	// SKU must be uppercase alphanumeric, 6-12 chars
	SKU string `json:"sku" validate:"required,alphanum,min=6,max=12"`
}

// CreateProduct handles POST /api/v1/products
// This demonstrates product validation with various constraints.
func (h *HelloHTTPHandler) CreateProduct(c transport.Context) {
	var req CreateProductRequest

	// Use MustBindAndValidate for quick error message
	if errMsg, ok := c.MustBindAndValidate(&req); !ok {
		response.FailWithCode(c, errors.ErrValidationFailed.Code, errMsg)
		return
	}

	// Business logic would go here (delegate to service layer)
	response.OK(c, map[string]interface{}{
		"message": "Product created successfully",
		"product": req,
	})
}

// ============================================================================
// Validation Example 3: Order Creation
// Demonstrates: nested struct validation, slice validation, conditional validation
// ============================================================================

// OrderItem represents an item in the order.
type OrderItem struct {
	// ProductID is required
	ProductID string `json:"product_id" validate:"required,min=1"`
	// Quantity must be at least 1
	Quantity int `json:"quantity" validate:"required,min=1,max=99"`
	// Price must be positive
	Price float64 `json:"price" validate:"required,gt=0"`
}

// CreateOrderRequest is the request body for creating an order.
type CreateOrderRequest struct {
	// UserID is required
	UserID string `json:"user_id" validate:"required"`
	// Items must have at least one item, max 50 items
	Items []OrderItem `json:"items" validate:"required,min=1,max=50,dive"`
	// ShippingAddress is required for physical goods
	ShippingAddress string `json:"shipping_address" validate:"required,min=10,max=500"`
	// Coupon code is optional, must be alphanumeric if provided
	CouponCode string `json:"coupon_code" validate:"omitempty,alphanum,min=4,max=20"`
	// Notes are optional, max 500 chars
	Notes string `json:"notes" validate:"omitempty,max=500"`
}

// CreateOrder handles POST /api/v1/orders
// This demonstrates complex nested validation with slices.
func (h *HelloHTTPHandler) CreateOrder(c transport.Context) {
	var req CreateOrderRequest

	// Use ShouldBindAndValidate for detailed error information
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	// Calculate total
	var total float64
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}

	// Business logic would go here (delegate to service layer)
	response.OK(c, map[string]interface{}{
		"message":    "Order created successfully",
		"user_id":    req.UserID,
		"item_count": len(req.Items),
		"total":      total,
	})
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
	// Use ShouldBindAndValidate for combined binding and validation
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
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
