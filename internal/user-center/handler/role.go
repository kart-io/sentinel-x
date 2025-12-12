package handler

import (
	stderrors "errors"
	"net/http"
	"strconv"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// RoleHandler handles role-related HTTP requests.
type RoleHandler struct {
	svc *biz.RoleService
}

// NewRoleHandler creates a new RoleHandler.
func NewRoleHandler(svc *biz.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// CreateRoleRequest is the request body for creating a role.
type CreateRoleRequest struct {
	Code        string `json:"code" validate:"required,min=3,max=32"`
	Name        string `json:"name" validate:"required,min=2,max=64"`
	Description string `json:"description" validate:"omitempty,max=255"`
}

// Create handles role creation.
func (h *RoleHandler) Create(c transport.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	role := &model.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Status:      1, // Default enabled
	}

	if err := h.svc.Create(c.Request(), role); err != nil {
		logger.Errorf("failed to create role: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(role)
	defer response.Release(resp)
	c.JSON(http.StatusCreated, resp)
}

// Update handles role updates.
func (h *RoleHandler) Update(c transport.Context) {
	code := c.Param("code")
	if code == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("role code is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// Fetch existing role first
	role, err := h.svc.Get(c.Request(), code)
	if err != nil {
		resp := response.Err(errors.ErrNotFound.WithMessage("role not found"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// Bind new data
	// UpdateRoleRequest is the request body for updating a role.
	type UpdateRoleRequest struct {
		Name        string `json:"name" validate:"omitempty,min=2,max=64"`
		Description string `json:"description" validate:"omitempty,max=255"`
		Status      *int   `json:"status" validate:"omitempty,oneof=0 1"`
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.Status != nil {
		role.Status = *req.Status
	}

	if err := h.svc.Update(c.Request(), role); err != nil {
		logger.Errorf("failed to update role: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(role)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// Delete handles role deletion.
func (h *RoleHandler) Delete(c transport.Context) {
	code := c.Param("code")
	if code == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("role code is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	if err := h.svc.Delete(c.Request(), code); err != nil {
		logger.Errorf("failed to delete role: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.SuccessWithMessage("role deleted", nil)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// Get handles retrieving a role.
func (h *RoleHandler) Get(c transport.Context) {
	code := c.Param("code")
	if code == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("role code is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	role, err := h.svc.Get(c.Request(), code)
	if err != nil {
		resp := response.Err(errors.ErrNotFound)
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(role)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// List handles listing roles.
func (h *RoleHandler) List(c transport.Context) {
	// ListRolesRequest is the query parameters for listing roles.
	type ListRolesRequest struct {
		Page     int `form:"page" validate:"omitempty,min=1"`
		PageSize int `form:"page_size" validate:"omitempty,min=1,max=100"`
	}

	var req ListRolesRequest
	req.Page = 1
	req.PageSize = 10

	// Use ShouldBindQuery for query params, ignore error as we have defaults/manual overrides
	_ = c.Bind(&req)

	// Manual override if bind failed or not present
	if val, err := strconv.Atoi(c.Query("page")); err == nil && val > 0 {
		req.Page = val
	}
	if val, err := strconv.Atoi(c.Query("page_size")); err == nil && val > 0 {
		req.PageSize = val
	}

	offset := (req.Page - 1) * req.PageSize
	limit := req.PageSize

	count, roles, err := h.svc.List(c.Request(), offset, limit)
	if err != nil {
		logger.Errorf("failed to list roles: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Page(roles, count, offset/limit+1, limit)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// AssignRole handles assigning a role to a user.
func (h *RoleHandler) AssignRole(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("username is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// AssignRoleRequest is the request body for assigning a role.
	type AssignRoleRequest struct {
		RoleCode string `json:"role_code" validate:"required"`
	}

	var req AssignRoleRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	if err := h.svc.AssignRoleToUser(c.Request(), username, req.RoleCode); err != nil {
		logger.Errorf("failed to assign role: %v", err)
		errResp := errors.ErrInternal.WithMessage(err.Error())
		if stderrors.Is(err, errors.ErrAlreadyExists) {
			errResp = errors.ErrAlreadyExists
		}
		resp := response.Err(errResp)
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.SuccessWithMessage("role assigned", nil)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// GetUserRoles handles retrieving roles for a user.
func (h *RoleHandler) GetUserRoles(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("username is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	roles, err := h.svc.GetUserRoles(c.Request(), username)
	if err != nil {
		logger.Errorf("failed to get user roles: %v", err)
		resp := response.Err(errors.ErrInternal.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(roles)
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}
