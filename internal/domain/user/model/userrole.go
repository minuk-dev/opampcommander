package usermodel

import (
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// UserRole represents the assignment of a role to a user.
type UserRole struct {
	Metadata UserRoleMetadata
	Spec     UserRoleSpec
	Status   UserRoleStatus
}

// UserRoleMetadata contains metadata about the user role assignment.
type UserRoleMetadata struct {
	UID       uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// UserRoleSpec defines the user role assignment details.
type UserRoleSpec struct {
	UserID     uuid.UUID
	RoleID     uuid.UUID
	AssignedAt time.Time
	AssignedBy uuid.UUID // User who assigned the role
}

// UserRoleStatus represents the current state of the user role assignment.
type UserRoleStatus struct {
	Conditions []model.Condition
}

// NewUserRole creates a new user role assignment.
func NewUserRole(userID, roleID, assignedBy uuid.UUID) *UserRole {
	now := time.Now()

	return &UserRole{
		Metadata: UserRoleMetadata{
			UID:       uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
		},
		Spec: UserRoleSpec{
			UserID:     userID,
			RoleID:     roleID,
			AssignedAt: now,
			AssignedBy: assignedBy,
		},
		Status: UserRoleStatus{
			Conditions: []model.Condition{},
		},
	}
}

// IsDeleted returns whether the user role assignment is deleted.
func (ur *UserRole) IsDeleted() bool {
	return ur.Metadata.DeletedAt != nil
}

// Delete marks the user role assignment as deleted.
func (ur *UserRole) Delete() {
	now := time.Now()
	ur.Metadata.DeletedAt = &now
}

// Restore removes the deletion mark from the user role assignment.
func (ur *UserRole) Restore() {
	ur.Metadata.DeletedAt = nil
}
