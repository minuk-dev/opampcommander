// Package command provides the command api model for opampcommander.
package command

// Audit is a common struct that represents a command to be sent to an agent.
type Audit struct {
	Kind              string         `json:"kind"`
	ID                string         `json:"id"`
	TargetInstanceUID string         `json:"targetInstanceUid"`
	Data              map[string]any `json:"data"`
} // @name CommandAudit
