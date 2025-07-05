// Package command provides the command api model for opampcommander.
package command

import v1 "github.com/minuk-dev/opampcommander/api/v1"

const (
	// CommandKind is the kind of the command.
	CommandKind = "CommandAudit"
)

// Audit is a common struct that represents a command to be sent to an agent.
type Audit struct {
	Kind              string         `json:"kind"`
	ID                string         `json:"id"`
	TargetInstanceUID string         `json:"targetInstanceUid"`
	Data              map[string]any `json:"data"`
} // @name CommandAudit

// ListResponse is a struct that represents the response for listing commands.
type ListResponse struct {
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Metadata   v1.ListMeta `json:"metadata"`
	Items      []Audit     `json:"items"`
} // @name CommandAuditListResponse

// NewListResponse creates a new ListResponse with the given commands and metadata.
func NewListResponse(commands []Audit, metadata v1.ListMeta) *ListResponse {
	return &ListResponse{
		Kind:       CommandKind,
		APIVersion: v1.APIVersion,
		Metadata:   metadata,
		Items:      commands,
	}
}
