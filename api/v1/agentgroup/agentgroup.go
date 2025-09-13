// Package agentgroup proviedes the agentgroup API for the server
package agentgroup

import "github.com/google/uuid"

const (
	// AgentGroupKind is the kind of the agent group resource.
	AgentGroupKind = "AgentGroup"
)

// AgentGroup represents a struct that represents an agent group.
type AgentGroup struct {
	UID        uuid.UUID     `json:"uid"`
	Name       string        `json:"name"`
	Attributes Attributes    `json:"attributes"`
	Selector   AgentSelector `json:"selector"`
	CreatedAt  string        `json:"createdAt"`
	CreatedBy  string        `json:"createdBy"`
	DeletedAt  *string       `json:"deletedAt,omitempty"`
	DeletedBy  *string       `json:"deletedBy,omitempty"`
} // @name AgentGroup

// Attributes represents a map of attributes for the agent group.
// @name AgentGroupAttributes.
type Attributes map[string]string

// AgentSelector defines the criteria for selecting agents to be included in the agent group.
// @name AgentGroupAgentSelector.
type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}
