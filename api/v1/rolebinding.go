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
// Either subject or labelSelector must be provided.
type RoleBindingSpec struct {
	// RoleRef references the role to bind.
	RoleRef RoleBindingRoleRef `json:"roleRef" yaml:"roleRef"`
	// Subject identifies a specific user to bind the role to.
	// Omit when using labelSelector.
	Subject RoleBindingSubject `json:"subject,omitempty" yaml:"subject,omitempty"`
	// LabelSelector binds the role to any user whose metadata labels match all specified key/value pairs.
	// Omit when using subject.
	LabelSelector map[string]string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
} // @name RoleBindingSpec

// RoleBindingRoleRef references a Role resource.
type RoleBindingRoleRef struct {
	// Kind is the type of the referenced resource (e.g., "Role").
	Kind string `json:"kind" yaml:"kind"`
	// Name is the display name of the role.
	Name string `json:"name" yaml:"name"`
} // @name RoleBindingRoleRef

// RoleBindingSubject identifies a user.
type RoleBindingSubject struct {
	// Kind is the type of the subject (e.g., "User").
	Kind string `json:"kind" yaml:"kind"`
	// Name is the email of the user.
	Name string `json:"name" yaml:"name"`
} // @name RoleBindingSubject

// RoleBindingStatus represents the current state of the role binding.
type RoleBindingStatus struct {
	// Conditions contains the conditions of the role binding.
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
} // @name RoleBindingStatus
