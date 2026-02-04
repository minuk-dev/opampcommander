package model

import (
	"maps"
	"time"
)

// Attributes represents a map of attributes for the agent group.
type Attributes map[string]string

// AgentGroup represents a group of agents with their associated metadata.
type AgentGroup struct {
	Metadata AgentGroupMetadata
	Spec     AgentGroupSpec
	Status   AgentGroupStatus
}

// AgentGroupMetadata represents metadata information for an agent group.
type AgentGroupMetadata struct {
	// Name is the name of the agent group.
	Name string
	// Priority is the priority of the agent group.
	// When multiple agent groups match an agent, the one with the highest priority is applied.
	Priority int
	// Attributes is a map of attributes associated with the agent group.
	Attributes Attributes
	// Selector is a set of criteria used to select agents for the group.
	Selector AgentSelector
	// DeletedAt is the timestamp when the agent group was soft deleted.
	// If nil, the agent group is not deleted.
	DeletedAt *time.Time
}

// AgentGroupSpec represents the specification of an agent group.
type AgentGroupSpec struct {
	// AgentRemoteConfig is a single remote configuration (for API compatibility).
	// Deprecated: Use AgentRemoteConfigs for multiple configs.
	AgentRemoteConfig *AgentGroupAgentRemoteConfig

	// AgentRemoteConfigs is a list of remote configurations for the agent group.
	AgentRemoteConfigs []AgentGroupAgentRemoteConfig

	// AgentConnection settings for agents in this group.
	AgentConnectionConfig *AgentConnectionConfig
}

type AgentGroupAgentRemoteConfig struct {
	// AgentRemoteConfigName is the name of a standalone remote configuration resource.
	AgentRemoteConfigName *string
	// AgentRemoteConfigSpec is the remote configuration to be applied to agents in this group.
	AgentRemoteConfigSpec *AgentRemoteConfigSpec

	// AgentRemoteConfigRef is a reference to a standalone remote configuration resource.
	AgentRemoteConfigRef *string
}

// AgentConnectionConfig represents connection settings for agents in the group.
type AgentConnectionConfig struct {
	OpAMPConnection  OpAMPConnectionSettings
	OwnMetrics       TelemetryConnectionSettings
	OwnLogs          TelemetryConnectionSettings
	OwnTraces        TelemetryConnectionSettings
	OtherConnections map[string]OtherConnectionSettings
}

// AgentGroupStatus represents the status of an agent group.
type AgentGroupStatus struct {
	// NumAgents is the total number of agents in the agent group.
	// NumAgents = NumConnectedAgents + NumNotConnectedAgents
	NumAgents int

	// NumConnectedAgents is the number of connected agents in the agent group.
	// NumConnectedAgents = NumHealthyAgents + NumUnhealthyAgents
	NumConnectedAgents int

	// NumHealthyAgents is the number of healthy agents in the agent group.
	NumHealthyAgents int

	// NumUnhealthyAgents is the number of unhealthy agents in the agent group.
	NumUnhealthyAgents int

	// NumNotConnectedAgents is the number of not connected agents in the agent group.
	NumNotConnectedAgents int

	// Conditions is a list of conditions that apply to the agent group.
	Conditions []Condition
}

// Condition represents a condition of an agent group.
type Condition struct {
	// Type is the type of the condition.
	Type ConditionType
	// LastTransitionTime is the last time the condition transitioned.
	LastTransitionTime time.Time
	// Status is the status of the condition.
	Status ConditionStatus
	// Reason is the identifier of the user or system that triggered the condition.
	Reason string
	// Message is a human readable message indicating details about the condition.
	Message string
}

// ConditionType represents the type of a condition.
type ConditionType string

const (
	// ConditionTypeCreated represents the condition when the agent group was created.
	ConditionTypeCreated ConditionType = "Created"
	// ConditionTypeUpdated represents the condition when the agent group was updated.
	ConditionTypeUpdated ConditionType = "Updated"
	// ConditionTypeDeleted represents the condition when the agent group was deleted.
	ConditionTypeDeleted ConditionType = "Deleted"
	// ConditionTypeAlive represents the condition when the server is alive.
	ConditionTypeAlive ConditionType = "Alive"
)

// ConditionStatus represents the status of a condition.
type ConditionStatus string

const (
	// ConditionStatusTrue represents a true condition status.
	ConditionStatusTrue ConditionStatus = "True"
	// ConditionStatusFalse represents a false condition status.
	ConditionStatusFalse ConditionStatus = "False"
	// ConditionStatusUnknown represents an unknown condition status.
	ConditionStatusUnknown ConditionStatus = "Unknown"
)

// NewAgentGroup creates a new instance of AgentGroup with the provided name, attributes,
// createdAt timestamp, and createdBy identifier.
func NewAgentGroup(
	name string,
	attributes Attributes,
	createdAt time.Time,
	createdBy string,
) *AgentGroup {
	return &AgentGroup{
		Metadata: AgentGroupMetadata{
			Name:       name,
			Attributes: attributes,
			Priority:   0,
			Selector: AgentSelector{
				IdentifyingAttributes:    nil,
				NonIdentifyingAttributes: nil,
			},
			DeletedAt: nil,
		},
		Spec: AgentGroupSpec{
			AgentRemoteConfigs:    nil,
			AgentConnectionConfig: nil,
		},
		Status: AgentGroupStatus{
			NumAgents:             0,
			NumConnectedAgents:    0,
			NumHealthyAgents:      0,
			NumUnhealthyAgents:    0,
			NumNotConnectedAgents: 0,

			Conditions: []Condition{
				{
					Type:               ConditionTypeCreated,
					LastTransitionTime: createdAt,
					Status:             ConditionStatusTrue,
					Reason:             createdBy,
					Message:            "Agent group created",
				},
			},
		},
	}
}

// IsDeleted returns true if the agent group is marked as deleted.
func (ag *AgentGroup) IsDeleted() bool {
	// Check deletedAt field first (new approach)
	if ag.Metadata.DeletedAt != nil {
		return true
	}

	// Fallback to condition-based check for backward compatibility
	for _, condition := range ag.Status.Conditions {
		if condition.Type == ConditionTypeDeleted && condition.Status == ConditionStatusTrue {
			return true
		}
	}

	return false
}

// MarkDeleted marks the agent group as deleted by setting deletedAt and adding a deleted condition.
func (ag *AgentGroup) MarkDeleted(deletedAt time.Time, deletedBy string) {
	// Set deletedAt field (new approach)
	ag.Metadata.DeletedAt = &deletedAt

	// Also maintain condition for backward compatibility
	for i, condition := range ag.Status.Conditions {
		if condition.Type == ConditionTypeDeleted {
			// Update existing deleted condition
			ag.Status.Conditions[i].LastTransitionTime = deletedAt
			ag.Status.Conditions[i].Status = ConditionStatusTrue
			ag.Status.Conditions[i].Reason = deletedBy

			return
		}
	}

	// Add new deleted condition
	ag.Status.Conditions = append(ag.Status.Conditions, Condition{
		Type:               ConditionTypeDeleted,
		LastTransitionTime: deletedAt,
		Status:             ConditionStatusTrue,
		Reason:             deletedBy,
		Message:            "Agent group deleted",
	})
}

// GetCreatedAt returns the timestamp when the agent group was created.
func (ag *AgentGroup) GetCreatedAt() *time.Time {
	for _, condition := range ag.Status.Conditions {
		if condition.Type == ConditionTypeCreated && condition.Status == ConditionStatusTrue {
			return &condition.LastTransitionTime
		}
	}

	return nil
}

// GetCreatedBy returns the identifier of the user or system that created the agent group.
func (ag *AgentGroup) GetCreatedBy() string {
	for _, condition := range ag.Status.Conditions {
		if condition.Type == ConditionTypeCreated && condition.Status == ConditionStatusTrue {
			return condition.Reason
		}
	}

	return ""
}

// GetDeletedAt returns the timestamp when the agent group was deleted.
func (ag *AgentGroup) GetDeletedAt() *time.Time {
	// Check deletedAt field first (new approach)
	if ag.Metadata.DeletedAt != nil {
		return ag.Metadata.DeletedAt
	}

	// Fallback to condition-based check for backward compatibility
	for _, condition := range ag.Status.Conditions {
		if condition.Type == ConditionTypeDeleted && condition.Status == ConditionStatusTrue {
			return &condition.LastTransitionTime
		}
	}

	return nil
}

// GetDeletedBy returns the identifier of the user or system that deleted the agent group.
func (ag *AgentGroup) GetDeletedBy() *string {
	for _, condition := range ag.Status.Conditions {
		if condition.Type == ConditionTypeDeleted && condition.Status == ConditionStatusTrue {
			reason := condition.Reason

			return &reason
		}
	}

	return nil
}

// OfAttributes creates an Attributes instance from a map of attributes.
func OfAttributes(attributes map[string]string) Attributes {
	if attributes == nil {
		return nil
	}

	// deep copy the attributes to avoid mutation
	attr := maps.Clone(attributes)

	return attr
}
