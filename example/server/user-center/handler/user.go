package handler

import (
	"context"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/kart-io/sentinel-x/example/server/user-center/api/v1"
	"github.com/kart-io/sentinel-x/example/server/user-center/service/userservice"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

type UserHandler struct {
	v1.UnimplementedUserServiceServer
	svc *userservice.Service
}

func NewUserHandler(svc *userservice.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

// HTTP Handler
func (h *UserHandler) GetProfile(c transport.Context) {
	userID := c.Param("id")

	user, err := h.svc.GetUser(c.Request(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) AdminAction(c transport.Context) {
	c.JSON(http.StatusOK, map[string]string{"message": "admin action allowed"})
}

// gRPC Handler
func (h *UserHandler) GetUser(ctx context.Context, req *v1.UserRequest) (*v1.UserResponse, error) {
	user, err := h.svc.GetUser(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &v1.UserResponse{
		Id:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}
