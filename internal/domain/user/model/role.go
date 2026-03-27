package usermodel

import (
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Role represents a role that can be assigned to users.
type Role struct {
	Metadata RoleMetadata
	Spec     RoleSpec
	Status   RoleStatus
}

// RoleMetadata contains metadata about the role.
type RoleMetadata struct {
	UID       uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// RoleSpec defines the desired state of the role.
type RoleSpec struct {
	DisplayName string
	Description string
	Permissions []string // Permission names (e.g., "agent:read")
	IsBuiltIn   bool
}

// RoleStatus represents the current state of the role.
type RoleStatus struct {
	Conditions []model.Condition
}

// NewRole creates a new role with the given display name.
func NewRole(displayName string, isBuiltIn bool) *Role {
	now := time.Now()

	return &Role{
		Metadata: RoleMetadata{
			UID:       uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
		},
		Spec: RoleSpec{
			DisplayName: displayName,
			Description: "",
			IsBuiltIn:   isBuiltIn,
			Permissions: []string{},
		},
		Status: RoleStatus{
			Conditions: []model.Condition{},
		},
	}
}

// IsDeleted returns whether the role is deleted.
func (r *Role) IsDeleted() bool {
	return r.Metadata.DeletedAt != nil
}

// Delete marks the role as deleted.
func (r *Role) Delete() {
	now := time.Now()
	r.Metadata.DeletedAt = &now
}

// Restore removes the deletion mark from the role.
func (r *Role) Restore() {
	r.Metadata.DeletedAt = nil
}

// AddPermission adds a permission to the role.
func (r *Role) AddPermission(permissionID string) {
	if slices.Contains(r.Spec.Permissions, permissionID) {
		return
	}

	r.Spec.Permissions = append(r.Spec.Permissions, permissionID)
}

// RemovePermission removes a permission from the role.
func (r *Role) RemovePermission(permissionID string) {
	for i, p := range r.Spec.Permissions {
		if p == permissionID {
			r.Spec.Permissions = append(r.Spec.Permissions[:i], r.Spec.Permissions[i+1:]...)

			return
		}
	}
}

// HasPermission checks if the role has a permission.
func (r *Role) HasPermission(permissionID string) bool {
	return slices.Contains(r.Spec.Permissions, permissionID)
}
