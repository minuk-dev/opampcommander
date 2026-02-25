package model

import "time"

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
	Attributes Attributes
	CreatedAt  *time.Time
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
	Conditions []Condition
}

// IsDeleted returns true if the agent remote config is marked as deleted.
func (arc *AgentRemoteConfig) IsDeleted() bool {
	return arc.Metadata.DeletedAt != nil
}

// MarkDeleted marks the agent remote config as deleted by adding a deleted condition.
func (arc *AgentRemoteConfig) MarkDeleted(deletedAt time.Time, deletedBy string) {
	arc.Metadata.DeletedAt = &deletedAt
	arc.Status.Conditions = append(arc.Status.Conditions, Condition{
		Type:               ConditionTypeDeleted,
		Status:             ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Agent remote config deleted",
	})
}
