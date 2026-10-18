package agentmodel

// AgentSelector defines the criteria for selecting agent.
type AgentSelector struct {
	// IdentifyingAttributes is a map of identifying attributes used to select agents.
	IdentifyingAttributes map[string]string
	// NonIdentifyingAttributes is a map of non-identifying attributes used to select agents.
	NonIdentifyingAttributes map[string]string
}

// Matches reports whether all selector attributes are present and equal on agent.
func (selector AgentSelector) Matches(agent *Agent) bool {
	for key, value := range selector.IdentifyingAttributes {
		agentValue, ok := agent.Metadata.Description.IdentifyingAttributes[key]
		if !ok || agentValue != value {
			return false
		}
	}

	for key, value := range selector.NonIdentifyingAttributes {
		agentValue, ok := agent.Metadata.Description.NonIdentifyingAttributes[key]
		if !ok || agentValue != value {
			return false
		}
	}

	return true
}
