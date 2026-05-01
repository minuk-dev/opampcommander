package usermodel

import (
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

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
// LabelSelector binds the role to any user whose labels match all specified key/value pairs.
type RoleBindingSpec struct {
	RoleRef       RoleRef
	LabelSelector map[string]string
}

// RoleRef references a role by kind and name.
type RoleRef struct {
	Kind string
	Name string
}

// RoleBindingStatus represents the current state of the role binding.
type RoleBindingStatus struct {
	Conditions []model.Condition
}

// NewRoleBinding creates a new RoleBinding instance.
// Set Spec.LabelSelector to define the set of users this binding applies to.
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
			RoleRef:       roleRef,
			LabelSelector: nil,
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
