package store

import (
	"context"
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/pkg/store"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

type roles struct {
	db *gorm.DB
}

func newRoles(db *gorm.DB) *roles {
	return &roles{db}
}

// Create creates a new role.
func (r *roles) Create(ctx context.Context, role *model.Role) error {
	if err := r.db.WithContext(ctx).Create(role).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.ErrAlreadyExists.WithMessage("role code already exists")
		}
		return errors.ErrDatabase.WithCause(err)
	}
	return nil
}

// Update updates an existing role.
func (r *roles) Update(ctx context.Context, role *model.Role) error {
	result := r.db.WithContext(ctx).Save(role)
	if result.Error != nil {
		return errors.ErrDatabase.WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithMessage("role not found")
	}
	return nil
}

// Delete deletes a role by code.
func (r *roles) Delete(ctx context.Context, code string) error {
	result := r.db.WithContext(ctx).Where("code = ?", code).Delete(&model.Role{})
	if result.Error != nil {
		return errors.ErrDatabase.WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithMessage("role not found")
	}
	return nil
}

// Get retrieves a role by code.
func (r *roles) Get(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrNotFound.WithMessage("role not found")
		}
		return nil, errors.ErrDatabase.WithCause(err)
	}
	return &role, nil
}

// List lists roles with pagination.
func (r *roles) List(ctx context.Context, opts ...store.Option) (int64, []*model.Role, error) {
	var count int64
	var roles []*model.Role

	db := store.NewWhere(opts...).Where(r.db.WithContext(ctx))

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
func (r *roles) AssignRole(ctx context.Context, userID, roleID uint64) error {
	userRole := &model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	if err := r.db.WithContext(ctx).Create(userRole).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.ErrAlreadyExists.WithMessage("user already has this role")
		}
		return errors.ErrDatabase.WithCause(err)
	}
	return nil
}

// RevokeRole revokes a role from a user.
func (r *roles) RevokeRole(ctx context.Context, userID, roleID uint64) error {
	result := r.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserRole{})
	if result.Error != nil {
		return errors.ErrDatabase.WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithMessage("role assignment not found")
	}
	return nil
}

// ListByUserID lists roles assigned to a user.
func (r *roles) ListByUserID(ctx context.Context, userID uint64) ([]*model.Role, error) {
	var roles []*model.Role
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return nil, errors.ErrDatabase.WithCause(err)
	}
	return roles, nil
}
