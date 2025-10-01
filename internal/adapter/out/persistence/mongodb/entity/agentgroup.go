package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
)

// AgentGroup is the etcd entity representation of the AgentGroup domain model.
type AgentGroup struct {
	EntityCommon `bson:",inline"`

	UID        uuid.UUID         `bson:"uid"`
	Attributes map[string]string `bson:"attributes"`
	Selector   AgentSelector     `bson:"selector"`
	CreatedAt  time.Time         `bson:"createdAt"`
	CreatedBy  string            `bson:"createdBy"`
	DeletedAt  *time.Time        `bson:"deletedAt,omitempty"`
	DeletedBy  *string           `bson:"deletedBy,omitempty"`
}

// AgentSelector defines the criteria for selecting agents to be included in the agent group.
type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}

// ToDomain converts the AgentGroup entity to the domain model.
func (e *AgentGroup) ToDomain() *agentgroup.AgentGroup {
	return &agentgroup.AgentGroup{
		UID:        e.UID,
		Name:       e.Name,
		Attributes: e.Attributes,
		Selector: model.AgentSelector{
			IdentifyingAttributes:    e.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: e.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: e.CreatedAt,
		CreatedBy: e.CreatedBy,
		DeletedAt: e.DeletedAt,
		DeletedBy: e.DeletedBy,
	}
}

// AgentGroupFromDomain converts the AgentGroup domain model to the entity representation.
func AgentGroupFromDomain(agentgroup *agentgroup.AgentGroup) *AgentGroup {
	return &AgentGroup{
		EntityCommon: EntityCommon{
			Version: Version1,
			ID:      nil, // ID will be set by MongoDB
			Name:    agentgroup.Name,
		},
		UID:        agentgroup.UID,
		Attributes: agentgroup.Attributes,
		Selector: AgentSelector{
			IdentifyingAttributes:    agentgroup.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: agentgroup.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: agentgroup.CreatedAt,
		CreatedBy: agentgroup.CreatedBy,
		DeletedAt: agentgroup.DeletedAt,
		DeletedBy: agentgroup.DeletedBy,
	}
}
