package v1

const (
	// UserKind is the kind for User resources.
	UserKind = "User"
	// RoleKind is the kind for Role resources.
	RoleKind = "Role"
	// PermissionKind is the kind for Permission resources.
	PermissionKind = "Permission"
	// UserRoleKind is the kind for UserRole resources.
	UserRoleKind = "UserRole"
)

// User represents an authenticated user resource.
type User struct {
	// Kind is the type of the resource.
	Kind string `json:"kind"`
	// APIVersion is the version of the API.
	APIVersion string `json:"apiVersion"`
	// Metadata contains the metadata of the user.
	Metadata UserMetadata `json:"metadata"`
	// Spec contains the specification of the user.
	Spec UserSpec `json:"spec"`
	// Status contains the status of the user.
	Status UserStatus `json:"status,omitzero"`
} // @name User

// UserMetadata represents metadata for a user.
type UserMetadata struct {
	// UID is the unique identifier of the user.
	UID string `json:"uid"`
	// CreatedAt is the timestamp when the user was created.
	CreatedAt Time `json:"createdAt"`
	// UpdatedAt is the timestamp when the user was last updated.
	UpdatedAt Time `json:"updatedAt"`
	// DeletedAt is the timestamp when the user was soft deleted.
	DeletedAt *Time `json:"deletedAt,omitempty"`
} // @name UserMetadata

// UserSpec represents the specification of a user.
type UserSpec struct {
	// Email is the email address of the user.
	Email string `json:"email"`
	// Username is the username of the user.
	Username string `json:"username"`
	// IsActive indicates whether the user is active.
	IsActive bool `json:"isActive"`
} // @name UserSpec

// UserStatus represents the status of a user.
type UserStatus struct {
	// Conditions contains the conditions of the user.
	Conditions []Condition `json:"conditions,omitempty"`
	// Roles contains the role IDs assigned to the user.
	Roles []string `json:"roles,omitempty"`
} // @name UserStatus

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

// Permission represents a permission resource.
type Permission struct {
	// Kind is the type of the resource.
	Kind string `json:"kind"`
	// APIVersion is the version of the API.
	APIVersion string `json:"apiVersion"`
	// Metadata contains the metadata of the permission.
	Metadata PermissionMetadata `json:"metadata"`
	// Spec contains the specification of the permission.
	Spec PermissionSpec `json:"spec"`
	// Status contains the status of the permission.
	Status PermissionStatus `json:"status,omitzero"`
} // @name Permission

// PermissionMetadata represents metadata for a permission.
type PermissionMetadata struct {
	// UID is the unique identifier of the permission.
	UID string `json:"uid"`
	// CreatedAt is the timestamp when the permission was created.
	CreatedAt Time `json:"createdAt"`
	// UpdatedAt is the timestamp when the permission was last updated.
	UpdatedAt Time `json:"updatedAt"`
	// DeletedAt is the timestamp when the permission was soft deleted.
	DeletedAt *Time `json:"deletedAt,omitempty"`
} // @name PermissionMetadata

// PermissionSpec represents the specification of a permission.
type PermissionSpec struct {
	// Name is the name of the permission (e.g., "agent:read").
	Name string `json:"name"`
	// Description is the description of the permission.
	Description string `json:"description"`
	// Resource is the resource this permission applies to.
	Resource string `json:"resource"`
	// Action is the action this permission grants.
	Action string `json:"action"`
	// IsBuiltIn indicates whether the permission is a built-in permission.
	IsBuiltIn bool `json:"isBuiltIn"`
} // @name PermissionSpec

// PermissionStatus represents the status of a permission.
type PermissionStatus struct {
	// Conditions contains the conditions of the permission.
	Conditions []Condition `json:"conditions,omitempty"`
} // @name PermissionStatus

// UserRole represents a user-role assignment resource.
type UserRole struct {
	// Kind is the type of the resource.
	Kind string `json:"kind"`
	// APIVersion is the version of the API.
	APIVersion string `json:"apiVersion"`
	// Metadata contains the metadata of the user role assignment.
	Metadata UserRoleMetadata `json:"metadata"`
	// Spec contains the specification of the user role assignment.
	Spec UserRoleSpec `json:"spec"`
	// Status contains the status of the user role assignment.
	Status UserRoleStatus `json:"status,omitzero"`
} // @name UserRole

// UserRoleMetadata represents metadata for a user role assignment.
type UserRoleMetadata struct {
	// UID is the unique identifier of the user role assignment.
	UID string `json:"uid"`
	// CreatedAt is the timestamp when the assignment was created.
	CreatedAt Time `json:"createdAt"`
	// UpdatedAt is the timestamp when the assignment was last updated.
	UpdatedAt Time `json:"updatedAt"`
	// DeletedAt is the timestamp when the assignment was soft deleted.
	DeletedAt *Time `json:"deletedAt,omitempty"`
} // @name UserRoleMetadata

// UserRoleSpec represents the specification of a user role assignment.
type UserRoleSpec struct {
	// UserID is the ID of the user.
	UserID string `json:"userId"`
	// RoleID is the ID of the role.
	RoleID string `json:"roleId"`
	// Namespace is the namespace scope for this role assignment. "*" means all namespaces.
	Namespace string `json:"namespace"`
	// AssignedAt is the timestamp when the role was assigned.
	AssignedAt Time `json:"assignedAt"`
	// AssignedBy is the ID of the user who assigned the role.
	AssignedBy string `json:"assignedBy"`
} // @name UserRoleSpec

// UserRoleStatus represents the status of a user role assignment.
type UserRoleStatus struct {
	// Conditions contains the conditions of the user role assignment.
	Conditions []Condition `json:"conditions,omitempty"`
} // @name UserRoleStatus

// AssignRoleRequest represents a request to assign a role to a user.
type AssignRoleRequest struct {
	// UserID is the ID of the user to assign the role to.
	UserID string `json:"userId"`
	// RoleID is the ID of the role to assign.
	RoleID string `json:"roleId"`
	// Namespace is the namespace scope for this assignment. "*" means all namespaces.
	Namespace string `json:"namespace"`
	// AssignedBy is the ID of the user performing the assignment.
	AssignedBy string `json:"assignedBy"`
} // @name AssignRoleRequest

// CheckPermissionRequest represents a request to check a user's permission.
type CheckPermissionRequest struct {
	// UserID is the ID of the user to check.
	UserID string `json:"userId"`
	// Namespace is the namespace to check permission in.
	Namespace string `json:"namespace"`
	// Resource is the resource to check access for.
	Resource string `json:"resource"`
	// Action is the action to check access for.
	Action string `json:"action"`
} // @name CheckPermissionRequest

// CheckPermissionResponse represents the response for a permission check.
type CheckPermissionResponse struct {
	// Allowed indicates whether the user has the permission.
	Allowed bool `json:"allowed"`
} // @name CheckPermissionResponse

// UserProfileResponse represents a user's profile with their roles and permissions.
type UserProfileResponse struct {
	// User contains the user information.
	User User `json:"user"`
	// Roles contains the roles assigned to the user.
	Roles []Role `json:"roles"`
	// Permissions contains the effective permissions for the user.
	Permissions []Permission `json:"permissions"`
} // @name UserProfileResponse
