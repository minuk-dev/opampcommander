// Package agentgroup provides the agentgroup API for the server
package agentgroup

import (
	"time"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

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
	CreatedAt  time.Time     `json:"createdAt"`
	CreatedBy  string        `json:"createdBy"`
	DeletedAt  *time.Time    `json:"deletedAt,omitempty"`
	DeletedBy  *string       `json:"deletedBy,omitempty"`
} // @name AgentGroup

// ListResponse represents a response for listing agent groups.
type ListResponse = v1.ListResponse[AgentGroup]

// CreateRequest represents a request to create an agent group.
type CreateRequest struct {
	Name       string        `binding:"required" json:"name"`
	Attributes Attributes    `json:"attributes"`
	Selector   AgentSelector `json:"selector"`
} // @name AgentGroupCreateRequest

// Attributes represents a map of attributes for the agent group.
// @name AgentGroupAttributes.
type Attributes map[string]string

// AgentSelector defines the criteria for selecting agents to be included in the agent group.
// @name AgentGroupAgentSelector.
type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}
