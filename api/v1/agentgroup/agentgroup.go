// Package agentgroup provides the agentgroup API for the server
package agentgroup

import (
	"time"
)

const (
	// AgentGroupKind is the kind of the agent group resource.
	AgentGroupKind = "AgentGroup"
)

// AgentGroup represents a struct that represents an agent group.
type AgentGroup struct {
	Name        string        `json:"name"`
	Attributes  Attributes    `json:"attributes"`
	Selector    AgentSelector `json:"selector"`
	AgentConfig *AgentConfig  `json:"agentConfig,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	CreatedBy   string        `json:"createdBy"`
	DeletedAt   *time.Time    `json:"deletedAt,omitempty"`
	DeletedBy   *string       `json:"deletedBy,omitempty"`
} // @name AgentGroup

// CreateRequest represents a request to create an agent group.
type CreateRequest struct {
	Name        string        `binding:"required"           json:"name"`
	Attributes  Attributes    `json:"attributes"`
	Selector    AgentSelector `json:"selector"`
	AgentConfig *AgentConfig  `json:"agentConfig,omitempty"`
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

// AgentConfig represents the remote configuration for agents in the group.
// @name AgentGroupAgentConfig.
type AgentConfig struct {
	Value string `json:"value"`
}
