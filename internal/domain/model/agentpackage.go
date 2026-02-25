package model

import "time"

// AgentPackage represents an agent package resource.
type AgentPackage struct {
	Metadata AgentPackageMetadata
	Spec     AgentPackageSpec
	Status   AgentPackageStatus
}

// MarkAsCreated marks the agent package as created by setting the CreatedAt timestamp.
func (a *AgentPackage) MarkAsCreated(createdAt time.Time, createdBy string) {
	// Set the CreatedAt timestamp in metadata
	a.Metadata.CreatedAt = createdAt

	a.Status.Conditions = append(a.Status.Conditions, Condition{
		Type:               ConditionTypeCreated,
		Status:             ConditionStatusTrue,
		LastTransitionTime: createdAt,
		Reason:             createdBy,
		Message:            "Agent package created",
	})
}

// MarkAsDeleted marks the agent package as deleted by setting the DeletedAt timestamp.
func (a *AgentPackage) MarkAsDeleted(deletedAt time.Time, deletedBy string) {
	// Set the DeletedAt timestamp in metadata for soft delete filtering
	a.Metadata.DeletedAt = &deletedAt

	// Mark as deleted by adding a condition
	a.Status.Conditions = append(a.Status.Conditions, Condition{
		Type:               ConditionTypeDeleted,
		Status:             ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Agent package deleted",
	})
}

// AgentPackageMetadata represents the metadata of an agent package.
type AgentPackageMetadata struct {
	Name       string
	Attributes Attributes
	// CreatedAt is the timestamp when the agent package was created.
	CreatedAt time.Time
	// DeletedAt is the timestamp when the agent package was soft deleted.
	// If nil, the agent package is not deleted.
	DeletedAt *time.Time
}

// AgentPackageSpec represents the specification of an agent package.
type AgentPackageSpec struct {
	PackageType string
	Version     string
	DownloadURL string
	ContentHash []byte
	Signature   []byte
	Headers     map[string]string
	Hash        []byte
}

// AgentPackageStatus represents the status of an agent package.
type AgentPackageStatus struct {
	Conditions []Condition
}
