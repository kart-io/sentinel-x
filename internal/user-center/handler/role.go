package handler

import (
	"strconv"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/pkg/utils"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	pkgstore "github.com/kart-io/sentinel-x/pkg/store"
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
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	role := &model.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Status:      1, // Default enabled
	}

	if err := h.svc.Create(c.Request(), role); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, role)
}

// Update handles role updates.
func (h *RoleHandler) Update(c transport.Context) {
	code := c.Param("code")
	if code == "" {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage("role code is required"), nil)
		return
	}

	// Fetch existing role first
	role, err := h.svc.Get(c.Request(), code)
	if err != nil {
		utils.WriteResponse(c, err, nil)
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
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
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
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, role)
}

// Delete handles role deletion.
func (h *RoleHandler) Delete(c transport.Context) {
	code := c.Param("code")
	if code == "" {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage("role code is required"), nil)
		return
	}

	if err := h.svc.Delete(c.Request(), code); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, "role deleted")
}

// Get handles retrieving a role.
func (h *RoleHandler) Get(c transport.Context) {
	code := c.Param("code")
	if code == "" {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage("role code is required"), nil)
		return
	}

	role, err := h.svc.Get(c.Request(), code)
	if err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, role)
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

	// offset := (req.Page - 1) * req.PageSize
	// limit := req.PageSize

	count, roles, err := h.svc.List(c.Request(), pkgstore.WithPage(req.Page, req.PageSize))
	if err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, response.Page(roles, count, req.Page, req.PageSize))
}

// AssignRole handles assigning a role to a user.
func (h *RoleHandler) AssignRole(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage("username is required"), nil)
		return
	}

	// AssignRoleRequest is the request body for assigning a role.
	type AssignRoleRequest struct {
		RoleCode string `json:"role_code" validate:"required"`
	}

	var req AssignRoleRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.AssignRoleToUser(c.Request(), username, req.RoleCode); err != nil {
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, "role assigned")
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
		utils.WriteResponse(c, err, nil)
		return
	}

	utils.WriteResponse(c, nil, roles)
}
