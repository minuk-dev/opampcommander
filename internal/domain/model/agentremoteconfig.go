package model

import "time"

// AgentRemoteConfigResource represents a standalone remote configuration resource.
// This is different from AgentRemoteConfig in agentgroup.go which is embedded in AgentGroup.
type AgentRemoteConfigResource struct {
	Metadata AgentRemoteConfigMetadata
	Spec     AgentRemoteConfigSpec
	Status   AgentRemoteConfigResourceStatus
}

// AgentRemoteConfigMetadata contains metadata for the agent remote config resource.
type AgentRemoteConfigMetadata struct {
	Name       string
	Attributes Attributes
	CreatedAt  time.Time
	CreatedBy  string
	UpdatedAt  time.Time
	UpdatedBy  string
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
func (arc *AgentRemoteConfigResource) IsDeleted() bool {
	for _, condition := range arc.Status.Conditions {
		if condition.Type == ConditionTypeDeleted && condition.Status == ConditionStatusTrue {
			return true
		}
	}

	return false
}

// MarkDeleted marks the agent remote config as deleted by adding a deleted condition.
func (arc *AgentRemoteConfigResource) MarkDeleted(deletedAt time.Time, deletedBy string) {
	for i, condition := range arc.Status.Conditions {
		if condition.Type == ConditionTypeDeleted {
			arc.Status.Conditions[i].Status = ConditionStatusTrue
			arc.Status.Conditions[i].LastTransitionTime = deletedAt
			arc.Status.Conditions[i].Reason = deletedBy
			arc.Status.Conditions[i].Message = "Deleted by " + deletedBy

			return
		}
	}

	arc.Status.Conditions = append(arc.Status.Conditions, Condition{
		Type:               ConditionTypeDeleted,
		Status:             ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Deleted by " + deletedBy,
	})
}
