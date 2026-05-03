package v1

const (
	// RoleBindingKind is the kind for RoleBinding resources.
	RoleBindingKind = "RoleBinding"
)

// RoleBinding represents a binding of a role to a user within a namespace.
type RoleBinding struct {
	// Kind is the type of the resource.
	Kind string `json:"kind" yaml:"kind"`
	// APIVersion is the version of the API.
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	// Metadata contains the metadata of the role binding.
	Metadata RoleBindingMetadata `json:"metadata" yaml:"metadata"`
	// Spec contains the specification of the role binding.
	Spec RoleBindingSpec `json:"spec" yaml:"spec"`
	// Status contains the status of the role binding.
	Status RoleBindingStatus `json:"status,omitzero" yaml:"status,omitempty"`
} // @name RoleBinding

// RoleBindingMetadata represents metadata for a role binding.
type RoleBindingMetadata struct {
	// Namespace is the namespace scope for this role binding.
	Namespace string `json:"namespace" yaml:"namespace"`
	// Name is the unique name of the role binding within the namespace.
	Name string `json:"name" yaml:"name"`
	// CreatedAt is the timestamp when the role binding was created.
	CreatedAt Time `json:"createdAt,omitzero" yaml:"createdAt,omitempty"`
	// UpdatedAt is the timestamp when the role binding was last updated.
	UpdatedAt Time `json:"updatedAt,omitzero" yaml:"updatedAt,omitempty"`
	// DeletedAt is the timestamp when the role binding was soft deleted.
	DeletedAt *Time `json:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
} // @name RoleBindingMetadata

// RoleBindingSpec defines the role binding details.
// Subjects enumerates the principals (e.g. Users) that this binding grants the referenced role to.
type RoleBindingSpec struct {
	// RoleRef references the role to bind.
	RoleRef RoleBindingRoleRef `json:"roleRef" yaml:"roleRef"`
	// Subjects lists the principals that this binding grants the role to.
	Subjects []RoleBindingSubject `json:"subjects,omitempty" yaml:"subjects,omitempty"`
} // @name RoleBindingSpec

// RoleBindingRoleRef references a Role resource.
type RoleBindingRoleRef struct {
	// Kind is the type of the referenced resource (e.g., "Role").
	Kind string `json:"kind" yaml:"kind"`
	// Name is the display name of the role.
	Name string `json:"name" yaml:"name"`
} // @name RoleBindingRoleRef

// RoleBindingSubject references a principal that a RoleBinding grants its role to.
type RoleBindingSubject struct {
	// Kind is the type of subject (e.g., "User").
	Kind string `json:"kind" yaml:"kind"`
	// Name is the principal identifier (for User, the email address).
	Name string `json:"name" yaml:"name"`
	// APIVersion is the API version of the referenced subject; optional.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
} // @name RoleBindingSubject

// RoleBindingStatus represents the current state of the role binding.
type RoleBindingStatus struct {
	// Conditions contains the conditions of the role binding.
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
} // @name RoleBindingStatus
