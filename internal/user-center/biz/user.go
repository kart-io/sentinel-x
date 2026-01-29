package biz

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	storepkg "github.com/kart-io/sentinel-x/pkg/store"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// UserService 处理用户业务逻辑
type UserService struct {
	userStore *store.UserStore
}

// NewUserService 创建新的 UserService
func NewUserService(userStore *store.UserStore) *UserService {
	return &UserService{userStore: userStore}
}

// Create 创建新用户并加密密码
func (s *UserService) Create(ctx context.Context, user *model.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.ErrInternal.WithCause(err)
	}
	user.Password = string(hashedPassword)
	return s.userStore.Create(ctx, user)
}

// Update 更新现有用户
func (s *UserService) Update(ctx context.Context, user *model.User) error {
	return s.userStore.Update(ctx, user)
}

// Delete 删除用户
func (s *UserService) Delete(ctx context.Context, username string) error {
	return s.userStore.Delete(ctx, username)
}

// Get 检索用户
func (s *UserService) Get(ctx context.Context, username string) (*model.User, error) {
	return s.userStore.Get(ctx, username)
}

// GetByUserID retrieves a user by ID.
func (s *UserService) GetByUserID(ctx context.Context, userID string) (*model.User, error) {
	return s.userStore.GetByUserID(ctx, userID)
}

// List 列出用户
func (s *UserService) List(ctx context.Context, opts ...storepkg.Option) (int64, []*model.User, error) {
	return s.userStore.List(ctx, opts...)
}

// ChangePassword 更改用户密码
func (s *UserService) ChangePassword(ctx context.Context, username, newPassword string) error {
	user, err := s.userStore.Get(ctx, username)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.ErrInternal.WithCause(err)
	}

	user.Password = string(hashedPassword)
	return s.userStore.Update(ctx, user)
}

// ValidatePassword 验证提供的密码是否与存储的哈希匹配
func (s *UserService) ValidatePassword(ctx context.Context, username, password string) error {
	user, err := s.userStore.Get(ctx, username)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return errors.ErrUnauthorized.WithMessage("密码不正确")
	}
	return nil
}
