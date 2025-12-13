package handler

import (
	"context"
	"strconv"
	"strings"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/pkg/utils"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/store"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// UserHandler handles user-related HTTP requests and gRPC requests.
type UserHandler struct {
	v1.UnimplementedUserServiceServer
	svc     *biz.UserService
	roleSvc *biz.RoleService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *biz.UserService, roleSvc *biz.RoleService) *UserHandler {
	return &UserHandler{
		svc:     svc,
		roleSvc: roleSvc,
	}
}

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	// Username must start with letter, 3-32 characters
	Username string `json:"username" validate:"required,username"`
	// Password must be at least 8 chars with letter and number
	Password string `json:"password" validate:"required,password"`
	// Email must be valid email format
	Email string `json:"email" validate:"required,email"`
	// Mobile is optional, must be valid mobile number if provided
	Mobile string `json:"mobile" validate:"omitempty,mobile"`
}

// Create handles user creation.
func (h *UserHandler) Create(c transport.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	user := &model.User{
		Username: req.Username,
		Password: req.Password,
		Email:    &req.Email,
		Mobile:   req.Mobile,
		// Status:   1, // Remove this line, it's not in the original struct, assuming default or handled elsewhere
	}

	if err := h.svc.Create(c.Request(), user); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, user)
}

// Update handles user updates.
func (h *UserHandler) Update(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage("username is required"), nil)
		return
	}

	user, err := h.svc.Get(c.Request(), username)
	if err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	var req struct {
		Email  string `json:"email"`
		Mobile string `json:"mobile"`
	}
	if err := c.Bind(&req); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	if req.Email != "" {
		user.Email = &req.Email
	}
	if req.Mobile != "" {
		user.Mobile = req.Mobile
	}

	if err := c.ShouldBindAndValidate(user); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	if err := h.svc.Update(c.Request(), user); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, user)
}

// Delete handles user deletion.
func (h *UserHandler) Delete(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("username is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	if err := h.svc.Delete(c.Request(), username); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, "user deleted")
}

// BatchDeleteRequest is the request for batch deleting users.
type BatchDeleteRequest struct {
	// IDs is the list of usernames/IDs to delete, 1-100 items
	Usernames []string `json:"usernames" validate:"required,min=1,max=100,dive,min=1"`
}

// BatchDelete handles batch deletion of users.
func (h *UserHandler) BatchDelete(c transport.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	for _, username := range req.Usernames {
		if err := h.svc.Delete(c.Request(), username); err != nil {
			utils.WriteResponse(c, err, nil)
			return
		}
	}

	utils.WriteResponse(c, nil, "users deleted")
}

// Get handles retrieving a user by username.
func (h *UserHandler) Get(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("username is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	user, err := h.svc.Get(c.Request(), username)
	if err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, user)
}

// ListUsersRequest is the query parameters for listing users.
type ListUsersRequest struct {
	// Page number, must be at least 1
	Page int `form:"page" validate:"omitempty,min=1"`
	// PageSize is the number of items per page, 1-100
	PageSize int `form:"page_size" validate:"omitempty,min=1,max=100"`
	// Search keyword for username or email
	Search string `form:"search" validate:"omitempty,max=100"`
}

// List handles listing users.
func (h *UserHandler) List(c transport.Context) {
	var req ListUsersRequest
	// Set defaults
	req.Page = 1
	req.PageSize = 10

	// Ignore bind error for optional params
	_ = c.Bind(&req)

	// Manual override if bind failed or not present
	if val, err := strconv.Atoi(c.Query("page")); err == nil && val > 0 {
		req.Page = val
	}
	if val, err := strconv.Atoi(c.Query("page_size")); err == nil && val > 0 {
		req.PageSize = val
	}

	count, users, err := h.svc.List(c.Request(), store.WithPage(req.Page, req.PageSize))
	if err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, response.Page(users, count, req.Page, req.PageSize))
}

// GetProfile handles retrieving the current user's profile.
func (h *UserHandler) GetProfile(c transport.Context) {
	username := auth.SubjectFromContext(c.Request())
	if username == "" {
		resp := response.Err(errors.ErrUnauthorized)
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	user, err := h.svc.Get(c.Request(), username)
	if err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, user)
}

// ChangePasswordRequest is the request body for changing password.
type ChangePasswordRequest struct {
	// OldPassword is the current password
	OldPassword string `json:"old_password" validate:"required,min=6,max=64"`
	// NewPassword must be at least 8 chars with letter and number
	NewPassword string `json:"new_password" validate:"required,password"`
	// ConfirmPassword must match NewPassword
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

// ChangePassword handles password change.
func (h *UserHandler) ChangePassword(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("username is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// Verify old password
	if err := h.svc.ValidatePassword(c.Request(), username, req.OldPassword); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	if err := h.svc.ChangePassword(c.Request(), username, req.NewPassword); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, "password changed")
}

// GetUser implements the gRPC method to get a user by ID.
func (h *UserHandler) GetUser(ctx context.Context, req *v1.UserRequest) (*v1.UserResponse, error) {
	// Note: The proto defines GetUser taking an ID, but our svc.Get takes a username.
	// We might need to handle this mapping or if ID is passed as string.
	// Assuming req.Id is the ID.
	var user *model.User
	id, err := strconv.ParseUint(req.Id, 10, 64)
	if err == nil {
		// Numeric ID, try get by ID
		user, err = h.svc.GetByUserId(ctx, id)
	} else {
		// Non-numeric, treat as username
		user, err = h.svc.Get(ctx, req.Id)
	}

	if err != nil {
		return nil, err
	}

	// Fetch roles
	roles, err := h.roleSvc.GetUserRoles(ctx, user.Username)
	var roleStr string
	if err == nil && len(roles) > 0 {
		var roleCodes []string
		for _, r := range roles {
			roleCodes = append(roleCodes, r.Code)
		}
		roleStr = strings.Join(roleCodes, ",")
	}

	return &v1.UserResponse{
		Id:       strconv.FormatUint(user.ID, 10),
		Username: user.Username,
		Role:     roleStr,
	}, nil
}
