package biz

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
)

// UserService handles user business logic.
type UserService struct {
	store store.Factory
}

// NewUserService creates a new UserService.
func NewUserService(store store.Factory) *UserService {
	return &UserService{store: store}
}

// Create creates a new user with encrypted password.
func (s *UserService) Create(ctx context.Context, user *model.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	return s.store.Users().Create(ctx, user)
}

// Update updates an existing user.
func (s *UserService) Update(ctx context.Context, user *model.User) error {
	return s.store.Users().Update(ctx, user)
}

// Delete deletes a user.
func (s *UserService) Delete(ctx context.Context, username string) error {
	return s.store.Users().Delete(ctx, username)
}

// Get retrieves a user.
func (s *UserService) Get(ctx context.Context, username string) (*model.User, error) {
	return s.store.Users().Get(ctx, username)
}

// List lists users.
func (s *UserService) List(ctx context.Context, offset, limit int) (int64, []*model.User, error) {
	return s.store.Users().List(ctx, offset, limit)
}

// ChangePassword changes a user's password.
func (s *UserService) ChangePassword(ctx context.Context, username, newPassword string) error {
	user, err := s.store.Users().Get(ctx, username)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	return s.store.Users().Update(ctx, user)
}

// ValidatePassword checks if the provided password matches the stored hash.
func (s *UserService) ValidatePassword(ctx context.Context, username, password string) error {
	user, err := s.store.Users().Get(ctx, username)
	if err != nil {
		return err
	}

	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}
