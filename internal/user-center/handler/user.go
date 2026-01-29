package handler

import (
	"context"
	"strings"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/pkg/httputils"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/store"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
	"github.com/kart-io/sentinel-x/pkg/utils/validator"
)

// UserHandler handles user-related HTTP requests and gRPC requests.
type UserHandler struct {
	v1.UnimplementedUserServiceServer
	svc     *biz.UserService
	roleSvc *biz.RoleService
	authSvc *biz.AuthService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *biz.UserService, roleSvc *biz.RoleService, authSvc *biz.AuthService) *UserHandler {
	return &UserHandler{
		svc:     svc,
		roleSvc: roleSvc,
		authSvc: authSvc,
	}
}

// Create godoc
//
//	@Summary		创建用户
//	@Description	创建新用户（公开接口，用于注册）
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		v1.CreateUserRequest	true	"创建用户请求"
//	@Success		200		{object}	response.Response		"成功响应"
//	@Failure		400		{object}	response.Response		"请求错误"
//	@Router			/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req v1.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	user := &model.User{
		Username: req.Username,
		Password: req.Password,
		Email:    &req.Email,
		Mobile:   req.Mobile,
	}

	if err := h.svc.Create(c.Request.Context(), user); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, user)
}

// Update godoc
//
//	@Summary		更新用户
//	@Description	更新用户信息
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		v1.UpdateUserRequest	true	"更新用户请求"
//	@Success		200			{object}	response.Response		"成功响应"
//	@Failure		400			{object}	response.Response		"请求错误"
//	@Failure		404			{object}	response.Response		"用户不存在"
//	@Router			/v1/users [put]
func (h *UserHandler) Update(c *gin.Context) {
	// All parameters from request body (JSON)
	var req v1.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	user, err := h.svc.Get(c.Request.Context(), req.Username)
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	// 只有字段不为 nil 时才更新（nil = 不更新,非 nil = 更新包括空值)
	if req.Email != nil {
		user.Email = &req.Email.Value
	}
	if req.Mobile != nil {
		user.Mobile = req.Mobile.Value
	}

	if err := h.svc.Update(c.Request.Context(), user); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, user)
}

// Delete godoc
//
//	@Summary		删除用户
//	@Description	删除指定用户
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			username	query		string				true	"用户名"
//	@Success		200			{object}	response.Response	"成功响应"
//	@Failure		404			{object}	response.Response	"用户不存在"
//	@Router			/v1/users [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	var req v1.DeleteUserRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), req.Username); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, "user deleted")
}

// BatchDelete godoc
//
//	@Summary		批量删除用户
//	@Description	批量删除多个用户
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		v1.BatchDeleteRequest	true	"批量删除请求"
//	@Success		200		{object}	response.Response		"成功响应"
//	@Failure		400		{object}	response.Response		"请求错误"
//	@Router			/v1/users/batch-delete [post]
func (h *UserHandler) BatchDelete(c *gin.Context) {
	var req v1.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		resp := response.Err(errors.ErrValidationFailed.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	for _, username := range req.Usernames {
		if err := h.svc.Delete(c.Request.Context(), username); err != nil {
			httputils.WriteResponse(c, err, nil)
			return
		}
	}

	httputils.WriteResponse(c, nil, "users deleted")
}

// Get godoc
//
//	@Summary		获取用户
//	@Description	获取指定用户信息
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			username	query		string				true	"用户名"
//	@Success		200			{object}	response.Response	"成功响应"
//	@Failure		404			{object}	response.Response	"用户不存在"
//	@Router			/v1/users/detail [get]
func (h *UserHandler) Get(c *gin.Context) {
	var req v1.GetUserRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	user, err := h.svc.Get(c.Request.Context(), req.Username)
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, user)
}

// List godoc
//
//	@Summary		用户列表
//	@Description	获取用户列表（分页）
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			page		query		int					false	"页码"	default(1)
//	@Param			page_size	query		int					false	"每页数量"	default(10)
//	@Success		200			{object}	response.Response	"成功响应"
//	@Router			/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	var req v1.ListUsersRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	// Set defaults if zero
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	count, users, err := h.svc.List(c.Request.Context(), store.WithPage(page, pageSize))
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, response.Page(users, count, page, pageSize))
}

// GetProfile godoc
//
//	@Summary		获取当前用户信息
//	@Description	获取当前登录用户的详细信息
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	response.Response	"成功响应"
//	@Failure		401	{object}	response.Response	"未授权"
//	@Router			/auth/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	username := auth.SubjectFromContext(c.Request.Context())
	if username == "" {
		resp := response.Err(errors.ErrUnauthorized)
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	user, err := h.svc.Get(c.Request.Context(), username)
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, user)
}

// UpdatePassword godoc
//
//	@Summary		修改密码
//	@Description	修改用户密码（需要提供旧密码）
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		v1.ChangePasswordRequest	true	"修改密码请求"
//	@Success		200			{object}	response.Response			"成功响应"
//	@Failure		400			{object}	response.Response			"请求错误"
//	@Router			/v1/users/password [post]
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	// All parameters from request body (JSON) - secure
	var req v1.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}
	// Manual cross-field validation not supported by base proto-validate
	if req.NewPassword != req.ConfirmPassword {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage("passwords do not match"), nil)
		return
	}

	// Verify old password
	if err := h.svc.ValidatePassword(c.Request.Context(), req.Username, req.OldPassword); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), req.Username, req.NewPassword); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, "password changed")
}

// ================= gRPC Methods =================

// GetUser implements the gRPC method to get a user by ID.
func (h *UserHandler) GetUser(ctx context.Context, req *v1.UserRequest) (*v1.UserResponse, error) {
	var user *model.User
	user, err := h.svc.GetByUserID(ctx, req.Id)
	if err != nil {
		user, err = h.svc.Get(ctx, req.Id)
	}

	if err != nil {
		return nil, err
	}

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
		Id:       user.ID,
		Username: user.Username,
		Role:     roleStr,
	}, nil
}

// CreateUser implements the gRPC method to create a user.
func (h *UserHandler) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.UserResponse, error) {
	user := &model.User{
		Username: req.Username,
		Password: req.Password,
		Email:    &req.Email,
		Mobile:   req.Mobile,
	}

	if err := h.svc.Create(ctx, user); err != nil {
		return nil, err
	}

	var email string
	if user.Email != nil {
		email = *user.Email
	}

	return &v1.UserResponse{
		Id:       user.ID,
		Username: user.Username,
		Email:    email,
		Mobile:   user.Mobile,
	}, nil
}

// UpdateUser implements the gRPC method to update a user.
func (h *UserHandler) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.UserResponse, error) {
	user, err := h.svc.Get(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	if req.Email != nil {
		user.Email = &req.Email.Value
	}
	if req.Mobile != nil {
		user.Mobile = req.Mobile.Value
	}

	if err := h.svc.Update(ctx, user); err != nil {
		return nil, err
	}

	var email string
	if user.Email != nil {
		email = *user.Email
	}

	return &v1.UserResponse{
		Id:       user.ID,
		Username: user.Username,
		Email:    email,
		Mobile:   user.Mobile,
	}, nil
}

// DeleteUser implements the gRPC method to delete a user.
func (h *UserHandler) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*emptypb.Empty, error) {
	if err := h.svc.Delete(ctx, req.Username); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ListUsers implements the gRPC method to list users.
func (h *UserHandler) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	total, users, err := h.svc.List(ctx, store.WithPage(page, pageSize))
	if err != nil {
		return nil, err
	}

	var userResponses []*v1.UserResponse
	for _, user := range users {
		var email string
		if user.Email != nil {
			email = *user.Email
		}
		userResponses = append(userResponses, &v1.UserResponse{
			Id:       user.ID,
			Username: user.Username,
			Email:    email,
			Mobile:   user.Mobile,
		})
	}

	return &v1.ListUsersResponse{
		Total: total,
		Items: userResponses,
	}, nil
}

// BatchDeleteUsers implements the gRPC method to batch delete users.
func (h *UserHandler) BatchDeleteUsers(ctx context.Context, req *v1.BatchDeleteRequest) (*emptypb.Empty, error) {
	for _, username := range req.Usernames {
		if err := h.svc.Delete(ctx, username); err != nil {
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}

// ChangePassword implements the gRPC method to change password.
func (h *UserHandler) ChangePassword(ctx context.Context, req *v1.ChangePasswordRequest) (*emptypb.Empty, error) {
	if req.NewPassword != req.ConfirmPassword {
		return nil, errors.ErrBadRequest.WithMessage("passwords do not match")
	}

	username := auth.SubjectFromContext(ctx)
	if username == "" {
		return nil, errors.ErrUnauthorized
	}

	if err := h.svc.ValidatePassword(ctx, username, req.OldPassword); err != nil {
		return nil, err
	}

	if err := h.svc.ChangePassword(ctx, username, req.NewPassword); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Login implements the gRPC method to login.
func (h *UserHandler) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	respData, err := h.authSvc.Login(ctx, &model.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	}, "", "")
	if err != nil {
		return nil, err
	}
	return &v1.LoginResponse{
		Token:    respData.Token,
		ExpireAt: respData.ExpiresIn,
	}, nil
}

// Register implements the gRPC method to register.
func (h *UserHandler) Register(ctx context.Context, req *v1.RegisterRequest) (*emptypb.Empty, error) {
	if err := h.authSvc.Register(ctx, &model.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Mobile:   req.Mobile,
	}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Logout implements the gRPC method to logout.
func (h *UserHandler) Logout(ctx context.Context, req *v1.LogoutRequest) (*emptypb.Empty, error) {
	if err := h.authSvc.Logout(ctx, req.Token); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
