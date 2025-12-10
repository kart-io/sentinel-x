// Package rbac provides Role-Based Access Control implementation for Sentinel-X.
//
// This package implements the authz.Authorizer interface using RBAC model.
// It supports role hierarchies, wildcards, and deny rules.
//
// Usage:
//
//	rbac := rbac.New()
//
//	// Define roles with permissions
//	rbac.AddRole("admin",
//	    authz.NewPermission("*", "*"),
//	)
//	rbac.AddRole("editor",
//	    authz.NewPermission("posts", "*"),
//	    authz.NewPermission("comments", "*"),
//	)
//	rbac.AddRole("viewer",
//	    authz.NewPermission("posts", "read"),
//	    authz.NewPermission("comments", "read"),
//	)
//
//	// Assign roles to users
//	rbac.AssignRole("user-1", "admin")
//	rbac.AssignRole("user-2", "editor")
//
//	// Check authorization
//	allowed, _ := rbac.Authorize(ctx, "user-2", "posts", "write")
package rbac

import (
	"context"
	"sync"

	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// RBAC implements Role-Based Access Control.
type RBAC struct {
	mu sync.RWMutex

	// roles maps role name to permissions
	roles map[string][]authz.Permission

	// assignments maps subject to roles
	assignments map[string]map[string]struct{}

	// roleHierarchy maps role to parent roles (for inheritance)
	roleHierarchy map[string][]string

	// store is the optional persistent store
	store authz.PolicyStore

	// superAdmin is the role that has all permissions
	superAdmin string
}

// Option is a functional option for RBAC.
type Option func(*RBAC)

// New creates a new RBAC authorizer.
func New(opts ...Option) *RBAC {
	r := &RBAC{
		roles:         make(map[string][]authz.Permission),
		assignments:   make(map[string]map[string]struct{}),
		roleHierarchy: make(map[string][]string),
		superAdmin:    "super_admin",
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithStore sets the policy store.
func WithStore(store authz.PolicyStore) Option {
	return func(r *RBAC) {
		r.store = store
	}
}

// WithSuperAdmin sets the super admin role name.
func WithSuperAdmin(role string) Option {
	return func(r *RBAC) {
		r.superAdmin = role
	}
}

// Authorize checks if the subject can perform the action on the resource.
func (r *RBAC) Authorize(ctx context.Context, subject, resource, action string) (bool, error) {
	return r.AuthorizeWithContext(ctx, subject, resource, action, nil)
}

// AuthorizeWithContext checks authorization with additional context.
func (r *RBAC) AuthorizeWithContext(ctx context.Context, subject, resource, action string, context map[string]interface{}) (bool, error) {
	// Validate all required inputs
	if subject == "" {
		return false, errors.ErrInvalidParam.WithMessage("subject is required")
	}
	if resource == "" {
		return false, errors.ErrInvalidParam.WithMessage("resource is required")
	}
	if action == "" {
		return false, errors.ErrInvalidParam.WithMessage("action is required")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all roles for the subject (including inherited roles)
	roles := r.getAllRoles(subject)

	// Check super admin first
	for _, role := range roles {
		if role == r.superAdmin {
			return true, nil
		}
	}

	// Check each role's permissions
	for _, role := range roles {
		permissions, ok := r.roles[role]
		if !ok {
			continue
		}

		// Check for deny rules first
		for _, perm := range permissions {
			if perm.Effect == authz.EffectDeny && perm.Matches(resource, action) {
				return false, nil
			}
		}

		// Check for allow rules
		for _, perm := range permissions {
			if perm.Effect != authz.EffectDeny && perm.Matches(resource, action) {
				return true, nil
			}
		}
	}

	return false, nil
}

// getAllRoles returns all roles for a subject, including inherited roles.
func (r *RBAC) getAllRoles(subject string) []string {
	directRoles, ok := r.assignments[subject]
	if !ok {
		return nil
	}

	visited := make(map[string]struct{})
	var result []string

	var collect func(role string)
	collect = func(role string) {
		if _, seen := visited[role]; seen {
			return
		}
		visited[role] = struct{}{}
		result = append(result, role)

		// Collect parent roles
		for _, parent := range r.roleHierarchy[role] {
			collect(parent)
		}
	}

	for role := range directRoles {
		collect(role)
	}

	return result
}

// AddRole creates a new role with the given permissions.
func (r *RBAC) AddRole(role string, permissions ...authz.Permission) error {
	if role == "" {
		return errors.ErrInvalidParam.WithMessage("role name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.roles[role] = permissions

	if r.store != nil {
		if err := r.store.SaveRole(context.Background(), role, permissions); err != nil {
			return err
		}
	}

	return nil
}

// RemoveRole removes a role.
func (r *RBAC) RemoveRole(role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.roles, role)
	delete(r.roleHierarchy, role)

	if r.store != nil {
		if err := r.store.DeleteRole(context.Background(), role); err != nil {
			return err
		}
	}

	return nil
}

// GetRole returns the permissions for a role.
func (r *RBAC) GetRole(role string) ([]authz.Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	permissions, ok := r.roles[role]
	if !ok {
		return nil, errors.ErrNotFound.WithMessagef("role not found: %s", role)
	}

	return permissions, nil
}

// AssignRole assigns a role to a subject.
func (r *RBAC) AssignRole(subject, role string) error {
	if subject == "" || role == "" {
		return errors.ErrInvalidParam.WithMessage("subject and role are required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if role exists
	if _, ok := r.roles[role]; !ok {
		return errors.ErrNotFound.WithMessagef("role not found: %s", role)
	}

	if r.assignments[subject] == nil {
		r.assignments[subject] = make(map[string]struct{})
	}
	r.assignments[subject][role] = struct{}{}

	if r.store != nil {
		if err := r.store.SaveRoleAssignment(context.Background(), subject, role); err != nil {
			return err
		}
	}

	return nil
}

// RevokeRole revokes a role from a subject.
func (r *RBAC) RevokeRole(subject, role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if roles, ok := r.assignments[subject]; ok {
		delete(roles, role)
		if len(roles) == 0 {
			delete(r.assignments, subject)
		}
	}

	if r.store != nil {
		if err := r.store.DeleteRoleAssignment(context.Background(), subject, role); err != nil {
			return err
		}
	}

	return nil
}

// GetRoles returns all roles assigned to a subject.
func (r *RBAC) GetRoles(subject string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles, ok := r.assignments[subject]
	if !ok {
		return nil, nil
	}

	result := make([]string, 0, len(roles))
	for role := range roles {
		result = append(result, role)
	}
	return result, nil
}

// HasRole checks if a subject has a specific role.
func (r *RBAC) HasRole(subject, role string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles, ok := r.assignments[subject]
	if !ok {
		return false, nil
	}

	_, has := roles[role]
	return has, nil
}

// SetRoleParent sets the parent roles for a role (inheritance).
func (r *RBAC) SetRoleParent(role string, parents ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if role exists
	if _, ok := r.roles[role]; !ok {
		return errors.ErrNotFound.WithMessagef("role not found: %s", role)
	}

	// Check if parents exist
	for _, parent := range parents {
		if _, ok := r.roles[parent]; !ok {
			return errors.ErrNotFound.WithMessagef("parent role not found: %s", parent)
		}
	}

	// Temporarily set the new hierarchy
	oldParents := r.roleHierarchy[role]
	r.roleHierarchy[role] = parents

	// Detect circular dependency
	if cycle := r.detectCycle(role); cycle != nil {
		// Rollback to old hierarchy
		if oldParents == nil {
			delete(r.roleHierarchy, role)
		} else {
			r.roleHierarchy[role] = oldParents
		}
		return errors.ErrInvalidParam.WithMessagef("circular role dependency detected: %v", cycle)
	}

	return nil
}

// detectCycle detects circular dependencies in role hierarchy using DFS.
// Returns the cycle path if detected, nil otherwise.
func (r *RBAC) detectCycle(startRole string) []string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var dfs func(string) bool
	dfs = func(role string) bool {
		visited[role] = true
		recStack[role] = true
		path = append(path, role)

		// Visit all parent roles
		for _, parent := range r.roleHierarchy[role] {
			if !visited[parent] {
				if dfs(parent) {
					return true
				}
			} else if recStack[parent] {
				// Cycle detected, find the cycle path
				cycleStart := -1
				for i, r := range path {
					if r == parent {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					path = append(path[cycleStart:], parent)
				}
				return true
			}
		}

		recStack[role] = false
		path = path[:len(path)-1]
		return false
	}

	if dfs(startRole) {
		return path
	}
	return nil
}

// ListRoles lists all defined roles.
func (r *RBAC) ListRoles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := make([]string, 0, len(r.roles))
	for role := range r.roles {
		roles = append(roles, role)
	}
	return roles
}

// Clear removes all roles and assignments.
func (r *RBAC) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.roles = make(map[string][]authz.Permission)
	r.assignments = make(map[string]map[string]struct{})
	r.roleHierarchy = make(map[string][]string)
}

// Load loads roles and assignments from the store.
func (r *RBAC) Load(ctx context.Context) error {
	if r.store == nil {
		return nil
	}

	// Load roles
	roles, err := r.store.ListRoles(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, role := range roles {
		permissions, err := r.store.GetRole(ctx, role)
		if err != nil {
			continue
		}
		r.roles[role] = permissions
	}

	return nil
}
