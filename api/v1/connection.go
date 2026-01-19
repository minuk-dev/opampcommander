package v1

import "github.com/google/uuid"

const (
	// ConnectionKind is the kind of the connection resource.
	ConnectionKind = "Connection"
)

// Connection represents a connection to an agent.
// It follows the Kubernetes-style resource structure.
type Connection struct {
	// ID is the unique identifier of the connection.
	ID uuid.UUID `json:"id"`

	// InstanceUID is the unique identifier of the agent instance.
	InstanceUID uuid.UUID `json:"instanceUid"`

	// Type is the type of connection (e.g., "http", "websocket").
	Type string `json:"type"`

	// LastCommunicatedAt is the timestamp of the last communication with the agent.
	LastCommunicatedAt Time `json:"lastCommunicatedAt"`

	// Alive indicates whether the connection is currently alive.
	Alive bool `json:"alive"`
} // @name Connection

// ConnectionListResponse represents a list of connections with metadata.
type ConnectionListResponse = ListResponse[Connection]

// NewConnectionListResponse creates a new ConnectionListResponse with the given connections and metadata.
func NewConnectionListResponse(connections []Connection, metadata ListMeta) *ConnectionListResponse {
	return &ConnectionListResponse{
		Kind:       ConnectionKind,
		APIVersion: APIVersion,
		Metadata:   metadata,
		Items:      connections,
	}
}
