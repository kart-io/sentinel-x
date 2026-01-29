package biz

import (
	"context"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	"github.com/kart-io/sentinel-x/pkg/security/authz/casbin"
	storepkg "github.com/kart-io/sentinel-x/pkg/store"
)

// RoleService handles role-related business logic.
type RoleService struct {
	roleStore         *store.RoleStore
	userStore         *store.UserStore
	permissionService *casbin.Service
}

// NewRoleService creates a new RoleService.
func NewRoleService(roleStore *store.RoleStore, userStore *store.UserStore, permissionService *casbin.Service) *RoleService {
	return &RoleService{
		roleStore:         roleStore,
		userStore:         userStore,
		permissionService: permissionService,
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

	if err := s.roleStore.AssignRole(ctx, user.ID, role.ID); err != nil {
		return err
	}

	// Add grouping policy to Casbin (g, userID, roleCode)
	_, err = s.permissionService.AddGroupingPolicy(user.ID, role.Code)
	return err
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

	if err := s.roleStore.RevokeRole(ctx, user.ID, role.ID); err != nil {
		return err
	}

	// Remove grouping policy from Casbin
	_, err = s.permissionService.RemoveGroupingPolicy(user.ID, role.Code)
	return err
}

// AssignPermission assigns a permission to a role.
func (s *RoleService) AssignPermission(ctx context.Context, roleCode, resource, action string) error {
	// Ensure role exists
	if _, err := s.roleStore.Get(ctx, roleCode); err != nil {
		return err
	}

	_, err := s.permissionService.AddPolicy(roleCode, resource, action)
	return err
}

// RemovePermission removes a permission from a role.
func (s *RoleService) RemovePermission(ctx context.Context, roleCode, resource, action string) error {
	// Ensure role exists
	if _, err := s.roleStore.Get(ctx, roleCode); err != nil {
		return err
	}

	_, err := s.permissionService.RemovePolicy(roleCode, resource, action)
	return err
}

// CheckPermission checks if a user has permission to access a resource.
func (s *RoleService) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	return s.permissionService.Enforce(userID, resource, action)
}

// GetUserRoles retrieves all roles assigned to a user.
func (s *RoleService) GetUserRoles(ctx context.Context, username string) ([]*model.Role, error) {
	user, err := s.userStore.Get(ctx, username)
	if err != nil {
		return nil, err
	}

	return s.roleStore.ListByUserID(ctx, user.ID)
}
