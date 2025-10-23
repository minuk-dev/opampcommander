package entity

import (
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
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
	CreatedAt   time.Time         `bson:"createdAt"`
	CreatedBy   string            `bson:"createdBy"`
	DeletedAt   *time.Time        `bson:"deletedAt,omitempty"`
	DeletedBy   *string           `bson:"deletedBy,omitempty"`
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

// ToDomain converts the AgentGroup entity to the domain model.
func (e *AgentGroup) ToDomain() *agentgroup.AgentGroup {
	var agentConfig *agentgroup.AgentConfig
	if e.AgentConfig != nil {
		agentConfig = &agentgroup.AgentConfig{
			Value: e.AgentConfig.Value,
		}
	}

	return &agentgroup.AgentGroup{
		Name:       e.Name,
		Attributes: e.Attributes,
		Priority:   e.Priority,
		Selector: model.AgentSelector{
			IdentifyingAttributes:    e.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: e.Selector.NonIdentifyingAttributes,
		},
		AgentConfig: agentConfig,
		CreatedAt:   e.CreatedAt,
		CreatedBy:   e.CreatedBy,
		DeletedAt:   e.DeletedAt,
		DeletedBy:   e.DeletedBy,
	}
}

// AgentGroupFromDomain converts the AgentGroup domain model to the entity representation.
func AgentGroupFromDomain(agentgroup *agentgroup.AgentGroup) *AgentGroup {
	var agentConfig *AgentConfig
	if agentgroup.AgentConfig != nil {
		agentConfig = &AgentConfig{
			Value: agentgroup.AgentConfig.Value,
		}
	}

	return &AgentGroup{
		Common: Common{
			Version: VersionV1,
			ID:      nil, // ID will be set by MongoDB
		},
		Name:       agentgroup.Name,
		Attributes: agentgroup.Attributes,
		Priority:   agentgroup.Priority,
		Selector: AgentSelector{
			IdentifyingAttributes:    agentgroup.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: agentgroup.Selector.NonIdentifyingAttributes,
		},
		AgentConfig: agentConfig,
		CreatedAt:   agentgroup.CreatedAt,
		CreatedBy:   agentgroup.CreatedBy,
		DeletedAt:   agentgroup.DeletedAt,
		DeletedBy:   agentgroup.DeletedBy,
	}
}
