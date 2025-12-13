package handler

import (
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/pkg/utils"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
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

// LoginRequest is the request body for user login.
type LoginRequest struct {
	// Username must be provided, 3-32 characters
	Username string `json:"username" form:"username" validate:"required,min=3,max=32"`
	// Password must be provided, 6-64 characters
	Password string `json:"password" form:"password" validate:"required,min=6,max=64"`
}

// Login handles user login.
func (h *AuthHandler) Login(c transport.Context) {
	var req LoginRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	respData, err := h.svc.Login(c.Request(), &model.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		logger.Warnf("Login failed: %v", err)
		utils.WriteResponse(c, errors.ErrUnauthorized.WithMessage(err.Error()), nil)
		return
	}

	utils.WriteResponse(c, nil, respData)
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
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage("token required"), nil)
		return
	}

	if err := h.svc.Logout(c.Request(), token); err != nil {
		logger.Errorf("Logout failed: %v", err)
		utils.WriteResponse(c, errors.ErrInternal.WithMessage("failed to logout"), nil)
		return
	}

	utils.WriteResponse(c, nil, "logged out")
}

// RegisterRequest is the request body for user registration.
type RegisterRequest struct {
	// Username must start with letter, contain letters/numbers/underscore, 3-32 chars
	Username string `json:"username" form:"username" validate:"required,username"`
	// Password must be at least 8 chars with letter and number
	Password string `json:"password" form:"password" validate:"required,password"`
	// Email must be valid email format
	Email string `json:"email" form:"email" validate:"required,email"`
	// Mobile must be valid mobile number (optional)
	Mobile string `json:"mobile" form:"mobile" validate:"omitempty,mobile"`
}

// Register handles user registration.
func (h *AuthHandler) Register(c transport.Context) {
	var req RegisterRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		utils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.Register(c.Request(), &model.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Mobile:   req.Mobile,
	}); err != nil {
		logger.Errorf("Register failed: %v", err)
		utils.WriteResponse(c, errors.ErrInternal.WithMessage(err.Error()), nil)
		return
	}

	utils.WriteResponse(c, nil, "user registered")
}
