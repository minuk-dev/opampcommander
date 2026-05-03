package usermodel

import (
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// SubjectKindUser identifies a Subject that names an individual user by email.
const SubjectKindUser = "User"

// RoleBinding represents a binding of a role to a user within a namespace.
type RoleBinding struct {
	Metadata RoleBindingMetadata
	Spec     RoleBindingSpec
	Status   RoleBindingStatus
}

// RoleBindingMetadata contains metadata about the role binding.
type RoleBindingMetadata struct {
	Namespace string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// RoleBindingSpec defines the role binding details.
// Subjects enumerates the principals (e.g. Users) that this binding grants the referenced role to.
type RoleBindingSpec struct {
	RoleRef  RoleRef
	Subjects []Subject
}

// RoleRef references a role by kind and name.
type RoleRef struct {
	Kind string
	Name string
}

// Subject names a principal that a RoleBinding grants its role to.
// Kind is e.g. "User"; Name is the principal identifier (for User, the email address).
type Subject struct {
	Kind       string
	Name       string
	APIVersion string
}

// RoleBindingStatus represents the current state of the role binding.
type RoleBindingStatus struct {
	Conditions []model.Condition
}

// NewRoleBinding creates a new RoleBinding instance.
// Set Spec.Subjects to define the set of principals this binding applies to.
func NewRoleBinding(namespace, name string, roleRef RoleRef) *RoleBinding {
	now := time.Now()

	return &RoleBinding{
		Metadata: RoleBindingMetadata{
			Namespace: namespace,
			Name:      name,
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
		},
		Spec: RoleBindingSpec{
			RoleRef:  roleRef,
			Subjects: nil,
		},
		Status: RoleBindingStatus{
			Conditions: []model.Condition{},
		},
	}
}

// IsDeleted returns whether the role binding is soft-deleted.
func (rb *RoleBinding) IsDeleted() bool {
	return rb.Metadata.DeletedAt != nil
}

// MarkDeleted marks the role binding as deleted.
func (rb *RoleBinding) MarkDeleted() {
	now := time.Now()
	rb.Metadata.DeletedAt = &now
}

// SetUpdatedAt sets the updatedAt timestamp.
func (rb *RoleBinding) SetUpdatedAt(t time.Time) {
	rb.Metadata.UpdatedAt = t
}

// MatchesUser returns true when any of the binding's Subjects names the given user.
// Currently only Subject.Kind == "User" is supported, matching against the user's email.
func (rb *RoleBinding) MatchesUser(user *User) bool {
	if user == nil {
		return false
	}

	for _, s := range rb.Spec.Subjects {
		if s.Kind == SubjectKindUser && s.Name == user.Spec.Email {
			return true
		}
	}

	return false
}
