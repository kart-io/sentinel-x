package handler

import (
	"net/http"
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
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
	var req model.LoginRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp, err := h.svc.Login(c.Request(), &req)
	if err != nil {
		logger.Warnf("Login failed: %v", err)
		c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
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
		c.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
		return
	}

	if err := h.svc.Logout(c.Request(), token); err != nil {
		logger.Errorf("Logout failed: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to logout"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}
