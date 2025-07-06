package agent

import v1 "github.com/minuk-dev/opampcommander/api/v1"

// ListResponse represents a list of agents with metadata.
type ListResponse = v1.ListResponse[Agent]

// NewListResponse creates a new ListResponse with the given agents and metadata.
func NewListResponse(agents []Agent, metadata v1.ListMeta) *ListResponse {
	return &ListResponse{
		Kind:       AgentKind,
		APIVersion: v1.APIVersion,
		Metadata:   metadata,
		Items:      agents,
	}
}
