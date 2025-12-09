// Package casbin provides Casbin-based authorization support.
// It implements policy-based access control using the Casbin library.
package casbin

// Policy represents a Casbin policy rule.
// This type is used by the Casbin adapter for policy storage.
type Policy struct {
	PType string `json:"ptype"` // Policy type (p, g, etc.)
	V0    string `json:"v0"`    // First value (usually subject/role)
	V1    string `json:"v1"`    // Second value (usually object/resource)
	V2    string `json:"v2"`    // Third value (usually action)
	V3    string `json:"v3"`    // Fourth value (optional)
	V4    string `json:"v4"`    // Fifth value (optional)
	V5    string `json:"v5"`    // Sixth value (optional)
}
