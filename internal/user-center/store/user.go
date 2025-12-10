package store

import (
	"context"

	"gorm.io/gorm"

	"github.com/kart-io/sentinel-x/internal/model"
)

type users struct {
	db *gorm.DB
}

func newUsers(db *gorm.DB) *users {
	return &users{db}
}

// Create creates a new user.
func (u *users) Create(ctx context.Context, user *model.User) error {
	return u.db.WithContext(ctx).Create(user).Error
}

// Update updates an existing user.
func (u *users) Update(ctx context.Context, user *model.User) error {
	return u.db.WithContext(ctx).Save(user).Error
}

// Delete deletes a user by username.
func (u *users) Delete(ctx context.Context, username string) error {
	return u.db.WithContext(ctx).Where("username = ?", username).Delete(&model.User{}).Error
}

// Get retrieves a user by username.
func (u *users) Get(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := u.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// List lists users with pagination.
func (u *users) List(ctx context.Context, offset, limit int) (int64, []*model.User, error) {
	var count int64
	var users []*model.User

	if err := u.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}

	if err := u.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return 0, nil, err
	}

	return count, users, nil
}
