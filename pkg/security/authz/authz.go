// Package authz provides authorization interfaces and implementations for Sentinel-X.
//
// This package follows the Sentinel-X project authorization architecture pattern,
// providing a unified authorization interface that supports RBAC (Role-Based Access Control)
// and policy-based authorization.
//
// The authorization flow:
//  1. After authentication, the user's identity (subject) is known
//  2. The request specifies a resource and action (e.g., "users", "read")
//  3. Authorizer.Authorize() checks if the subject can perform the action on the resource
//  4. Access is granted or denied based on roles/policies
//
// Design Principles:
//   - Interface-driven: All authorizers implement the Authorizer interface
//   - RBAC support: Built-in role-based access control
//   - Policy extensible: Can be extended with Casbin or custom policy engines
//   - Centralized: All authorization logic in one place
//
// Usage:
//
//	// Create RBAC authorizer
//	authz := rbac.New()
//
//	// Add roles and permissions
//	authz.AddRole("admin", "users", "read", "write", "delete")
//	authz.AddRole("viewer", "users", "read")
//
//	// Assign role to user
//	authz.AssignRole("user-123", "admin")
//
//	// Check authorization
//	allowed, err := authz.Authorize(ctx, "user-123", "users", "write")
package authz

import (
	"context"
)

// Authorizer defines the authorization interface.
// All authorization providers must implement this interface.
type Authorizer interface {
	// Authorize checks if the subject can perform the action on the resource.
	// subject: the user or entity performing the action (e.g., user ID)
	// resource: the resource being accessed (e.g., "users", "posts", "/api/v1/users")
	// action: the action being performed (e.g., "read", "write", "delete", "GET", "POST")
	// Returns true if authorized, false otherwise.
	Authorize(ctx context.Context, subject, resource, action string) (bool, error)

	// AuthorizeWithContext checks authorization with additional context.
	// The context map can contain additional information like resource owner, attributes, etc.
	AuthorizeWithContext(ctx context.Context, subject, resource, action string, context map[string]interface{}) (bool, error)
}

// RoleManager defines the interface for role management.
type RoleManager interface {
	// AddRole creates a new role with the given permissions.
	AddRole(role string, permissions ...Permission) error

	// RemoveRole removes a role.
	RemoveRole(role string) error

	// GetRole returns the permissions for a role.
	GetRole(role string) ([]Permission, error)

	// AssignRole assigns a role to a subject.
	AssignRole(subject, role string) error

	// RevokeRole revokes a role from a subject.
	RevokeRole(subject, role string) error

	// GetRoles returns all roles assigned to a subject.
	GetRoles(subject string) ([]string, error)

	// HasRole checks if a subject has a specific role.
	HasRole(subject, role string) (bool, error)
}

// Permission represents a permission for a resource and action.
type Permission struct {
	// Resource is the resource being accessed.
	Resource string `json:"resource"`

	// Action is the action being performed.
	// Can be a specific action (e.g., "read") or wildcard ("*").
	Action string `json:"action"`

	// Effect is the permission effect (allow or deny).
	// Default is "allow".
	Effect Effect `json:"effect,omitempty"`

	// Conditions are optional conditions for the permission.
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// Effect represents the effect of a permission.
type Effect string

const (
	// EffectAllow allows the action.
	EffectAllow Effect = "allow"

	// EffectDeny denies the action.
	EffectDeny Effect = "deny"
)

// NewPermission creates a new permission.
func NewPermission(resource, action string) Permission {
	return Permission{
		Resource: resource,
		Action:   action,
		Effect:   EffectAllow,
	}
}

// WithEffect sets the permission effect.
func (p Permission) WithEffect(effect Effect) Permission {
	p.Effect = effect
	return p
}

// WithConditions sets the permission conditions.
func (p Permission) WithConditions(conditions map[string]interface{}) Permission {
	p.Conditions = conditions
	return p
}

// Matches checks if the permission matches the given resource and action.
func (p Permission) Matches(resource, action string) bool {
	return p.matchesResource(resource) && p.matchesAction(action)
}

// matchesResource checks if the permission resource matches.
func (p Permission) matchesResource(resource string) bool {
	if p.Resource == "*" {
		return true
	}
	return p.Resource == resource
}

// matchesAction checks if the permission action matches.
func (p Permission) matchesAction(action string) bool {
	if p.Action == "*" {
		return true
	}
	return p.Action == action
}

// PolicyLoader defines the interface for loading policies.
type PolicyLoader interface {
	// Load loads policies from the source.
	Load(ctx context.Context) error

	// Reload reloads policies from the source.
	Reload(ctx context.Context) error
}

// PolicyStore defines the interface for storing policies.
type PolicyStore interface {
	// SaveRole saves a role with its permissions.
	SaveRole(ctx context.Context, role string, permissions []Permission) error

	// DeleteRole deletes a role.
	DeleteRole(ctx context.Context, role string) error

	// GetRole retrieves a role's permissions.
	GetRole(ctx context.Context, role string) ([]Permission, error)

	// ListRoles lists all roles.
	ListRoles(ctx context.Context) ([]string, error)

	// SaveRoleAssignment saves a role assignment.
	SaveRoleAssignment(ctx context.Context, subject, role string) error

	// DeleteRoleAssignment deletes a role assignment.
	DeleteRoleAssignment(ctx context.Context, subject, role string) error

	// GetRoleAssignments gets all roles for a subject.
	GetRoleAssignments(ctx context.Context, subject string) ([]string, error)
}

// AuthorizationRequest represents an authorization request.
type AuthorizationRequest struct {
	// Subject is the entity requesting access.
	Subject string

	// Resource is the resource being accessed.
	Resource string

	// Action is the action being performed.
	Action string

	// Context contains additional authorization context.
	Context map[string]interface{}
}

// AuthorizationResult represents an authorization result.
type AuthorizationResult struct {
	// Allowed indicates if the request is authorized.
	Allowed bool

	// Reason explains why the request was allowed or denied.
	Reason string

	// MatchedPermission is the permission that matched (if any).
	MatchedPermission *Permission
}

// Decision represents an authorization decision.
type Decision struct {
	// Allowed is the authorization decision.
	Allowed bool `json:"allowed"`

	// Reason is the reason for the decision.
	Reason string `json:"reason,omitempty"`

	// Subject is the subject that was authorized.
	Subject string `json:"subject,omitempty"`

	// Resource is the resource that was accessed.
	Resource string `json:"resource,omitempty"`

	// Action is the action that was performed.
	Action string `json:"action,omitempty"`
}
