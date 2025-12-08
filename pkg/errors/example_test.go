// Package errors_test demonstrates the usage of the Sentinel-X error code system.
//
// This file provides practical examples of:
//   - Using predefined error codes
//   - Creating custom error codes
//   - HTTP error responses
//   - gRPC error responses
//   - Error handling patterns
package errors_test

import (
	"context"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/response"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// ============================================================================
// Custom Error Codes Example
// ============================================================================

// Define custom error codes for your business service.
// Use MakeCode(service, category, sequence) to generate unique codes.
var (
	// Order Service Errors (Service Code: 20)
	ErrOrderInvalidAmount = errors.Register(&errors.Errno{
		Code:      errors.MakeCode(20, errors.CategoryRequest, 1),
		HTTP:      http.StatusBadRequest,
		GRPCCode:  codes.InvalidArgument,
		MessageEN: "Invalid order amount",
		MessageZH: "订单金额无效",
	})

	ErrOrderNotFound = errors.Register(&errors.Errno{
		Code:      errors.MakeCode(20, errors.CategoryResource, 1),
		HTTP:      http.StatusNotFound,
		GRPCCode:  codes.NotFound,
		MessageEN: "Order not found",
		MessageZH: "订单不存在",
	})

	ErrOrderAlreadyPaid = errors.Register(&errors.Errno{
		Code:      errors.MakeCode(20, errors.CategoryConflict, 1),
		HTTP:      http.StatusConflict,
		GRPCCode:  codes.FailedPrecondition,
		MessageEN: "Order already paid",
		MessageZH: "订单已支付",
	})

	ErrOrderPaymentFailed = errors.Register(&errors.Errno{
		Code:      errors.MakeCode(20, errors.CategoryInternal, 1),
		HTTP:      http.StatusInternalServerError,
		GRPCCode:  codes.Internal,
		MessageEN: "Payment processing failed",
		MessageZH: "支付处理失败",
	})
)

// ============================================================================
// HTTP Error Response Examples
// ============================================================================

// HTTPHandlerExample demonstrates HTTP error handling.
type HTTPHandlerExample struct{}

// CreateOrder handles POST /api/v1/orders
// Returns appropriate error responses based on the error type.
func (h *HTTPHandlerExample) CreateOrder(c transport.Context) {
	var req CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		// Return validation error with custom message
		response.Fail(c, errors.ErrInvalidParam.WithMessage("invalid request body"))
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		response.Fail(c, ErrOrderInvalidAmount.WithMessagef("amount must be positive, got %d", req.Amount))
		return
	}

	// Process order (simulated)
	order, err := processOrder(req)
	if err != nil {
		// Wrap the underlying error
		response.Fail(c, errors.ErrInternal.WithCause(err))
		return
	}

	// Return success
	response.OK(c, order)
}

// GetOrder handles GET /api/v1/orders/:id
func (h *HTTPHandlerExample) GetOrder(c transport.Context) {
	orderID := c.Param("id")

	// Simulate order lookup
	order, err := findOrder(orderID)
	if err != nil {
		// Check error type and return appropriate response
		if errors.IsCode(err, ErrOrderNotFound.Code) {
			response.Fail(c, ErrOrderNotFound)
			return
		}
		response.Fail(c, errors.FromError(err))
		return
	}

	response.OK(c, order)
}

// PayOrder handles POST /api/v1/orders/:id/pay
func (h *HTTPHandlerExample) PayOrder(c transport.Context) {
	orderID := c.Param("id")

	// Check if order exists
	order, err := findOrder(orderID)
	if err != nil {
		response.Fail(c, ErrOrderNotFound)
		return
	}

	// Check if already paid
	if order.Status == "paid" {
		response.Fail(c, ErrOrderAlreadyPaid)
		return
	}

	// Process payment
	if err := processPayment(order); err != nil {
		response.Fail(c, ErrOrderPaymentFailed.WithCause(err))
		return
	}

	response.OKWithMessage(c, "Payment successful", nil)
}

// ============================================================================
// gRPC Error Response Examples
// ============================================================================

// GRPCServiceExample demonstrates gRPC error handling.
type GRPCServiceExample struct{}

// CreateOrder handles gRPC order creation request.
func (s *GRPCServiceExample) CreateOrder(ctx context.Context, req *CreateOrderGRPCRequest) (*OrderGRPCResponse, error) {
	// Validate request
	if req.Amount <= 0 {
		errno := ErrOrderInvalidAmount.WithMessagef("amount must be positive, got %d", req.Amount)
		return nil, ErrnoToGRPCStatus(errno)
	}

	// Process order
	order, err := processOrderGRPC(ctx, req)
	if err != nil {
		return nil, ErrnoToGRPCStatus(errors.FromError(err))
	}

	return order, nil
}

// GetOrder handles gRPC order retrieval request.
func (s *GRPCServiceExample) GetOrder(ctx context.Context, req *GetOrderGRPCRequest) (*OrderGRPCResponse, error) {
	order, err := findOrderGRPC(ctx, req.OrderId)
	if err != nil {
		if errors.IsCode(err, ErrOrderNotFound.Code) {
			return nil, ErrnoToGRPCStatus(ErrOrderNotFound)
		}
		return nil, ErrnoToGRPCStatus(errors.FromError(err))
	}

	return order, nil
}

// ============================================================================
// Error Conversion Utilities
// ============================================================================

// ErrnoToGRPCStatus converts an Errno to a gRPC status error.
func ErrnoToGRPCStatus(e *errors.Errno) error {
	return status.Error(e.GRPCCode, e.MessageEN)
}

// ErrnoToGRPCStatusWithDetails converts an Errno to a gRPC status error with details.
func ErrnoToGRPCStatusWithDetails(e *errors.Errno) error {
	st := status.New(e.GRPCCode, e.MessageEN)
	// You can add custom details here if needed
	// st, _ = st.WithDetails(&errdetails.BadRequest{...})
	return st.Err()
}

// GRPCStatusToErrno converts a gRPC status error to an Errno.
func GRPCStatusToErrno(err error) *errors.Errno {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return errors.ErrInternal.WithCause(err)
	}

	// Map gRPC code to Errno
	switch st.Code() {
	case codes.InvalidArgument:
		return errors.ErrInvalidParam.WithMessage(st.Message())
	case codes.Unauthenticated:
		return errors.ErrUnauthorized.WithMessage(st.Message())
	case codes.PermissionDenied:
		return errors.ErrForbidden.WithMessage(st.Message())
	case codes.NotFound:
		return errors.ErrNotFound.WithMessage(st.Message())
	case codes.AlreadyExists:
		return errors.ErrAlreadyExists.WithMessage(st.Message())
	case codes.ResourceExhausted:
		return errors.ErrTooManyRequests.WithMessage(st.Message())
	case codes.DeadlineExceeded:
		return errors.ErrTimeout.WithMessage(st.Message())
	case codes.Unavailable:
		return errors.ErrServiceUnavailable.WithMessage(st.Message())
	default:
		return errors.ErrInternal.WithMessage(st.Message())
	}
}

// ============================================================================
// Error Handling Patterns
// ============================================================================

// Example 1: Basic error check
func example1() {
	err := someOperation()
	if err != nil {
		// Check for specific error
		if errors.IsCode(err, errors.ErrNotFound.Code) {
			// Handle not found
		}
	}
}

// Example 2: Error wrapping with cause
func example2() error {
	data, err := fetchData()
	if err != nil {
		// Wrap with business context
		return errors.ErrDatabase.WithCause(err).WithMessage("failed to fetch user data")
	}
	_ = data
	return nil
}

// Example 3: Multi-language error messages
func example3(lang string) string {
	err := errors.ErrInvalidParam.WithMessages(
		"Username must be 3-20 characters",
		"用户名必须为3-20个字符",
	)
	return err.Message(lang) // Returns message based on lang
}

// Example 4: Error chain inspection
func example4() {
	err := errors.ErrDatabase.WithCause(someDBError())

	// Get the error code
	code := errors.GetCode(err)

	// Get service and category from code
	service, category, seq := errors.ParseCode(code)
	_, _, _ = service, category, seq

	// Check if it's a client or server error
	if errors.IsClientError(code) {
		// Handle client error
	}
	if errors.IsServerError(code) {
		// Handle server error
	}
}

// Example 5: Legacy code migration
func example5() {
	// Convert legacy code to new code
	newCode := errors.LegacyToNewCode(1001)
	_ = newCode

	// Get Errno from legacy code
	errno := errors.FromLegacyCode(2001)
	_ = errno
}

// ============================================================================
// Helper Types (for compilation)
// ============================================================================

type CreateOrderRequest struct {
	Amount int64 `json:"amount"`
}

type Order struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type CreateOrderGRPCRequest struct {
	Amount int64
}

type GetOrderGRPCRequest struct {
	OrderId string
}

type OrderGRPCResponse struct{}

func processOrder(req CreateOrderRequest) (*Order, error) { return nil, nil }
func findOrder(id string) (*Order, error)                 { return nil, nil }
func processPayment(order *Order) error                   { return nil }
func processOrderGRPC(ctx context.Context, req *CreateOrderGRPCRequest) (*OrderGRPCResponse, error) {
	return nil, nil
}
func findOrderGRPC(ctx context.Context, id string) (*OrderGRPCResponse, error) { return nil, nil }
func someOperation() error                                                     { return nil }
func fetchData() (interface{}, error)                                          { return nil, nil }
func someDBError() error                                                       { return nil }
