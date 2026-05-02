package v1

const (
	// RoleKind is the kind for Role resources.
	RoleKind = "Role"
)

// Role represents a role resource.
type Role struct {
	// Kind is the type of the resource.
	Kind string `json:"kind"`
	// APIVersion is the version of the API.
	APIVersion string `json:"apiVersion"`
	// Metadata contains the metadata of the role.
	Metadata RoleMetadata `json:"metadata"`
	// Spec contains the specification of the role.
	Spec RoleSpec `json:"spec"`
	// Status contains the status of the role.
	Status RoleStatus `json:"status,omitzero"`
} // @name Role

// RoleMetadata represents metadata for a role.
type RoleMetadata struct {
	// UID is the unique identifier of the role.
	UID string `json:"uid"`
	// CreatedAt is the timestamp when the role was created.
	CreatedAt Time `json:"createdAt"`
	// UpdatedAt is the timestamp when the role was last updated.
	UpdatedAt Time `json:"updatedAt"`
	// DeletedAt is the timestamp when the role was soft deleted.
	DeletedAt *Time `json:"deletedAt,omitempty"`
} // @name RoleMetadata

// RoleSpec represents the specification of a role.
type RoleSpec struct {
	// DisplayName is the display name of the role.
	DisplayName string `json:"displayName"`
	// Description is the description of the role.
	Description string `json:"description"`
	// Permissions contains the permission names assigned to the role (e.g., "agent:read").
	Permissions []string `json:"permissions,omitempty"`
	// IsBuiltIn indicates whether the role is a built-in role.
	IsBuiltIn bool `json:"isBuiltIn"`
} // @name RoleSpec

// RoleStatus represents the status of a role.
type RoleStatus struct {
	// Conditions contains the conditions of the role.
	Conditions []Condition `json:"conditions,omitempty"`
} // @name RoleStatus
