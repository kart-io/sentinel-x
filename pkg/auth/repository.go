package auth

import "context"

// Repository defines the interface for policy storage
type Repository interface {
	// LoadPolicies loads all policies from storage
	LoadPolicies(ctx context.Context) ([]*Policy, error)

	// SavePolicies saves all policies to storage (overwrite)
	SavePolicies(ctx context.Context, policies []*Policy) error

	// AddPolicy adds a single policy
	AddPolicy(ctx context.Context, policy *Policy) error

	// RemovePolicy removes a single policy
	RemovePolicy(ctx context.Context, policy *Policy) error

	// RemoveFilteredPolicy removes policies matching the filter
	// fieldIndex: the index of the first field to match (0-5)
	// fieldValues: the values to match, starting from fieldIndex
	RemoveFilteredPolicy(ctx context.Context, ptype string, fieldIndex int, fieldValues ...string) error
}
