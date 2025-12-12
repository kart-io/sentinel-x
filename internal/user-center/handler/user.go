package handler

import (
	"net/http"
	"strconv"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	svc *biz.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *biz.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Create handles user creation.
func (h *UserHandler) Create(c transport.Context) {
	var req struct {
		model.User
		Password string `json:"password"`
	}
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	user := req.User
	user.Password = req.Password

	if err := h.svc.Create(c.Request(), &user); err != nil {
		logger.Errorf("failed to create user: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(user)
	defer response.Release(resp)
	c.JSON(http.StatusCreated, resp)
}

// UpdateUserRequest 定义用户更新请求的 DTO，明确排除敏感字段
type UpdateUserRequest struct {
	Email  *string `json:"email" validate:"omitempty,email"`
	Avatar string  `json:"avatar"`
	Mobile string  `json:"mobile"`
	Status *int    `json:"status" validate:"omitempty,min=0,max=1"`
}

// Update 处理用户更新请求
func (h *UserHandler) Update(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("用户名不能为空"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// 获取现有用户
	existingUser, err := h.svc.Get(c.Request(), username)
	if err != nil {
		resp := response.Err(errors.ErrUserNotFound)
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// 使用专用 DTO 接收请求，禁止更新敏感字段
	var req UpdateUserRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// 选择性更新字段，确保 Password 和 Username 永远不会被更新
	if req.Email != nil {
		existingUser.Email = req.Email
	}
	if req.Avatar != "" {
		existingUser.Avatar = req.Avatar
	}
	if req.Mobile != "" {
		existingUser.Mobile = req.Mobile
	}
	if req.Status != nil {
		existingUser.Status = *req.Status
	}

	if err := h.svc.Update(c.Request(), existingUser); err != nil {
		logger.Errorf("更新用户失败: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(existingUser)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
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
		logger.Errorf("failed to delete user: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.SuccessWithMessage("user deleted", nil)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// Get handles retrieving a single user.
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
		resp := response.Err(errors.ErrUserNotFound)
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(user)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// List handles listing users.
func (h *UserHandler) List(c transport.Context) {
	offsetStr := c.Query("offset")
	limitStr := c.Query("limit")

	offset := 0
	limit := 10

	if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
		offset = val
	}
	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		limit = val
	}

	count, users, err := h.svc.List(c.Request(), offset, limit)
	if err != nil {
		logger.Errorf("failed to list users: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Page(users, count, offset/limit+1, limit)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
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

	var req struct {
		NewPassword string `json:"newPassword"`
	}
	if err := c.Bind(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	if err := h.svc.ChangePassword(c.Request(), username, req.NewPassword); err != nil {
		logger.Errorf("failed to change password: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.SuccessWithMessage("password changed", nil)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// GetProfile handles retrieving the current user's profile.
func (h *UserHandler) GetProfile(c transport.Context) {
	username := auth.UserIDFromContext(c.Request())

	user, err := h.svc.Get(c.Request(), username)
	if err != nil {
		resp := response.Err(errors.ErrUserNotFound)
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(user)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}
