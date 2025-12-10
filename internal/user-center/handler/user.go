package handler

import (
	"net/http"
	"strconv"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	svc *biz.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *biz.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Create handles user creation.
func (h *UserHandler) Create(c transport.Context) {
	var req struct {
		model.User
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user := req.User
	user.Password = req.Password

	if err := h.svc.Create(c.Request(), &user); err != nil {
		logger.Errorf("failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// Update handles user updates.
func (h *UserHandler) Update(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}

	// Existing user passed in specific fields via JSON?
	// Bind to struct with pointers to tell difference between missing and empty?
	// Or use anonymous struct again.
	// But to preserve password, we must fetch -> copy.

	// Fetch existing user to ensure we don't overwrite password with empty string
	existingUser, err := h.svc.Get(c.Request(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	// Bind new values directly to existing user struct
	if err := c.Bind(existingUser); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	existingUser.Username = username // Ensure username from path is respected

	if err := h.svc.Update(c.Request(), existingUser); err != nil {
		logger.Errorf("failed to update user: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existingUser)
}

// Delete handles user deletion.
func (h *UserHandler) Delete(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}

	if err := h.svc.Delete(c.Request(), username); err != nil {
		logger.Errorf("failed to delete user: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"message": "user deleted"})
}

// Get handles retrieving a single user.
func (h *UserHandler) Get(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}

	user, err := h.svc.Get(c.Request(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// List handles listing users.
func (h *UserHandler) List(c transport.Context) {
	offsetStr := c.Query("offset")
	limitStr := c.Query("limit")

	offset := 0
	limit := 10

	if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
		offset = val
	}
	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		limit = val
	}

	count, users, err := h.svc.List(c.Request(), offset, limit)
	if err != nil {
		logger.Errorf("failed to list users: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.UserList{
		TotalCount: count,
		Items:      users,
	})
}

// ChangePassword handles password change.
func (h *UserHandler) ChangePassword(c transport.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}

	var req struct {
		NewPassword string `json:"newPassword"`
	}
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.svc.ChangePassword(c.Request(), username, req.NewPassword); err != nil {
		logger.Errorf("failed to change password: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"message": "password changed"})
}
