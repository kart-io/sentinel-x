package auth

// Policy represents a Casbin policy rule
type Policy struct {
	PType string `json:"ptype"`
	V0    string `json:"v0"`
	V1    string `json:"v1"`
	V2    string `json:"v2"`
	V3    string `json:"v3"`
	V4    string `json:"v4"`
	V5    string `json:"v5"`
}

// RoleBinding represents a user-role assignment
type RoleBinding struct {
	User string `json:"user"`
	Role string `json:"role"`
}

// Permission represents a permission rule
type Permission struct {
	Role     string `json:"role"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}
