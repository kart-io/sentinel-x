package handler

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/pkg/httputils"
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
	authSvc *biz.AuthService
}

// NewUserHandler creates a new UserHandler.
// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *biz.UserService, roleSvc *biz.RoleService, authSvc *biz.AuthService) *UserHandler {
	return &UserHandler{
		svc:     svc,
		roleSvc: roleSvc,
		authSvc: authSvc,
	}
}

// Create handles user creation.
func (h *UserHandler) Create(c transport.Context) {
	var req v1.CreateUserRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	user := &model.User{
		Username: req.Username,
		Password: req.Password,
		Email:    &req.Email,
		Mobile:   req.Mobile,
	}

	if err := h.svc.Create(c.Request(), user); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, user)
}

// Update handles user updates.
func (h *UserHandler) Update(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage("username is required"), nil)
		return
	}

	user, err := h.svc.Get(c.Request(), username)
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	var req v1.UpdateUserRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	if req.Email != "" {
		user.Email = &req.Email
	}
	if req.Mobile != "" {
		user.Mobile = req.Mobile
	}

	if err := h.svc.Update(c.Request(), user); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, user)
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
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, "user deleted")
}

// BatchDelete handles batch deletion of users.
func (h *UserHandler) BatchDelete(c transport.Context) {
	var req v1.BatchDeleteRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	for _, username := range req.Usernames {
		if err := h.svc.Delete(c.Request(), username); err != nil {
			httputils.WriteResponse(c, err, nil)
			return
		}
	}

	httputils.WriteResponse(c, nil, "users deleted")
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
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, user)
}

// List handles listing users.
func (h *UserHandler) List(c transport.Context) {
	var req v1.ListUsersRequest

	// Ignore bind error for optional params
	_ = c.ShouldBindAndValidate(&req)

	// Set defaults if zero
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	count, users, err := h.svc.List(c.Request(), store.WithPage(page, pageSize))
	if err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, response.Page(users, count, page, pageSize))
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
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, user)
}

// UpdatePassword handles password change.
func (h *UserHandler) UpdatePassword(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		resp := response.Err(errors.ErrBadRequest.WithMessage("username is required"))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	var req v1.ChangePasswordRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		resp := response.Err(errors.ErrBadRequest.WithMessage(err.Error()))
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// Manual cross-field validation not supported by base proto-validate
	if req.NewPassword != req.ConfirmPassword {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage("passwords do not match"), nil)
		return
	}

	// Verify old password
	if err := h.svc.ValidatePassword(c.Request(), username, req.OldPassword); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	if err := h.svc.ChangePassword(c.Request(), username, req.NewPassword); err != nil {
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, "password changed")
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
		user, err = h.svc.GetByUserID(ctx, id)
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
		Id:       strconv.FormatUint(user.ID, 10),
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

	if req.Email != "" {
		user.Email = &req.Email
	}
	if req.Mobile != "" {
		user.Mobile = req.Mobile
	}

	if err := h.svc.Update(ctx, user); err != nil {
		return nil, err
	}

	var email string
	if user.Email != nil {
		email = *user.Email
	}

	return &v1.UserResponse{
		Id:       strconv.FormatUint(user.ID, 10),
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
			Id:       strconv.FormatUint(user.ID, 10),
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
	})
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
