package handler

import (
	"github.com/kart-io/sentinel-x/example/server/user-center/service/userservice"
	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/response"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

type AuthHandler struct {
	svc *userservice.Service
}

func NewAuthHandler(svc *userservice.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// LoginRequest is the request body for user login.
type LoginRequest struct {
	// Username must be provided, 3-32 characters
	Username string `json:"username" validate:"required,min=3,max=32"`
	// Password must be provided, 6-64 characters
	Password string `json:"password" validate:"required,min=6,max=64"`
}

// Login handles POST /api/v1/auth/login
// This demonstrates login validation with username and password.
func (h *AuthHandler) Login(c transport.Context) {
	var req LoginRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	token, user, err := h.svc.Login(c.Request(), req.Username, req.Password)
	if err != nil {
		if err.Error() == "invalid credentials" {
			response.Fail(c, errors.ErrInvalidCredentials)
		} else {
			response.Fail(c, errors.ErrInternal.WithCause(err))
		}
		return
	}

	response.OK(c, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// RegisterRequest is the request body for user registration.
type RegisterRequest struct {
	// Username must start with letter, contain letters/numbers/underscore, 3-32 chars
	Username string `json:"username" validate:"required,username"`
	// Password must be at least 8 chars with letter and number
	Password string `json:"password" validate:"required,password"`
	// Email must be valid email format
	Email string `json:"email" validate:"required,email"`
	// Mobile must be valid mobile number (optional)
	Mobile string `json:"mobile" validate:"omitempty,mobile"`
}

// Register handles POST /api/v1/auth/register
// This demonstrates user registration with comprehensive validation.
func (h *AuthHandler) Register(c transport.Context) {
	var req RegisterRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	// In production, delegate to service layer for actual registration
	response.OK(c, map[string]interface{}{
		"message":  "User registered successfully (demo)",
		"username": req.Username,
		"email":    req.Email,
	})
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

// ChangePassword handles POST /api/v1/auth/change-password
// This demonstrates password change with validation.
func (h *AuthHandler) ChangePassword(c transport.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(c, err)
		return
	}

	// In production, delegate to service layer
	response.OK(c, map[string]string{
		"message": "Password changed successfully (demo)",
	})
}
