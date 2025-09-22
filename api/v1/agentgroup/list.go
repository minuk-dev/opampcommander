package agentgroup

import v1 "github.com/minuk-dev/opampcommander/api/v1"

// ListResponse represents a response for listing agent groups.
type ListResponse = v1.ListResponse[AgentGroup]

// NewListResponse creates a new ListResponse with the given agent groups and metadata.
func NewListResponse(agentGroups []AgentGroup, metadata v1.ListMeta) *ListResponse {
	return &ListResponse{
		Kind:       AgentGroupKind,
		APIVersion: v1.APIVersion,
		Metadata:   metadata,
		Items:      agentGroups,
	}
}
