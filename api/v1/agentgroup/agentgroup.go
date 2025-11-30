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
	Metadata Metadata `json:"metadata"`
	Spec     Spec     `json:"spec"`
	Status   Status   `json:"status"`
} // @name AgentGroup

// Metadata represents metadata information for an agent group.
type Metadata struct {
	Name       string        `json:"name"`
	Priority   int           `json:"priority"`
	Attributes Attributes    `json:"attributes"`
	Selector   AgentSelector `json:"selector"`
} // @name AgentGroupMetadata

// Spec represents the specification of an agent group.
type Spec struct {
	AgentConfig *AgentConfig `json:"agentConfig,omitempty"`
} // @name AgentGroupSpec

// Status represents the status of an agent group.
type Status struct {
	Conditions []Condition `json:"conditions"`
} // @name AgentGroupStatus

// Condition represents a condition of an agent group.
type Condition struct {
	Type               ConditionType   `json:"type"`
	LastTransitionTime time.Time       `json:"lastTransitionTime"`
	Status             ConditionStatus `json:"status"`
	Reason             string          `json:"reason"`
	Message            string          `json:"message,omitempty"`
} // @name AgentGroupCondition

// ConditionType represents the type of a condition.
type ConditionType string // @name AgentGroupConditionType

const (
	// ConditionTypeCreated represents the condition when the agent group was created.
	ConditionTypeCreated ConditionType = "Created"
	// ConditionTypeDeleted represents the condition when the agent group was deleted.
	ConditionTypeDeleted ConditionType = "Deleted"
)

// ConditionStatus represents the status of a condition.
type ConditionStatus string // @name AgentGroupConditionStatus

const (
	// ConditionStatusTrue represents a true condition status.
	ConditionStatusTrue ConditionStatus = "True"
	// ConditionStatusFalse represents a false condition status.
	ConditionStatusFalse ConditionStatus = "False"
)

// CreateRequest represents a request to create an agent group.
type CreateRequest struct {
	Name        string        `binding:"required"           json:"name"`
	Priority    int           `json:"priority"`
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
