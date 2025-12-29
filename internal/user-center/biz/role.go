package biz

import (
	"context"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	storepkg "github.com/kart-io/sentinel-x/pkg/store"
)

// RoleService handles role-related business logic.
type RoleService struct {
	roleStore *store.RoleStore
	userStore *store.UserStore
}

// NewRoleService creates a new RoleService.
func NewRoleService(roleStore *store.RoleStore, userStore *store.UserStore) *RoleService {
	return &RoleService{
		roleStore: roleStore,
		userStore: userStore,
	}
}

// Create creates a new role.
func (s *RoleService) Create(ctx context.Context, role *model.Role) error {
	return s.roleStore.Create(ctx, role)
}

// Update updates an existing role.
func (s *RoleService) Update(ctx context.Context, role *model.Role) error {
	return s.roleStore.Update(ctx, role)
}

// Delete deletes a role by code.
func (s *RoleService) Delete(ctx context.Context, code string) error {
	return s.roleStore.Delete(ctx, code)
}

// Get retrieves a role by code.
func (s *RoleService) Get(ctx context.Context, code string) (*model.Role, error) {
	return s.roleStore.Get(ctx, code)
}

// List lists roles with pagination.
func (s *RoleService) List(ctx context.Context, opts ...storepkg.Option) (int64, []*model.Role, error) {
	return s.roleStore.List(ctx, opts...)
}

// AssignRoleToUser assigns a specific role to a user.
func (s *RoleService) AssignRoleToUser(ctx context.Context, username, roleCode string) error {
	user, err := s.userStore.Get(ctx, username)
	if err != nil {
		return err
	}

	role, err := s.roleStore.Get(ctx, roleCode)
	if err != nil {
		return err
	}

	return s.roleStore.AssignRole(ctx, user.ID, role.ID)
}

// RevokeRoleFromUser revokes a role from a user.
func (s *RoleService) RevokeRoleFromUser(ctx context.Context, username, roleCode string) error {
	user, err := s.userStore.Get(ctx, username)
	if err != nil {
		return err
	}

	role, err := s.roleStore.Get(ctx, roleCode)
	if err != nil {
		return err
	}

	return s.roleStore.RevokeRole(ctx, user.ID, role.ID)
}

// GetUserRoles retrieves all roles assigned to a user.
func (s *RoleService) GetUserRoles(ctx context.Context, username string) ([]*model.Role, error) {
	user, err := s.userStore.Get(ctx, username)
	if err != nil {
		return nil, err
	}

	return s.roleStore.ListByUserID(ctx, user.ID)
}
