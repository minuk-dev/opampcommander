package agent

// Command is a common struct that represents a command to be sent to an agent.
type Command struct {
	Kind              string         `json:"kind"`
	ID                string         `json:"id"`
	TargetInstanceUID string         `json:"targetInstanceUid"`
	Data              map[string]any `json:"data"`
}
