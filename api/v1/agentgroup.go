package v1

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
	// NumAgents is the total number of agents in the agent group.
	NumAgents int `json:"numAgents"`

	// NumConnectedAgents is the number of connected agents in the agent group.
	NumConnectedAgents int `json:"numConnectedAgents"`

	// NumHealthyAgents is the number of healthy agents in the agent group.
	NumHealthyAgents int `json:"numHealthyAgents"`

	// NumUnhealthyAgents is the number of unhealthy agents in the agent group.
	NumUnhealthyAgents int `json:"numUnhealthyAgents"`

	// NumNotConnectedAgents is the number of not connected agents in the agent group.
	NumNotConnectedAgents int `json:"numNotConnectedAgents"`

	// Conditions is a list of conditions that apply to the agent group.
	Conditions []Condition `json:"conditions"`
} // @name AgentGroupStatus

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
	AgentRemoteConfig  *AgentGroupRemoteConfig `json:"agentRemoteConfig,omitempty"`
	ConnectionSettings *ConnectionSettings     `json:"connectionSettings,omitempty"`
}

// AgentGroupRemoteConfig represents the remote configuration in an agent group.
// @name AgentGroupRemoteConfig.
type AgentGroupRemoteConfig struct {
	// AgentRemoteConfigName is the name of a standalone remote configuration resource.
	AgentRemoteConfigName *string `json:"agentRemoteConfigName,omitempty"`
	// AgentRemoteConfigSpec is the remote configuration to be applied to agents in this group.
	AgentRemoteConfigSpec *AgentRemoteConfigSpec `json:"agentRemoteConfigSpec,omitempty"`
	// AgentRemoteConfigRef is a reference to a standalone remote configuration resource.
	AgentRemoteConfigRef *string `json:"agentRemoteConfigRef,omitempty"`
}
