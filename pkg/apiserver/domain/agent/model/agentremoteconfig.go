package agentmodel

import (
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// AgentRemoteConfig represents a standalone remote configuration resource.
// This is different from AgentRemoteConfig in agentgroup.go which is embedded in AgentGroup.
type AgentRemoteConfig struct {
	Metadata AgentRemoteConfigMetadata
	Spec     AgentRemoteConfigSpec
	Status   AgentRemoteConfigResourceStatus
}

// AgentRemoteConfigMetadata contains metadata for the agent remote config resource.
type AgentRemoteConfigMetadata struct {
	Name       string
	Namespace  string
	Attributes Attributes
	CreatedAt  time.Time
	DeletedAt  *time.Time
}

// AgentRemoteConfigSpec contains the specification for the agent remote config resource.
type AgentRemoteConfigSpec struct {
	// Value is the configuration content.
	Value []byte
	// ContentType is the MIME type of the configuration content.
	ContentType string
}

// AgentRemoteConfigResourceStatus contains the status of the agent remote config resource.
type AgentRemoteConfigResourceStatus struct {
	Conditions []model.Condition
}

// IsDeleted returns true if the agent remote config is marked as deleted.
func (arc *AgentRemoteConfig) IsDeleted() bool {
	return arc.Metadata.DeletedAt != nil
}

// MarkAsCreated stamps the creation timestamp and records a Created condition.
func (arc *AgentRemoteConfig) MarkAsCreated(createdAt time.Time, createdBy string) {
	arc.Metadata.CreatedAt = createdAt
	arc.Status.Conditions = append(arc.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeCreated,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: createdAt,
		Reason:             createdBy,
		Message:            "Agent remote config created",
	})
}

// ApplyUpdate copies the mutable fields from incoming into the receiver while
// preserving immutable identity and lifecycle state (Name, Namespace, CreatedAt,
// DeletedAt, and Status conditions). Callers should load the stored config,
// ApplyUpdate the client-supplied one onto it, and persist the receiver — this
// keeps the identity intact and avoids forking a phantom record on update.
func (arc *AgentRemoteConfig) ApplyUpdate(incoming *AgentRemoteConfig) {
	arc.Spec = incoming.Spec
	arc.Metadata.Attributes = incoming.Metadata.Attributes
}

// MarkDeleted marks the agent remote config as deleted by adding a deleted condition.
func (arc *AgentRemoteConfig) MarkDeleted(deletedAt time.Time, deletedBy string) {
	arc.Metadata.DeletedAt = &deletedAt
	arc.Status.Conditions = append(arc.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeDeleted,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Agent remote config deleted",
	})
}
