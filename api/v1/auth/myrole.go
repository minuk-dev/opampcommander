package auth

import v1 "github.com/minuk-dev/opampcommander/api/v1"

// MyRole represents the current user's role and rolebinding.
type MyRole struct {
	// Role is the role of the user
	Role v1.Role `json:"role"`
	// RoleBinding is the rolebinding of the user
	RoleBinding v1.RoleBinding `json:"rolebinding"`
}
