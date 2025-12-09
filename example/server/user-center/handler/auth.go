package handler

import (
	"net/http"

	"github.com/kart-io/sentinel-x/example/server/user-center/service/userservice"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

type AuthHandler struct {
	svc *userservice.Service
}

func NewAuthHandler(svc *userservice.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c transport.Context) {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	token, user, err := h.svc.Login(c.Request(), req.Username, req.Password)
	if err != nil {
		if err.Error() == "invalid credentials" {
			c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}
