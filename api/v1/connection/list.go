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
	Type               string    `json:"type"`
	LastCommunicatedAt time.Time `json:"lastCommunicatedAt"`
	Alive              bool      `json:"alive"`
} // @name Connection

// ListResponse represents a list of connections with metadata.
type ListResponse = v1.ListResponse[Connection]

// NewListResponse creates a new ListResponse with the given connections and metadata.
func NewListResponse(connections []Connection, metadata v1.ListMeta) *ListResponse {
	return &ListResponse{
		Kind:       Kind,
		APIVersion: v1.APIVersion,
		Metadata:   metadata,
		Items:      connections,
	}
}
