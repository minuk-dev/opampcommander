package usermodel

import (
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Permission represents a permission in the system.
type Permission struct {
	Metadata PermissionMetadata
	Spec     PermissionSpec
	Status   PermissionStatus
}

// PermissionMetadata contains metadata about the permission.
type PermissionMetadata struct {
	UID       uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// PermissionSpec defines the permission details.
type PermissionSpec struct {
	Name        string // e.g., "agent:read", "agent:write"
	Description string
	Resource    string // e.g., "agent", "agentgroup", "certificate"
	Action      string // e.g., "read", "write", "delete", "execute"
	IsBuiltIn   bool
}

// PermissionStatus represents the current state of the permission.
type PermissionStatus struct {
	Conditions []model.Condition
}

// NewPermission creates a new permission with the given resource and action.
func NewPermission(resource, action string, isBuiltIn bool) *Permission {
	now := time.Now()

	return &Permission{
		Metadata: PermissionMetadata{
			UID:       uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
		},
		Spec: PermissionSpec{
			Name:        resource + ":" + action,
			Description: "",
			Resource:    resource,
			Action:      action,
			IsBuiltIn:   isBuiltIn,
		},
		Status: PermissionStatus{
			Conditions: []model.Condition{},
		},
	}
}

// IsDeleted returns whether the permission is deleted.
func (p *Permission) IsDeleted() bool {
	return p.Metadata.DeletedAt != nil
}

// Delete marks the permission as deleted.
func (p *Permission) Delete() {
	now := time.Now()
	p.Metadata.DeletedAt = &now
}

// Restore removes the deletion mark from the permission.
func (p *Permission) Restore() {
	p.Metadata.DeletedAt = nil
}
