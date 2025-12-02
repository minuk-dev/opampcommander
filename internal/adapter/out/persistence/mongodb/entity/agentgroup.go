package entity

import (
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// AgentGroupKeyFieldName is the field name used as the key for AgentGroup entities in MongoDB.
	AgentGroupKeyFieldName string = "name"
)

// AgentGroup is the mongo entity representation of the AgentGroup domain model.
type AgentGroup struct {
	Common `bson:",inline"`

	Name        string            `bson:"name"`
	Priority    int               `bson:"priority"`
	Attributes  map[string]string `bson:"attributes"`
	Selector    AgentSelector     `bson:"selector"`
	AgentConfig *AgentConfig      `bson:"agentConfig,omitempty"`
	Conditions  []Condition       `bson:"conditions"`
}

// AgentSelector defines the criteria for selecting agents to be included in the agent group.
type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}

// AgentConfig represents the remote configuration for agents in the group.
type AgentConfig struct {
	Value string `bson:"value" json:"value"`
}

// Condition represents a condition of an agent group in MongoDB.
type Condition struct {
	Type               string    `bson:"type"`
	LastTransitionTime time.Time `bson:"lastTransitionTime"`
	Status             string    `bson:"status"`
	Reason             string    `bson:"reason"`
	Message            string    `bson:"message,omitempty"`
}

// AgentGroupStatistics holds statistical data for an agent group.
type AgentGroupStatistics struct {
	NumAgents             int64
	NumConnectedAgents    int64
	NumHealthyAgents      int64
	NumUnhealthyAgents    int64
	NumNotConnectedAgents int64
}

// ToDomain converts the AgentGroup entity to the domain model.
func (e *AgentGroup) ToDomain(statistics *AgentGroupStatistics) *model.AgentGroup {
	var agentConfig *model.AgentConfig
	if e.AgentConfig != nil {
		agentConfig = &model.AgentConfig{
			Value: e.AgentConfig.Value,
		}
	}

	conditions := make([]model.Condition, len(e.Conditions))
	for i, condition := range e.Conditions {
		conditions[i] = model.Condition{
			Type:               model.ConditionType(condition.Type),
			LastTransitionTime: condition.LastTransitionTime,
			Status:             model.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return &model.AgentGroup{
		Metadata: model.AgentGroupMetadata{
			Name:       e.Name,
			Attributes: e.Attributes,
			Priority:   e.Priority,
			Selector: model.AgentSelector{
				IdentifyingAttributes:    e.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: e.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: model.AgentGroupSpec{
			AgentConfig: agentConfig,
		},
		Status: model.AgentGroupStatus{
			NumAgents:             int(statistics.NumAgents),
			NumHealthyAgents:      int(statistics.NumHealthyAgents),
			NumConnectedAgents:    int(statistics.NumConnectedAgents),
			NumUnhealthyAgents:    int(statistics.NumUnhealthyAgents),
			NumNotConnectedAgents: int(statistics.NumNotConnectedAgents),

			Conditions: conditions,
		},
	}
}

// AgentGroupFromDomain converts the AgentGroup domain model to the entity representation.
func AgentGroupFromDomain(agentgroup *model.AgentGroup) *AgentGroup {
	var agentConfig *AgentConfig
	if agentgroup.Spec.AgentConfig != nil {
		agentConfig = &AgentConfig{
			Value: agentgroup.Spec.AgentConfig.Value,
		}
	}

	conditions := make([]Condition, len(agentgroup.Status.Conditions))
	for i, condition := range agentgroup.Status.Conditions {
		conditions[i] = Condition{
			Type:               string(condition.Type),
			LastTransitionTime: condition.LastTransitionTime,
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return &AgentGroup{
		Common: Common{
			Version: VersionV1,
			ID:      nil, // ID will be set by MongoDB
		},
		Name:       agentgroup.Metadata.Name,
		Attributes: agentgroup.Metadata.Attributes,
		Priority:   agentgroup.Metadata.Priority,
		Selector: AgentSelector{
			IdentifyingAttributes:    agentgroup.Metadata.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: agentgroup.Metadata.Selector.NonIdentifyingAttributes,
		},
		AgentConfig: agentConfig,
		Conditions:  conditions,
	}
}
