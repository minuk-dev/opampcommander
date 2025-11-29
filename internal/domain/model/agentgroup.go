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
}

// AgentGroupSpec represents the specification of an agent group.
type AgentGroupSpec struct {
	// AgentConfig is the remote configuration to be applied to agents in this group.
	AgentConfig *AgentConfig
}

// AgentGroupStatus represents the status of an agent group.
type AgentGroupStatus struct {
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
	// ConditionTypeDeleted represents the condition when the agent group was deleted.
	ConditionTypeDeleted ConditionType = "Deleted"
)

// ConditionStatus represents the status of a condition.
type ConditionStatus string

const (
	// ConditionStatusTrue represents a true condition status.
	ConditionStatusTrue ConditionStatus = "True"
	// ConditionStatusFalse represents a false condition status.
	ConditionStatusFalse ConditionStatus = "False"
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
		},
		Spec: AgentGroupSpec{
			AgentConfig: nil,
		},
		Status: AgentGroupStatus{
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

// AgentConfig represents the remote configuration for agents in the group.
type AgentConfig struct {
	// Value is the configuration content in string format.
	// This can be used directly or as a reference to a configuration.
	Value string
}

// IsDeleted returns true if the agent group is marked as deleted.
func (ag *AgentGroup) IsDeleted() bool {
	for _, condition := range ag.Status.Conditions {
		if condition.Type == ConditionTypeDeleted && condition.Status == ConditionStatusTrue {
			return true
		}
	}

	return false
}

// MarkDeleted marks the agent group as deleted by adding a deleted condition.
func (ag *AgentGroup) MarkDeleted(deletedAt time.Time, deletedBy string) {
	// Check if already marked as deleted
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
