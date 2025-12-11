package biz

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
)

// AuthService handles authentication business logic.
type AuthService struct {
	jwtAuth *jwt.JWT
	store   store.Factory
}

// NewAuthService creates a new AuthService.
func NewAuthService(jwtAuth *jwt.JWT, store store.Factory) *AuthService {
	return &AuthService{
		jwtAuth: jwtAuth,
		store:   store,
	}
}

// Login authenticates a user and returns a token.
func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	// TODO: Validate user against DB
	user, err := s.store.Users().Get(ctx, req.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate token
	token, err := s.jwtAuth.Sign(ctx, req.Username, auth.WithExtra(map[string]interface{}{
		"id": user.ID,
	}))
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		Token:     token.GetAccessToken(),
		ExpiresIn: token.GetExpiresAt(),
		UserID:    user.ID,
	}, nil
}

// Logout revokes a user token.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.jwtAuth.Revoke(ctx, token)
}

// Register registers a new user.
func (s *AuthService) Register(ctx context.Context, req *model.RegisterRequest) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    &req.Email,
		Status:   1,
	}

	return s.store.Users().Create(ctx, user)
}
