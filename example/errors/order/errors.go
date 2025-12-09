// Package order provides error codes for Order Service.
//
// This example shows the recommended patterns for defining error codes
// outside of the core errors package. Each business service should have
// its own error definitions file following this pattern.
//
// Key Points:
//  1. Register your service code in init()
//  2. Use errors.NewBuilder or preset builders (NewNotFoundError, etc.)
//  3. Follow the error code format: AABBCCC
//  4. Provide both English and Chinese messages
package order

import (
	"net/http"

	"google.golang.org/grpc/codes"

	"github.com/kart-io/sentinel-x/pkg/errors"
)

// ============================================================================
// Service Code Definition
// ============================================================================

// ServiceOrder is the service code for Order Service.
// Service codes 20-79 are reserved for business services.
// Coordinate with your team to avoid conflicts.
const ServiceOrder = 25

// Register the service on package initialization.
func init() {
	errors.RegisterService(ServiceOrder, "order-service")
}

// ============================================================================
// Order Service Error Codes
// ============================================================================

// Request Errors (Category: 01, HTTP: 400)
var (
	// ErrOrderInvalidAmount indicates the order amount is invalid.
	ErrOrderInvalidAmount = errors.NewRequestError(ServiceOrder, 1).
				Message("Invalid order amount", "订单金额无效").
				MustBuild()

	// ErrOrderInvalidQuantity indicates the order quantity is invalid.
	ErrOrderInvalidQuantity = errors.NewRequestError(ServiceOrder, 2).
				Message("Invalid order quantity", "订单数量无效").
				MustBuild()

	// ErrOrderInvalidProduct indicates the product in order is invalid.
	ErrOrderInvalidProduct = errors.NewRequestError(ServiceOrder, 3).
				Message("Invalid product in order", "订单商品无效").
				MustBuild()

	// ErrOrderInvalidAddress indicates the delivery address is invalid.
	ErrOrderInvalidAddress = errors.NewRequestError(ServiceOrder, 4).
				Message("Invalid delivery address", "配送地址无效").
				MustBuild()

	// ErrOrderInvalidCoupon indicates the coupon is invalid.
	ErrOrderInvalidCoupon = errors.NewRequestError(ServiceOrder, 5).
				Message("Invalid coupon", "优惠券无效").
				MustBuild()
)

// Resource Errors (Category: 04, HTTP: 404)
var (
	// ErrOrderNotFound indicates the order was not found.
	ErrOrderNotFound = errors.NewNotFoundError(ServiceOrder, 1).
				Message("Order not found", "订单不存在").
				MustBuild()

	// ErrOrderItemNotFound indicates the order item was not found.
	ErrOrderItemNotFound = errors.NewNotFoundError(ServiceOrder, 2).
				Message("Order item not found", "订单项不存在").
				MustBuild()

	// ErrProductNotFound indicates the product was not found.
	ErrProductNotFound = errors.NewNotFoundError(ServiceOrder, 3).
				Message("Product not found", "商品不存在").
				MustBuild()

	// ErrInventoryNotFound indicates the inventory record was not found.
	ErrInventoryNotFound = errors.NewNotFoundError(ServiceOrder, 4).
				Message("Inventory not found", "库存记录不存在").
				MustBuild()
)

// Conflict Errors (Category: 05, HTTP: 409)
var (
	// ErrOrderAlreadyExists indicates the order already exists.
	ErrOrderAlreadyExists = errors.NewConflictError(ServiceOrder, 1).
				Message("Order already exists", "订单已存在").
				MustBuild()

	// ErrOrderAlreadyPaid indicates the order has already been paid.
	ErrOrderAlreadyPaid = errors.NewConflictError(ServiceOrder, 2).
				Message("Order already paid", "订单已支付").
				MustBuild()

	// ErrOrderAlreadyCanceled indicates the order has already been canceled.
	ErrOrderAlreadyCanceled = errors.NewConflictError(ServiceOrder, 3).
				Message("Order already canceled", "订单已取消").
				MustBuild()

	// ErrOrderAlreadyCompleted indicates the order has already been completed.
	ErrOrderAlreadyCompleted = errors.NewConflictError(ServiceOrder, 4).
					Message("Order already completed", "订单已完成").
					MustBuild()

	// ErrInsufficientInventory indicates insufficient inventory.
	ErrInsufficientInventory = errors.NewConflictError(ServiceOrder, 5).
					Message("Insufficient inventory", "库存不足").
					MustBuild()
)

// Internal Errors (Category: 07, HTTP: 500)
var (
	// ErrOrderCreateFailed indicates order creation failed.
	ErrOrderCreateFailed = errors.NewInternalError(ServiceOrder, 1).
				Message("Failed to create order", "订单创建失败").
				MustBuild()

	// ErrOrderUpdateFailed indicates order update failed.
	ErrOrderUpdateFailed = errors.NewInternalError(ServiceOrder, 2).
				Message("Failed to update order", "订单更新失败").
				MustBuild()

	// ErrPaymentProcessingFailed indicates payment processing failed.
	ErrPaymentProcessingFailed = errors.NewInternalError(ServiceOrder, 3).
					Message("Payment processing failed", "支付处理失败").
					MustBuild()

	// ErrInventoryUpdateFailed indicates inventory update failed.
	ErrInventoryUpdateFailed = errors.NewInternalError(ServiceOrder, 4).
					Message("Inventory update failed", "库存更新失败").
					MustBuild()
)

// ============================================================================
// Custom Error with Full Builder Pattern
// ============================================================================

// ErrOrderExpired is an example of using the full builder pattern
// for more complex error definitions.
var ErrOrderExpired = errors.NewBuilder(ServiceOrder, errors.CategoryConflict, 10).
	HTTP(http.StatusGone).          // Use 410 Gone instead of 409
	GRPC(codes.FailedPrecondition). // Use FailedPrecondition instead of AlreadyExists
	Message("Order has expired", "订单已过期").
	MustBuild()

	// ============================================================================
	// Usage Examples
	// ============================================================================

	// Example usage in service layer:
	//
	//	func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	//	    order, err := s.repo.FindByID(ctx, orderID)
	//	    if err != nil {
	//	        if errors.Is(err, gorm.ErrRecordNotFound) {
	//	            return nil, orderrors.ErrOrderNotFound.WithMessagef("order %s not found", orderID)
	//	        }
	//	        return nil, errors.ErrDatabase.WithCause(err)
	//	    }
	//	    return order, nil
	//	}
	//
	//	func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	//	    // Validate request
	//	    if req.Amount <= 0 {
	//	        return nil, orderrors.ErrOrderInvalidAmount
	//	    }
	//
	//	    // Check inventory
	//	    if !s.inventory.HasStock(req.ProductID, req.Quantity) {
	//	        return nil, orderrors.ErrInsufficientInventory.WithMessagef(
	//	            "product %s only has %d in stock",
	//	            req.ProductID,
	//	            s.inventory.GetStock(req.ProductID),
	//	        )
	//	    }
	//
	//	    // Create order
	//	    order, err := s.repo.Create(ctx, req)
	//	    if err != nil {
	//	        return nil, orderrors.ErrOrderCreateFailed.WithCause(err)
	//	    }
	//
	//	    return order, nil
	//	}

	// Example usage in HTTP handler:
	//
	//	func (h *OrderHandler) GetOrder(c transport.Context) {
	//	    orderID := c.Param("id")
	//
	//	    order, err := h.svc.GetOrder(c.Request().Context(), orderID)
	//	    if err != nil {
	//	        response.FailWithErrno(c, errors.FromError(err))
	//	        return
	//	    }
	//
	//	    response.OK(c, order)
	//	}

	// Example usage in gRPC handler:
	//
	//	func (s *OrderGRPCServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	//	    order, err := s.svc.GetOrder(ctx, req.OrderId)
	//	    if err != nil {
	//	        errno := errors.FromError(err)
	//	        return nil, status.Error(errno.GRPCCode, errno.MessageEN)
	//	    }
	//	    return toProto(order), nil
	//	}
