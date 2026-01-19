package v1

const (
	// ServerKind is the kind of the server.
	ServerKind = "Server"
)

// Server represents an API server instance.
type Server struct {
	ID              string            `json:"id"`
	LastHeartbeatAt Time              `json:"lastHeartbeatAt"`
	Conditions      []ServerCondition `json:"conditions"`
} // @name Server

// ServerCondition represents a condition of a server.
type ServerCondition struct {
	Type               ServerConditionType   `json:"type"`
	LastTransitionTime Time                  `json:"lastTransitionTime"`
	Status             ServerConditionStatus `json:"status"`
	Reason             string                `json:"reason"`
	Message            string                `json:"message,omitempty"`
} // @name ServerCondition

// ServerConditionType represents the type of a server condition.
type ServerConditionType string // @name ServerConditionType

const (
	// ServerConditionTypeRegistered represents the condition when the server was registered.
	ServerConditionTypeRegistered ServerConditionType = "Registered"
	// ServerConditionTypeAlive represents the condition when the server is alive.
	ServerConditionTypeAlive ServerConditionType = "Alive"
)

// ServerConditionStatus represents the status of a server condition.
type ServerConditionStatus string // @name ServerConditionStatus

const (
	// ServerConditionStatusTrue represents a true condition status.
	ServerConditionStatusTrue ServerConditionStatus = "True"
	// ServerConditionStatusFalse represents a false condition status.
	ServerConditionStatusFalse ServerConditionStatus = "False"
	// ServerConditionStatusUnknown represents an unknown condition status.
	ServerConditionStatusUnknown ServerConditionStatus = "Unknown"
)

// ServerListResponse represents a list of servers with metadata.
type ServerListResponse = ListResponse[Server]

// NewServerListResponse creates a new ServerListResponse with the given servers and metadata.
func NewServerListResponse(servers []Server, metadata ListMeta) *ServerListResponse {
	return &ServerListResponse{
		Kind:       ServerKind,
		APIVersion: APIVersion,
		Metadata:   metadata,
		Items:      servers,
	}
}
