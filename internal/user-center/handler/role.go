package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/pkg/httputils"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	pkgstore "github.com/kart-io/sentinel-x/pkg/store"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
	"github.com/kart-io/sentinel-x/pkg/utils/validator"
)

// AssignPermissionRequest represents the request to assign a permission to a role.
type AssignPermissionRequest struct {
	RoleCode string `json:"role_code" binding:"required"`
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

// RoleHandler handles role-related HTTP requests and gRPC requests.
type RoleHandler struct {
	v1.UnimplementedRoleServiceServer
	svc *biz.RoleService
}

// NewRoleHandler creates a new RoleHandler.
func NewRoleHandler(svc *biz.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// Create godoc
//
//	@Summary		创建角色
//	@Description	创建新角色
//	@Tags			Roles
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		v1.CreateRoleRequest	true	"创建角色请求"
//	@Success		200		{object}	response.Response		"成功响应"
//	@Failure		400		{object}	response.Response		"请求错误"
//	@Router			/v1/roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req v1.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	role := &model.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Status:      1, // Default enabled
	}

	if err := h.svc.Create(c.Request.Context(), role); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, role)
}

// Update godoc
//
//	@Summary		更新角色
//	@Description	更新角色信息
//	@Tags			Roles
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		v1.UpdateRoleRequest	true	"更新角色请求"
//	@Success		200		{object}	response.Response		"成功响应"
//	@Failure		400		{object}	response.Response		"请求错误"
//	@Failure		404		{object}	response.Response		"角色不存在"
//	@Router			/v1/roles [put]
func (h *RoleHandler) Update(c *gin.Context) {
	var req v1.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	role, err := h.svc.Get(c.Request.Context(), req.Code)
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	// 只有字段不为 nil 时才更新（nil = 不更新,非 nil = 更新包括空值)
	if req.Name != nil {
		role.Name = req.Name.Value
	}
	if req.Description != nil {
		role.Description = req.Description.Value
	}
	if req.Status != nil {
		role.Status = int(req.Status.Value)
	}

	if err := h.svc.Update(c.Request.Context(), role); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, role)
}

// Delete godoc
//
//	@Summary		删除角色
//	@Description	删除指定角色
//	@Tags			Roles
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			code	query		string				true	"角色代码"
//	@Success		200		{object}	response.Response	"成功响应"
//	@Failure		404		{object}	response.Response	"角色不存在"
//	@Router			/v1/roles [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	var req v1.DeleteRoleRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), req.Code); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, "role deleted")
}

// Get godoc
//
//	@Summary		获取角色
//	@Description	获取指定角色信息
//	@Tags			Roles
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			code	query		string				true	"角色代码"
//	@Success		200		{object}	response.Response	"成功响应"
//	@Failure		404		{object}	response.Response	"角色不存在"
//	@Router			/v1/roles/detail [get]
func (h *RoleHandler) Get(c *gin.Context) {
	var req v1.GetRoleRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	role, err := h.svc.Get(c.Request.Context(), req.Code)
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, role)
}

// List godoc
//
//	@Summary		角色列表
//	@Description	获取角色列表（分页）
//	@Tags			Roles
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			page		query		int					false	"页码"	default(1)
//	@Param			page_size	query		int					false	"每页数量"	default(10)
//	@Success		200			{object}	response.Response	"成功响应"
//	@Router			/v1/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	var req v1.ListRolesRequest
	// Ignore bind error for optional params
	_ = c.ShouldBindQuery(&req)
	_ = validator.Global().Validate(&req)

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	count, roles, err := h.svc.List(c.Request.Context(), pkgstore.WithPage(page, pageSize))
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, response.Page(roles, count, page, pageSize))
}

// AssignUserRole godoc
//
//	@Summary		分配角色
//	@Description	为用户分配角色
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		v1.AssignRoleRequest	true	"分配角色请求"
//	@Success		200			{object}	response.Response		"成功响应"
//	@Failure		400			{object}	response.Response		"请求错误"
//	@Router			/v1/users/roles [post]
func (h *RoleHandler) AssignUserRole(c *gin.Context) {
	var req v1.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.AssignRoleToUser(c.Request.Context(), req.Username, req.RoleCode); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, "role assigned")
}

// ListUserRoles godoc
//
//	@Summary		获取用户角色
//	@Description	获取用户的所有角色
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			username	query		string				true	"用户名"
//	@Success		200			{object}	response.Response	"成功响应"
//	@Failure		404			{object}	response.Response	"用户不存在"
//	@Router			/v1/users/roles [get]
func (h *RoleHandler) ListUserRoles(c *gin.Context) {
	var req v1.GetUserRolesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	roles, err := h.svc.GetUserRoles(c.Request.Context(), req.Username)
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, roles)
}

// AssignPermission godoc
//
//	@Summary		分配权限
//	@Description	为角色分配权限
//	@Tags			Roles
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		AssignPermissionRequest	true	"分配权限请求"
//	@Success		200		{object}	response.Response		"成功响应"
//	@Failure		400		{object}	response.Response		"请求错误"
//	@Router			/v1/roles/permissions [post]
func (h *RoleHandler) AssignPermission(c *gin.Context) {
	var req AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.AssignPermission(c.Request.Context(), req.RoleCode, req.Resource, req.Action); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, "permission assigned")
}

// RemovePermission godoc
//
//	@Summary		移除权限
//	@Description	移除角色的权限
//	@Tags			Roles
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		AssignPermissionRequest	true	"移除权限请求"
//	@Success		200		{object}	response.Response		"成功响应"
//	@Failure		400		{object}	response.Response		"请求错误"
//	@Router			/v1/roles/permissions [delete]
func (h *RoleHandler) RemovePermission(c *gin.Context) {
	var req AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.RemovePermission(c.Request.Context(), req.RoleCode, req.Resource, req.Action); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, "permission removed")
}

// ================= gRPC Methods =================

// CreateRole creates a new role (gRPC).
func (h *RoleHandler) CreateRole(ctx context.Context, req *v1.CreateRoleRequest) (*v1.Role, error) {
	role := &model.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Status:      1,
	}

	if err := h.svc.Create(ctx, role); err != nil {
		return nil, err
	}

	return &v1.Role{
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		Status:      int32(role.Status), //nolint:gosec
	}, nil
}

// UpdateRole updates a role (gRPC).
func (h *RoleHandler) UpdateRole(ctx context.Context, req *v1.UpdateRoleRequest) (*v1.Role, error) {
	role, err := h.svc.Get(ctx, req.Code)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		role.Name = req.Name.Value
	}
	if req.Description != nil {
		role.Description = req.Description.Value
	}
	if req.Status != nil {
		role.Status = int(req.Status.Value)
	}

	if err := h.svc.Update(ctx, role); err != nil {
		return nil, err
	}

	return &v1.Role{
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		Status:      int32(role.Status), //nolint:gosec
	}, nil
}

// DeleteRole deletes a role (gRPC).
func (h *RoleHandler) DeleteRole(ctx context.Context, req *v1.DeleteRoleRequest) (*emptypb.Empty, error) {
	if err := h.svc.Delete(ctx, req.Code); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// GetRole retrieves a role (gRPC).
func (h *RoleHandler) GetRole(ctx context.Context, req *v1.GetRoleRequest) (*v1.Role, error) {
	role, err := h.svc.Get(ctx, req.Code)
	if err != nil {
		return nil, err
	}

	return &v1.Role{
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		Status:      int32(role.Status), //nolint:gosec
	}, nil
}

// ListRoles lists roles (gRPC).
func (h *RoleHandler) ListRoles(ctx context.Context, req *v1.ListRolesRequest) (*v1.ListRolesResponse, error) {
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	count, roles, err := h.svc.List(ctx, pkgstore.WithPage(page, pageSize))
	if err != nil {
		return nil, err
	}

	var items []*v1.Role
	for _, r := range roles {
		items = append(items, &v1.Role{
			Code:        r.Code,
			Name:        r.Name,
			Description: r.Description,
			Status:      int32(r.Status), //nolint:gosec // Status is validated strictly
		})
	}

	return &v1.ListRolesResponse{
		Total: count,
		Items: items,
	}, nil
}

// AssignRole assigns a role to a user (gRPC).
func (h *RoleHandler) AssignRole(ctx context.Context, req *v1.AssignRoleRequest) (*emptypb.Empty, error) {
	if err := h.svc.AssignRoleToUser(ctx, req.Username, req.RoleCode); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// GetUserRoles gets roles for a user (gRPC).
func (h *RoleHandler) GetUserRoles(ctx context.Context, req *v1.GetUserRolesRequest) (*v1.GetUserRolesResponse, error) {
	roles, err := h.svc.GetUserRoles(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	var items []*v1.Role
	for _, r := range roles {
		items = append(items, &v1.Role{
			Code:        r.Code,
			Name:        r.Name,
			Description: r.Description,
			Status:      int32(r.Status), //nolint:gosec // Status is validated strictly
		})
	}

	return &v1.GetUserRolesResponse{
		Items: items,
	}, nil
}
