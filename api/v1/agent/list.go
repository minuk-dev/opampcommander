package agent

import v1 "github.com/minuk-dev/opampcommander/api/v1"

// ListResponse is a struct that represents the response for listing agents.
type ListResponse struct {
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Metadata   v1.ListMeta `json:"metadata"`
	Items      []Agent     `json:"items"`
} // @name AgentListResponse

func NewListResponse(agents []Agent, metadata v1.ListMeta) *ListResponse {
	return &ListResponse{
		Kind:       AgentKind,
		APIVersion: v1.APIVersion,
		Metadata:   metadata,
		Items:      agents,
	}
}
