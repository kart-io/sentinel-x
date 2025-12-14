package handler

import (
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/pkg/httputils"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	svc *biz.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *biz.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login handles user login.
func (h *AuthHandler) Login(c transport.Context) {
	var req v1.LoginRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	respData, err := h.svc.Login(c.Request(), &model.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		logger.Warnf("Login failed: %v", err)
		httputils.WriteResponse(c, errors.ErrUnauthorized.WithMessage(err.Error()), nil)
		return
	}

	httputils.WriteResponse(c, nil, respData)
}

// Logout handles user logout.
func (h *AuthHandler) Logout(c transport.Context) {
	token := c.Header("Authorization")
	if len(token) > 7 && strings.ToUpper(token[:7]) == "BEARER " {
		token = token[7:]
	}

	if msg := c.Query("token"); msg != "" && token == "" {
		token = msg
	}

	if token == "" {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage("token required"), nil)
		return
	}

	if err := h.svc.Logout(c.Request(), token); err != nil {
		logger.Errorf("Logout failed: %v", err)
		httputils.WriteResponse(c, errors.ErrInternal.WithMessage("failed to logout"), nil)
		return
	}

	httputils.WriteResponse(c, nil, "logged out")
}

// Register handles user registration.
func (h *AuthHandler) Register(c transport.Context) {
	var req v1.RegisterRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.Register(c.Request(), &model.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Mobile:   req.Mobile,
	}); err != nil {
		logger.Errorf("Register failed: %v", err)
		httputils.WriteResponse(c, errors.ErrInternal.WithMessage(err.Error()), nil)
		return
	}

	httputils.WriteResponse(c, nil, "user registered")
}
