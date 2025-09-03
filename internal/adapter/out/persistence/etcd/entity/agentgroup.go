package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
)

type AgentGroup struct {
	Version    string            `json:"version"`
	UID        string            `json:"uid"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
	Selector   AgentSelector     `json:"selector"`
	CreatedAt  time.Time         `json:"createdAt"`
	CreatedBy  string            `json:"createdBy"`
	DeletedAt  *time.Time        `json:"deletedAt,omitempty"`
	DeletedBy  *string           `json:"deletedBy,omitempty"`
}

type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}

func (e *AgentGroup) ToDomain() *agentgroup.AgentGroup {
	return &agentgroup.AgentGroup{
		Version:    agentgroup.Version(e.Version),
		UID:        uuid.MustParse(e.UID),
		Name:       e.Name,
		Attributes: e.Attributes,
		Selector: agentgroup.AgentSelector{
			IdentifyingAttributes:    e.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: e.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: e.CreatedAt,
		CreatedBy: e.CreatedBy,
		DeletedAt: e.DeletedAt,
		DeletedBy: e.DeletedBy,
	}
}

func AgentGroupFromDomain(ag *agentgroup.AgentGroup) *AgentGroup {
	return &AgentGroup{
		Version:    string(ag.Version),
		UID:        ag.UID.String(),
		Name:       ag.Name,
		Attributes: ag.Attributes,
		Selector: AgentSelector{
			IdentifyingAttributes:    ag.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: ag.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: ag.CreatedAt,
		CreatedBy: ag.CreatedBy,
		DeletedAt: ag.DeletedAt,
		DeletedBy: ag.DeletedBy,
	}
}
