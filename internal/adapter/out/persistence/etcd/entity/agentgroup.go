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
	CreatedAt  time.Time         `json:"created_at"`
	CreatedBy  string            `json:"created_by"`
	DeletedAt  *time.Time        `json:"deleted_at,omitempty"`
	DeletedBy  *string           `json:"deleted_by,omitempty"`
}

type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifying_attributes"`
	NonIdentifyingAttributes map[string]string `json:"non_identifying_attributes"`
}

func (e *AgentGroup) ToDomain() *agentgroup.AgentGroup {
	ag := &agentgroup.AgentGroup{
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
	return ag
}
