package model

// AgentSelector defines the criteria for selecting agent.
type AgentSelector struct {
	// IdentifyingAttributes is a map of identifying attributes used to select agents.
	IdentifyingAttributes map[string]string
	// NonIdentifyingAttributes is a map of non-identifying attributes used to select agents.
	NonIdentifyingAttributes map[string]string
}
