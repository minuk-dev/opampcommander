package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
)

// AgentGroup is the etcd entity representation of the AgentGroup domain model.
type AgentGroup struct {
	Version    int               `json:"version"`
	UID        string            `json:"uid"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
	Selector   AgentSelector     `json:"selector"`
	CreatedAt  time.Time         `json:"createdAt"`
	CreatedBy  string            `json:"createdBy"`
	DeletedAt  *time.Time        `json:"deletedAt,omitempty"`
	DeletedBy  *string           `json:"deletedBy,omitempty"`
}

// AgentSelector defines the criteria for selecting agents to be included in the agent group.
type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}

// ToDomain converts the AgentGroup entity to the domain model.
func (e *AgentGroup) ToDomain() *agentgroup.AgentGroup {
	return &agentgroup.AgentGroup{
		UID:        uuid.MustParse(e.UID),
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
		Version:    VersionV1,
		UID:        agentgroup.UID.String(),
		Name:       agentgroup.Name,
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
