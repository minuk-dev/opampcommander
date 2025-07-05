package connection

import (
	"time"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// Kind is the kind of the connection.
	Kind = "Connection"
)

// Connection represents a connection to an agent.
type Connection struct {
	ID                 uuid.UUID `json:"id"`
	InstanceUID        uuid.UUID `json:"instanceUid"`
	LastCommunicatedAt time.Time `json:"lastCommunicatedAt"`
	Alive              bool      `json:"alive"`
} // @name Connection

// ListResponse is a struct that represents the response for listing connections.
type ListResponse struct {
	Kind       string       `json:"kind"`
	APIVersion string       `json:"apiVersion"`
	Metadata   v1.ListMeta  `json:"metadata"`
	Items      []Connection `json:"items"`
} // @name ConnectionListResponse

// NewListResponse creates a new ListResponse with the given connections and metadata.
func NewListResponse(connections []Connection, metadata v1.ListMeta) *ListResponse {
	return &ListResponse{
		Kind:       Kind,
		APIVersion: v1.APIVersion,
		Metadata:   metadata,
		Items:      connections,
	}
}
