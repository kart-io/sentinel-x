package store

import (
	"context"
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/pkg/store"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// RoleStore 角色数据访问层
type RoleStore struct {
	db *gorm.DB
}

// NewRoleStore 创建角色存储实例
func NewRoleStore(db *gorm.DB) *RoleStore {
	return &RoleStore{db: db}
}

// Create creates a new role.
func (s *RoleStore) Create(ctx context.Context, role *model.Role) error {
	if err := s.db.WithContext(ctx).Create(role).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.ErrAlreadyExists.WithMessage("role code already exists")
		}
		return errors.ErrDatabase.WithCause(err)
	}
	return nil
}

// Update updates an existing role.
func (s *RoleStore) Update(ctx context.Context, role *model.Role) error {
	result := s.db.WithContext(ctx).Save(role)
	if result.Error != nil {
		return errors.ErrDatabase.WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithMessage("role not found")
	}
	return nil
}

// Delete deletes a role by code.
func (s *RoleStore) Delete(ctx context.Context, code string) error {
	result := s.db.WithContext(ctx).Where("code = ?", code).Delete(&model.Role{})
	if result.Error != nil {
		return errors.ErrDatabase.WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithMessage("role not found")
	}
	return nil
}

// Get retrieves a role by code.
func (s *RoleStore) Get(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&role).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrNotFound.WithMessage("role not found")
		}
		return nil, errors.ErrDatabase.WithCause(err)
	}
	return &role, nil
}

// List lists roles with pagination.
func (s *RoleStore) List(ctx context.Context, opts ...store.Option) (int64, []*model.Role, error) {
	var count int64
	var roles []*model.Role

	db := store.NewWhere(opts...).Where(s.db.WithContext(ctx))

	if err := db.Model(&model.Role{}).Count(&count).Error; err != nil {
		return 0, nil, errors.ErrDatabase.WithCause(err)
	}

	// store.Where applies offset and limit from opts
	if err := db.Find(&roles).Error; err != nil {
		return 0, nil, errors.ErrDatabase.WithCause(err)
	}

	return count, roles, nil
}

// AssignRole assigns a role to a user.
func (s *RoleStore) AssignRole(ctx context.Context, userID string, roleID uint64) error {
	userRole := &model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	if err := s.db.WithContext(ctx).Create(userRole).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.ErrAlreadyExists.WithMessage("user already has this role")
		}
		return errors.ErrDatabase.WithCause(err)
	}
	return nil
}

// RevokeRole revokes a role from a user.
func (s *RoleStore) RevokeRole(ctx context.Context, userID string, roleID uint64) error {
	result := s.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserRole{})
	if result.Error != nil {
		return errors.ErrDatabase.WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithMessage("role assignment not found")
	}
	return nil
}

// ListByUserID lists roles assigned to a user.
func (s *RoleStore) ListByUserID(ctx context.Context, userID string) ([]*model.Role, error) {
	var roles []*model.Role
	err := s.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return nil, errors.ErrDatabase.WithCause(err)
	}
	return roles, nil
}
