package userservice

import (
	"context"
	"errors"

	"github.com/kart-io/sentinel-x/example/server/user-center/model"
	"github.com/kart-io/sentinel-x/pkg/auth/jwt"
)

type Service struct {
	jwtAuth *jwt.JWT
}

func NewService(jwtAuth *jwt.JWT) *Service {
	return &Service{
		jwtAuth: jwtAuth,
	}
}

func (s *Service) ServiceName() string {
	return "user-center"
}

func (s *Service) Login(ctx context.Context, username, password string) (string, *model.User, error) {
	user, ok := model.Users[username]
	if !ok || user.Password != password {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := s.jwtAuth.Sign(ctx, user.ID)
	if err != nil {
		return "", nil, err
	}

	return token.GetAccessToken(), user, nil
}

func (s *Service) GetUser(ctx context.Context, id string) (*model.User, error) {
	for _, u := range model.Users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}
