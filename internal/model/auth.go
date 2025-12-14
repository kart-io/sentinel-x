// Package model defines the data models for the application.
package model

// LoginRequest represents the login request body.
type LoginRequest struct {
	Username string `json:"username" form:"username" validate:"required"`
	Password string `json:"password" form:"password" validate:"required"`
}

// LoginResponse represents the login response body.
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
	UserID    uint64 `json:"user_id"`
}

// RegisterRequest represents the register request body.
type RegisterRequest struct {
	Username string `json:"username" form:"username" validate:"required"`
	Password string `json:"password" form:"password" validate:"required"`
	Email    string `json:"email" form:"email" validate:"required,email"`
	Mobile   string `json:"mobile" form:"mobile" validate:"omitempty,mobile"`
}
