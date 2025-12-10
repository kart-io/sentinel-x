package rbac

import (
	"context"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/security/authz"
)

// TestRBACAuthorize tests basic authorization flow.
func TestRBACAuthorize(t *testing.T) {
	rbac := New()
	ctx := context.Background()

	// Add role with permissions
	err := rbac.AddRole("admin", authz.Permission{
		Resource: "*",
		Action:   "*",
		Effect:   authz.EffectAllow,
	})
	if err != nil {
		t.Fatalf("AddRole error: %v", err)
	}

	// Assign role to user
	err = rbac.AssignRole("user-123", "admin")
	if err != nil {
		t.Fatalf("AssignRole error: %v", err)
	}

	// Test authorization
	allowed, err := rbac.Authorize(ctx, "user-123", "posts", "delete")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Admin should be allowed to delete posts")
	}
}

// TestRBACInputValidation tests input validation.
func TestRBACInputValidation(t *testing.T) {
	rbac := New()
	ctx := context.Background()

	tests := []struct {
		name     string
		subject  string
		resource string
		action   string
		wantErr  bool
	}{
		{"empty subject", "", "posts", "read", true},
		{"empty resource", "user-1", "", "read", true},
		{"empty action", "user-1", "posts", "", true},
		{"valid input", "user-1", "posts", "read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := rbac.Authorize(ctx, tt.subject, tt.resource, tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRBACRoleHierarchy tests role inheritance.
func TestRBACRoleHierarchy(t *testing.T) {
	rbac := New()
	ctx := context.Background()

	// Create role hierarchy: manager -> employee -> viewer
	err := rbac.AddRole("viewer", authz.NewPermission("posts", "read"))
	if err != nil {
		t.Fatalf("AddRole viewer error: %v", err)
	}

	err = rbac.AddRole("employee", authz.NewPermission("posts", "write"))
	if err != nil {
		t.Fatalf("AddRole employee error: %v", err)
	}

	err = rbac.AddRole("manager", authz.NewPermission("posts", "delete"))
	if err != nil {
		t.Fatalf("AddRole manager error: %v", err)
	}

	// Set up hierarchy
	err = rbac.SetRoleParent("employee", "viewer")
	if err != nil {
		t.Fatalf("SetRoleParent employee->viewer error: %v", err)
	}

	err = rbac.SetRoleParent("manager", "employee")
	if err != nil {
		t.Fatalf("SetRoleParent manager->employee error: %v", err)
	}

	// Assign manager role to user
	err = rbac.AssignRole("user-1", "manager")
	if err != nil {
		t.Fatalf("AssignRole error: %v", err)
	}

	// Manager should inherit employee's write permission
	allowed, err := rbac.Authorize(ctx, "user-1", "posts", "write")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Manager should inherit employee's write permission")
	}

	// Manager should inherit viewer's read permission
	allowed, err = rbac.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Manager should inherit viewer's read permission")
	}

	// Manager should have own delete permission
	allowed, err = rbac.Authorize(ctx, "user-1", "posts", "delete")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Manager should have delete permission")
	}
}

// TestRBACWildcardPermission tests wildcard permissions.
func TestRBACWildcardPermission(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		permission authz.Permission
		resource   string
		action     string
		expected   bool
	}{
		{
			name:       "wildcard resource and action",
			permission: authz.NewPermission("*", "*"),
			resource:   "posts",
			action:     "delete",
			expected:   true,
		},
		{
			name:       "wildcard action",
			permission: authz.NewPermission("posts", "*"),
			resource:   "posts",
			action:     "read",
			expected:   true,
		},
		{
			name:       "wildcard action - wrong resource",
			permission: authz.NewPermission("posts", "*"),
			resource:   "comments",
			action:     "read",
			expected:   false,
		},
		{
			name:       "specific permission",
			permission: authz.NewPermission("posts", "read"),
			resource:   "posts",
			action:     "read",
			expected:   true,
		},
		{
			name:       "specific permission - wrong action",
			permission: authz.NewPermission("posts", "read"),
			resource:   "posts",
			action:     "write",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh RBAC for each test
			r := New()
			err := r.AddRole("test-role", tt.permission)
			if err != nil {
				t.Fatalf("AddRole error: %v", err)
			}

			err = r.AssignRole("user-1", "test-role")
			if err != nil {
				t.Fatalf("AssignRole error: %v", err)
			}

			allowed, err := r.Authorize(ctx, "user-1", tt.resource, tt.action)
			if err != nil {
				t.Fatalf("Authorize error: %v", err)
			}

			if allowed != tt.expected {
				t.Errorf("Authorize() = %v, expected %v", allowed, tt.expected)
			}
		})
	}
}

// TestRBACSuperAdmin tests super admin bypass.
func TestRBACSuperAdmin(t *testing.T) {
	rbac := New(WithSuperAdmin("super_admin"))
	ctx := context.Background()

	// Create super admin role (can be empty permissions)
	err := rbac.AddRole("super_admin")
	if err != nil {
		t.Fatalf("AddRole error: %v", err)
	}

	// Assign super admin role
	err = rbac.AssignRole("user-super", "super_admin")
	if err != nil {
		t.Fatalf("AssignRole error: %v", err)
	}

	// Super admin should be allowed to do anything
	tests := []struct {
		resource string
		action   string
	}{
		{"posts", "read"},
		{"posts", "write"},
		{"posts", "delete"},
		{"users", "create"},
		{"settings", "update"},
	}

	for _, tt := range tests {
		t.Run(tt.resource+":"+tt.action, func(t *testing.T) {
			allowed, err := rbac.Authorize(ctx, "user-super", tt.resource, tt.action)
			if err != nil {
				t.Fatalf("Authorize error: %v", err)
			}
			if !allowed {
				t.Errorf("Super admin should be allowed for %s:%s", tt.resource, tt.action)
			}
		})
	}
}

// TestRBACDenyRule tests deny rules take precedence.
func TestRBACDenyRule(t *testing.T) {
	rbac := New()
	ctx := context.Background()

	// Add role with allow and deny permissions
	err := rbac.AddRole("editor",
		authz.NewPermission("posts", "*"), // Allow all actions on posts
		authz.Permission{ // Deny delete
			Resource: "posts",
			Action:   "delete",
			Effect:   authz.EffectDeny,
		},
	)
	if err != nil {
		t.Fatalf("AddRole error: %v", err)
	}

	err = rbac.AssignRole("user-1", "editor")
	if err != nil {
		t.Fatalf("AssignRole error: %v", err)
	}

	// Should be allowed to read
	allowed, err := rbac.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("Should be allowed to read posts")
	}

	// Should be denied to delete
	allowed, err = rbac.Authorize(ctx, "user-1", "posts", "delete")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("Should be denied to delete posts")
	}
}

// TestRBACRoleManagement tests role management operations.
func TestRBACRoleManagement(t *testing.T) {
	rbac := New()

	// Test AddRole with empty name
	err := rbac.AddRole("", authz.NewPermission("posts", "read"))
	if err == nil {
		t.Error("AddRole with empty name should return error")
	}

	// Test AddRole
	err = rbac.AddRole("viewer", authz.NewPermission("posts", "read"))
	if err != nil {
		t.Fatalf("AddRole error: %v", err)
	}

	// Test GetRole
	perms, err := rbac.GetRole("viewer")
	if err != nil {
		t.Fatalf("GetRole error: %v", err)
	}
	if len(perms) != 1 {
		t.Errorf("Expected 1 permission, got %d", len(perms))
	}

	// Test GetRole - non-existent
	_, err = rbac.GetRole("nonexistent")
	if err == nil {
		t.Error("GetRole for non-existent role should return error")
	}

	// Test ListRoles
	roles := rbac.ListRoles()
	if len(roles) != 1 || roles[0] != "viewer" {
		t.Errorf("ListRoles = %v, expected [viewer]", roles)
	}

	// Test RemoveRole
	err = rbac.RemoveRole("viewer")
	if err != nil {
		t.Fatalf("RemoveRole error: %v", err)
	}

	roles = rbac.ListRoles()
	if len(roles) != 0 {
		t.Errorf("ListRoles after remove = %v, expected []", roles)
	}
}

// TestRBACRoleAssignment tests role assignment operations.
func TestRBACRoleAssignment(t *testing.T) {
	rbac := New()

	// Create a role first
	err := rbac.AddRole("viewer", authz.NewPermission("posts", "read"))
	if err != nil {
		t.Fatalf("AddRole error: %v", err)
	}

	// Test AssignRole with empty subject
	err = rbac.AssignRole("", "viewer")
	if err == nil {
		t.Error("AssignRole with empty subject should return error")
	}

	// Test AssignRole with empty role
	err = rbac.AssignRole("user-1", "")
	if err == nil {
		t.Error("AssignRole with empty role should return error")
	}

	// Test AssignRole with non-existent role
	err = rbac.AssignRole("user-1", "nonexistent")
	if err == nil {
		t.Error("AssignRole with non-existent role should return error")
	}

	// Test AssignRole
	err = rbac.AssignRole("user-1", "viewer")
	if err != nil {
		t.Fatalf("AssignRole error: %v", err)
	}

	// Test HasRole
	has, err := rbac.HasRole("user-1", "viewer")
	if err != nil {
		t.Fatalf("HasRole error: %v", err)
	}
	if !has {
		t.Error("HasRole should return true")
	}

	// Test GetRoles
	roles, err := rbac.GetRoles("user-1")
	if err != nil {
		t.Fatalf("GetRoles error: %v", err)
	}
	if len(roles) != 1 || roles[0] != "viewer" {
		t.Errorf("GetRoles = %v, expected [viewer]", roles)
	}

	// Test RevokeRole
	err = rbac.RevokeRole("user-1", "viewer")
	if err != nil {
		t.Fatalf("RevokeRole error: %v", err)
	}

	has, err = rbac.HasRole("user-1", "viewer")
	if err != nil {
		t.Fatalf("HasRole error: %v", err)
	}
	if has {
		t.Error("HasRole should return false after revoke")
	}
}

// TestRBACMultipleRoles tests user with multiple roles.
func TestRBACMultipleRoles(t *testing.T) {
	rbac := New()
	ctx := context.Background()

	// Create roles
	err := rbac.AddRole("reader", authz.NewPermission("posts", "read"))
	if err != nil {
		t.Fatalf("AddRole reader error: %v", err)
	}

	err = rbac.AddRole("writer", authz.NewPermission("posts", "write"))
	if err != nil {
		t.Fatalf("AddRole writer error: %v", err)
	}

	// Assign multiple roles
	err = rbac.AssignRole("user-1", "reader")
	if err != nil {
		t.Fatalf("AssignRole reader error: %v", err)
	}

	err = rbac.AssignRole("user-1", "writer")
	if err != nil {
		t.Fatalf("AssignRole writer error: %v", err)
	}

	// Should have both permissions
	allowed, err := rbac.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize read error: %v", err)
	}
	if !allowed {
		t.Error("Should be allowed to read")
	}

	allowed, err = rbac.Authorize(ctx, "user-1", "posts", "write")
	if err != nil {
		t.Fatalf("Authorize write error: %v", err)
	}
	if !allowed {
		t.Error("Should be allowed to write")
	}

	// Should not have delete permission
	allowed, err = rbac.Authorize(ctx, "user-1", "posts", "delete")
	if err != nil {
		t.Fatalf("Authorize delete error: %v", err)
	}
	if allowed {
		t.Error("Should not be allowed to delete")
	}
}

// TestRBACClear tests clearing all roles and assignments.
func TestRBACClear(t *testing.T) {
	rbac := New()

	// Add roles and assignments
	_ = rbac.AddRole("admin", authz.NewPermission("*", "*"))
	_ = rbac.AssignRole("user-1", "admin")

	// Clear
	rbac.Clear()

	// Check everything is cleared
	roles := rbac.ListRoles()
	if len(roles) != 0 {
		t.Errorf("ListRoles after clear = %v, expected []", roles)
	}

	userRoles, _ := rbac.GetRoles("user-1")
	if len(userRoles) != 0 {
		t.Errorf("GetRoles after clear = %v, expected []", userRoles)
	}
}

// TestRBACNoRoleAssignment tests authorization without role assignment.
func TestRBACNoRoleAssignment(t *testing.T) {
	rbac := New()
	ctx := context.Background()

	// Create role but don't assign it
	err := rbac.AddRole("admin", authz.NewPermission("*", "*"))
	if err != nil {
		t.Fatalf("AddRole error: %v", err)
	}

	// User without role should be denied
	allowed, err := rbac.Authorize(ctx, "user-1", "posts", "read")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("User without role should be denied")
	}
}

// TestRBACSetRoleParentValidation tests role parent validation.
func TestRBACSetRoleParentValidation(t *testing.T) {
	rbac := New()

	// Create roles
	_ = rbac.AddRole("child", authz.NewPermission("posts", "read"))
	_ = rbac.AddRole("parent", authz.NewPermission("posts", "write"))

	// Test setting parent for non-existent role
	err := rbac.SetRoleParent("nonexistent", "parent")
	if err == nil {
		t.Error("SetRoleParent for non-existent role should return error")
	}

	// Test setting non-existent parent
	err = rbac.SetRoleParent("child", "nonexistent")
	if err == nil {
		t.Error("SetRoleParent with non-existent parent should return error")
	}

	// Test valid parent assignment
	err = rbac.SetRoleParent("child", "parent")
	if err != nil {
		t.Fatalf("SetRoleParent error: %v", err)
	}
}

// TestRBACCircularDependencyDetection tests circular dependency detection in role hierarchy.
func TestRBACCircularDependencyDetection(t *testing.T) {
	rbac := New()

	// Create roles for testing circular dependencies
	_ = rbac.AddRole("roleA", authz.NewPermission("posts", "read"))
	_ = rbac.AddRole("roleB", authz.NewPermission("posts", "write"))
	_ = rbac.AddRole("roleC", authz.NewPermission("posts", "delete"))
	_ = rbac.AddRole("roleD", authz.NewPermission("comments", "read"))

	tests := []struct {
		name      string
		setup     func()
		role      string
		parents   []string
		wantError bool
		desc      string
	}{
		{
			name:      "direct self-reference",
			setup:     func() {},
			role:      "roleA",
			parents:   []string{"roleA"},
			wantError: true,
			desc:      "role cannot be its own parent",
		},
		{
			name: "two-node cycle",
			setup: func() {
				_ = rbac.SetRoleParent("roleA", "roleB")
			},
			role:      "roleB",
			parents:   []string{"roleA"},
			wantError: true,
			desc:      "A->B, B->A creates a cycle",
		},
		{
			name: "three-node cycle",
			setup: func() {
				rbac.Clear()
				_ = rbac.AddRole("roleA", authz.NewPermission("posts", "read"))
				_ = rbac.AddRole("roleB", authz.NewPermission("posts", "write"))
				_ = rbac.AddRole("roleC", authz.NewPermission("posts", "delete"))
				_ = rbac.SetRoleParent("roleA", "roleB")
				_ = rbac.SetRoleParent("roleB", "roleC")
			},
			role:      "roleC",
			parents:   []string{"roleA"},
			wantError: true,
			desc:      "A->B->C, C->A creates a cycle",
		},
		{
			name: "four-node cycle",
			setup: func() {
				rbac.Clear()
				_ = rbac.AddRole("roleA", authz.NewPermission("posts", "read"))
				_ = rbac.AddRole("roleB", authz.NewPermission("posts", "write"))
				_ = rbac.AddRole("roleC", authz.NewPermission("posts", "delete"))
				_ = rbac.AddRole("roleD", authz.NewPermission("comments", "read"))
				_ = rbac.SetRoleParent("roleA", "roleB")
				_ = rbac.SetRoleParent("roleB", "roleC")
				_ = rbac.SetRoleParent("roleC", "roleD")
			},
			role:      "roleD",
			parents:   []string{"roleA"},
			wantError: true,
			desc:      "A->B->C->D, D->A creates a cycle",
		},
		{
			name: "valid linear hierarchy",
			setup: func() {
				rbac.Clear()
				_ = rbac.AddRole("roleA", authz.NewPermission("posts", "read"))
				_ = rbac.AddRole("roleB", authz.NewPermission("posts", "write"))
				_ = rbac.AddRole("roleC", authz.NewPermission("posts", "delete"))
				_ = rbac.AddRole("roleD", authz.NewPermission("comments", "read"))
			},
			role:      "roleA",
			parents:   []string{"roleB"},
			wantError: false,
			desc:      "A->B (no cycle)",
		},
		{
			name: "valid multi-parent hierarchy",
			setup: func() {
				rbac.Clear()
				_ = rbac.AddRole("roleA", authz.NewPermission("posts", "read"))
				_ = rbac.AddRole("roleB", authz.NewPermission("posts", "write"))
				_ = rbac.AddRole("roleC", authz.NewPermission("posts", "delete"))
				_ = rbac.AddRole("roleD", authz.NewPermission("comments", "read"))
			},
			role:      "roleA",
			parents:   []string{"roleB", "roleC"},
			wantError: false,
			desc:      "A->B and A->C (no cycle)",
		},
		{
			name: "cycle with multiple parents",
			setup: func() {
				rbac.Clear()
				_ = rbac.AddRole("roleA", authz.NewPermission("posts", "read"))
				_ = rbac.AddRole("roleB", authz.NewPermission("posts", "write"))
				_ = rbac.AddRole("roleC", authz.NewPermission("posts", "delete"))
				_ = rbac.SetRoleParent("roleB", "roleC")
			},
			role:      "roleC",
			parents:   []string{"roleA", "roleB"},
			wantError: true,
			desc:      "C->(A,B), B->C creates a cycle through B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			err := rbac.SetRoleParent(tt.role, tt.parents...)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for %s, but got nil", tt.desc)
				} else {
					t.Logf("Correctly detected circular dependency: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.desc, err)
				}
			}
		})
	}
}

// TestRBACCircularDependencyRollback tests that failed circular dependency attempts don't modify state.
func TestRBACCircularDependencyRollback(t *testing.T) {
	rbac := New()

	// Create roles
	_ = rbac.AddRole("roleA", authz.NewPermission("posts", "read"))
	_ = rbac.AddRole("roleB", authz.NewPermission("posts", "write"))
	_ = rbac.AddRole("roleC", authz.NewPermission("posts", "delete"))

	// Set up valid hierarchy
	err := rbac.SetRoleParent("roleA", "roleB")
	if err != nil {
		t.Fatalf("Initial SetRoleParent failed: %v", err)
	}

	// Attempt to create a cycle
	err = rbac.SetRoleParent("roleB", "roleA")
	if err == nil {
		t.Fatal("Expected circular dependency error, got nil")
	}

	// Verify roleB's parent wasn't changed
	rbac.mu.RLock()
	parents := rbac.roleHierarchy["roleB"]
	rbac.mu.RUnlock()

	if len(parents) != 0 {
		t.Errorf("roleB should have no parents after rollback, got %v", parents)
	}

	// Verify roleA's parent is still roleB
	rbac.mu.RLock()
	parentsA := rbac.roleHierarchy["roleA"]
	rbac.mu.RUnlock()

	if len(parentsA) != 1 || parentsA[0] != "roleB" {
		t.Errorf("roleA's parent should still be roleB, got %v", parentsA)
	}
}

// TestRBACComplexHierarchyNoCycle tests complex valid hierarchy without cycles.
func TestRBACComplexHierarchyNoCycle(t *testing.T) {
	rbac := New()

	// Create a diamond hierarchy:
	//       D
	//      / \
	//     B   C
	//      \ /
	//       A
	_ = rbac.AddRole("roleA", authz.NewPermission("posts", "read"))
	_ = rbac.AddRole("roleB", authz.NewPermission("posts", "write"))
	_ = rbac.AddRole("roleC", authz.NewPermission("comments", "write"))
	_ = rbac.AddRole("roleD", authz.NewPermission("posts", "delete"))

	err := rbac.SetRoleParent("roleA", "roleB", "roleC")
	if err != nil {
		t.Fatalf("SetRoleParent A->(B,C) failed: %v", err)
	}

	err = rbac.SetRoleParent("roleB", "roleD")
	if err != nil {
		t.Fatalf("SetRoleParent B->D failed: %v", err)
	}

	err = rbac.SetRoleParent("roleC", "roleD")
	if err != nil {
		t.Fatalf("SetRoleParent C->D failed: %v", err)
	}

	// Now try to create a cycle from D to A (should fail)
	err = rbac.SetRoleParent("roleD", "roleA")
	if err == nil {
		t.Error("Expected circular dependency error for diamond cycle, got nil")
	}
}
