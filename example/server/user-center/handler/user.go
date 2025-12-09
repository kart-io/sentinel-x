package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/kart-io/sentinel-x/example/server/user-center/api/v1"
	"github.com/kart-io/sentinel-x/example/server/user-center/service/userservice"
	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/response"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

type UserHandler struct {
	v1.UnimplementedUserServiceServer
	svc *userservice.Service
}

func NewUserHandler(svc *userservice.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

// ============================================================================
// Request Types with Validation
// ============================================================================

// GetUserRequest is the request for getting user by ID.
type GetUserRequest struct {
	// ID must be a valid user ID
	ID string `uri:"id" validate:"required,min=1"`
}

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	// Username must start with letter, 3-32 characters
	Username string `json:"username" validate:"required,username"`
	// Password must be at least 8 chars with letter and number
	Password string `json:"password" validate:"required,password"`
	// Email must be valid email format
	Email string `json:"email" validate:"required,email"`
	// Role must be one of: admin, user, guest
	Role string `json:"role" validate:"required,oneof=admin user guest"`
	// Mobile is optional, must be valid mobile number if provided
	Mobile string `json:"mobile" validate:"omitempty,mobile"`
	// Age must be between 1 and 150 if provided
	Age int `json:"age" validate:"omitempty,min=1,max=150"`
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	// Email must be valid email format if provided
	Email string `json:"email" validate:"omitempty,email"`
	// Role must be one of: admin, user, guest if provided
	Role string `json:"role" validate:"omitempty,oneof=admin user guest"`
	// Mobile must be valid mobile number if provided
	Mobile string `json:"mobile" validate:"omitempty,mobile"`
	// Age must be between 1 and 150 if provided
	Age int `json:"age" validate:"omitempty,min=1,max=150"`
	// Nickname is optional, max 50 characters
	Nickname string `json:"nickname" validate:"omitempty,max=50"`
}

// ListUsersRequest is the query parameters for listing users.
type ListUsersRequest struct {
	// Page number, must be at least 1
	Page int `form:"page" validate:"omitempty,min=1"`
	// PageSize is the number of items per page, 1-100
	PageSize int `form:"page_size" validate:"omitempty,min=1,max=100"`
	// Role filter, must be one of: admin, user, guest
	Role string `form:"role" validate:"omitempty,oneof=admin user guest"`
	// Search keyword for username or email
	Search string `form:"search" validate:"omitempty,max=100"`
}

// BatchDeleteRequest is the request for batch deleting users.
type BatchDeleteRequest struct {
	// IDs is the list of user IDs to delete, 1-100 items
	IDs []string `json:"ids" validate:"required,min=1,max=100,dive,min=1"`
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// GetProfile handles GET /api/v1/users/:id
// This demonstrates path parameter validation.
func (h *UserHandler) GetProfile(c transport.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.Fail(c, errors.ErrInvalidParam.WithMessage("user id is required"))
		return
	}

	user, err := h.svc.GetUser(c.Request(), userID)
	if err != nil {
		response.Fail(c, errors.ErrUserNotFound)
		return
	}

	response.OK(c, user)
}

// CreateUser handles POST /api/v1/users
// This demonstrates user creation with comprehensive validation.
func (h *UserHandler) CreateUser(c transport.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	// In production, delegate to service layer
	response.OK(c, map[string]interface{}{
		"message":  "User created successfully (demo)",
		"username": req.Username,
		"email":    req.Email,
		"role":     req.Role,
	})
}

// UpdateUser handles PUT /api/v1/users/:id
// This demonstrates user update with partial validation.
func (h *UserHandler) UpdateUser(c transport.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.Fail(c, errors.ErrInvalidParam.WithMessage("user id is required"))
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	// In production, delegate to service layer
	response.OK(c, map[string]interface{}{
		"message": "User updated successfully (demo)",
		"id":      userID,
	})
}

// DeleteUser handles DELETE /api/v1/users/:id
// This demonstrates user deletion.
func (h *UserHandler) DeleteUser(c transport.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.Fail(c, errors.ErrInvalidParam.WithMessage("user id is required"))
		return
	}

	// In production, delegate to service layer
	response.OK(c, map[string]string{
		"message": "User deleted successfully (demo)",
		"id":      userID,
	})
}

// ListUsers handles GET /api/v1/users
// This demonstrates query parameter validation with pagination.
func (h *UserHandler) ListUsers(c transport.Context) {
	var req ListUsersRequest
	// Set defaults
	req.Page = 1
	req.PageSize = 20
	req.Role = c.Query("role")
	req.Search = c.Query("search")

	// Validate
	if err := c.Validate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	// In production, delegate to service layer with pagination
	response.OK(c, map[string]interface{}{
		"list": []map[string]interface{}{
			{"id": "1", "username": "admin", "role": "admin"},
			{"id": "2", "username": "user", "role": "user"},
		},
		"total":       2,
		"page":        req.Page,
		"page_size":   req.PageSize,
		"total_pages": 1,
	})
}

// BatchDelete handles POST /api/v1/users/batch-delete
// This demonstrates batch operation with slice validation.
func (h *UserHandler) BatchDelete(c transport.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	// In production, delegate to service layer
	response.OK(c, map[string]interface{}{
		"message": "Users deleted successfully (demo)",
		"count":   len(req.IDs),
	})
}

// AdminAction handles admin-specific actions.
func (h *UserHandler) AdminAction(c transport.Context) {
	response.OK(c, map[string]string{"message": "admin action allowed"})
}

// ============================================================================
// gRPC Handlers
// ============================================================================

// GetUser implements the gRPC GetUser method.
func (h *UserHandler) GetUser(ctx context.Context, req *v1.UserRequest) (*v1.UserResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	user, err := h.svc.GetUser(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &v1.UserResponse{
		Id:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}
