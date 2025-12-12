package biz

import (
	"context"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
)

// RoleService handles role-related business logic.
type RoleService struct {
	store store.Factory
}

// NewRoleService creates a new RoleService.
func NewRoleService(store store.Factory) *RoleService {
	return &RoleService{store: store}
}

// Create creates a new role.
func (s *RoleService) Create(ctx context.Context, role *model.Role) error {
	return s.store.Roles().Create(ctx, role)
}

// Update updates an existing role.
func (s *RoleService) Update(ctx context.Context, role *model.Role) error {
	return s.store.Roles().Update(ctx, role)
}

// Delete deletes a role by code.
func (s *RoleService) Delete(ctx context.Context, code string) error {
	return s.store.Roles().Delete(ctx, code)
}

// Get retrieves a role by code.
func (s *RoleService) Get(ctx context.Context, code string) (*model.Role, error) {
	return s.store.Roles().Get(ctx, code)
}

// List lists roles with pagination.
func (s *RoleService) List(ctx context.Context, offset, limit int) (int64, []*model.Role, error) {
	return s.store.Roles().List(ctx, offset, limit)
}

// AssignRoleToUser assigns a specific role to a user.
func (s *RoleService) AssignRoleToUser(ctx context.Context, username, roleCode string) error {
	user, err := s.store.Users().Get(ctx, username)
	if err != nil {
		return err
	}

	role, err := s.store.Roles().Get(ctx, roleCode)
	if err != nil {
		return err
	}

	return s.store.Roles().AssignRole(ctx, user.ID, role.ID)
}

// RevokeRoleFromUser revokes a role from a user.
func (s *RoleService) RevokeRoleFromUser(ctx context.Context, username, roleCode string) error {
	user, err := s.store.Users().Get(ctx, username)
	if err != nil {
		return err
	}

	role, err := s.store.Roles().Get(ctx, roleCode)
	if err != nil {
		return err
	}

	return s.store.Roles().RevokeRole(ctx, user.ID, role.ID)
}

// GetUserRoles retrieves all roles assigned to a user.
func (s *RoleService) GetUserRoles(ctx context.Context, username string) ([]*model.Role, error) {
	user, err := s.store.Users().Get(ctx, username)
	if err != nil {
		return nil, err
	}

	return s.store.Roles().ListByUserID(ctx, user.ID)
}
