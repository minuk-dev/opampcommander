package agentmodel

import (
	"maps"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Attributes represents a map of attributes for the agent group.
type Attributes map[string]string

// AgentGroup represents a group of agents with their associated metadata.
type AgentGroup struct {
	Metadata AgentGroupMetadata
	Spec     AgentGroupSpec
	Status   AgentGroupStatus
}

// NewAgentGroup creates a new instance of AgentGroup with the provided namespace, name, attributes,
// createdAt timestamp, and createdBy identifier.
func NewAgentGroup(
	namespace string,
	name string,
	attributes Attributes,
	createdAt time.Time,
	createdBy string,
) *AgentGroup {
	return &AgentGroup{
		Metadata: AgentGroupMetadata{
			Namespace:  namespace,
			Name:       name,
			Attributes: attributes,
			CreatedAt:  createdAt,
			DeletedAt:  time.Time{},
		},
		Spec: AgentGroupSpec{
			Priority: 0,
			Selector: AgentSelector{
				IdentifyingAttributes:    nil,
				NonIdentifyingAttributes: nil,
			},
			AgentRemoteConfig:     nil,
			AgentRemoteConfigs:    nil,
			AgentConnectionConfig: nil,
		},
		Status: AgentGroupStatus{
			NumAgents:             0,
			NumConnectedAgents:    0,
			NumHealthyAgents:      0,
			NumUnhealthyAgents:    0,
			NumNotConnectedAgents: 0,

			Conditions: []model.Condition{
				{
					Type:               model.ConditionTypeCreated,
					LastTransitionTime: createdAt,
					Status:             model.ConditionStatusTrue,
					Reason:             createdBy,
					Message:            "Agent group created",
				},
			},
		},
	}
}

// HasAgentConnectionConfig returns true if the agent group has connection configuration.
func (ag *AgentGroup) HasAgentConnectionConfig() bool {
	return ag.Spec.AgentConnectionConfig != nil
}

// AgentGroupMetadata represents metadata information for an agent group.
type AgentGroupMetadata struct {
	// Namespace is the namespace of the agent group.
	// Together with Name, it forms the unique identity of the agent group.
	Namespace string
	// Name is the name of the agent group.
	Name string
	// Attributes is a map of attributes associated with the agent group.
	Attributes Attributes
	// CreatedAt is the timestamp when the agent group was created.
	CreatedAt time.Time
	// DeletedAt is the timestamp when the agent group was soft deleted.
	// If is zero, the agent group is not deleted.
	DeletedAt time.Time
}

// AgentGroupSpec represents the specification of an agent group.
type AgentGroupSpec struct {
	// Priority is the priority of the agent group.
	// When multiple agent groups match an agent, the one with the highest priority is applied.
	Priority int

	// Selector is a set of criteria used to select agents for the group.
	Selector AgentSelector

	// AgentRemoteConfig is a single remote configuration (for API compatibility).
	//
	// Deprecated: Use AgentRemoteConfigs for multiple configs.
	AgentRemoteConfig *AgentGroupAgentRemoteConfig

	// AgentRemoteConfigs is a list of remote configurations for the agent group.
	AgentRemoteConfigs []AgentGroupAgentRemoteConfig

	// AgentConnection settings for agents in this group.
	AgentConnectionConfig *AgentGroupConnectionConfig
}

// AgentGroupAgentRemoteConfig represents a remote configuration for agents in the group.
type AgentGroupAgentRemoteConfig struct {
	// AgentRemoteConfigName is the name of a standalone remote configuration resource.
	AgentRemoteConfigName *string
	// AgentRemoteConfigSpec is the remote configuration to be applied to agents in this group.
	AgentRemoteConfigSpec *AgentRemoteConfigSpec

	// AgentRemoteConfigRef is a reference to a standalone remote configuration resource.
	AgentRemoteConfigRef *string
}

// AgentGroupConnectionConfig represents connection settings for agents in the group.
type AgentGroupConnectionConfig struct {
	OpAMPConnection  *OpAMPConnectionSettings
	OwnMetrics       *TelemetryConnectionSettings
	OwnLogs          *TelemetryConnectionSettings
	OwnTraces        *TelemetryConnectionSettings
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
	Conditions []model.Condition
}

// IsDeleted returns true if the agent group is marked as deleted.
func (ag *AgentGroup) IsDeleted() bool {
	// Check deletedAt field first (new approach)
	if !ag.Metadata.DeletedAt.IsZero() {
		return true
	}

	// Fallback to condition-based check for backward compatibility
	for _, condition := range ag.Status.Conditions {
		if condition.Type == model.ConditionTypeDeleted && condition.Status == model.ConditionStatusTrue {
			return true
		}
	}

	return false
}

// MarkDeleted marks the agent group as deleted by setting deletedAt and adding a deleted condition.
func (ag *AgentGroup) MarkDeleted(deletedAt time.Time, deletedBy string) {
	// Set deletedAt field (new approach)
	ag.Metadata.DeletedAt = deletedAt

	// Also maintain condition for backward compatibility
	for i, condition := range ag.Status.Conditions {
		if condition.Type == model.ConditionTypeDeleted {
			// Update existing deleted condition
			ag.Status.Conditions[i].LastTransitionTime = deletedAt
			ag.Status.Conditions[i].Status = model.ConditionStatusTrue
			ag.Status.Conditions[i].Reason = deletedBy

			return
		}
	}

	// Add new deleted condition
	ag.Status.Conditions = append(ag.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeDeleted,
		LastTransitionTime: deletedAt,
		Status:             model.ConditionStatusTrue,
		Reason:             deletedBy,
		Message:            "Agent group deleted",
	})
}

// GetCreatedAt returns the timestamp when the agent group was created.
func (ag *AgentGroup) GetCreatedAt() *time.Time {
	for _, condition := range ag.Status.Conditions {
		if condition.Type == model.ConditionTypeCreated && condition.Status == model.ConditionStatusTrue {
			return &condition.LastTransitionTime
		}
	}

	return nil
}

// GetCreatedBy returns the identifier of the user or system that created the agent group.
func (ag *AgentGroup) GetCreatedBy() string {
	for _, condition := range ag.Status.Conditions {
		if condition.Type == model.ConditionTypeCreated && condition.Status == model.ConditionStatusTrue {
			return condition.Reason
		}
	}

	return ""
}

// GetDeletedAt returns the timestamp when the agent group was deleted.
func (ag *AgentGroup) GetDeletedAt() *time.Time {
	// Check deletedAt field first (new approach)
	if !ag.Metadata.DeletedAt.IsZero() {
		return &ag.Metadata.DeletedAt
	}

	// Fallback to condition-based check for backward compatibility
	for _, condition := range ag.Status.Conditions {
		if condition.Type == model.ConditionTypeDeleted && condition.Status == model.ConditionStatusTrue {
			return &condition.LastTransitionTime
		}
	}

	return nil
}

// GetDeletedBy returns the identifier of the user or system that deleted the agent group.
func (ag *AgentGroup) GetDeletedBy() *string {
	for _, condition := range ag.Status.Conditions {
		if condition.Type == model.ConditionTypeDeleted && condition.Status == model.ConditionStatusTrue {
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
