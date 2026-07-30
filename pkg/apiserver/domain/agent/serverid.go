package agentmodel

// ServerID is a unique identifier for an API server instance.
type ServerID string

// String returns the string representation of the ServerID.
func (s ServerID) String() string {
	return string(s)
}

// ServerAddress is the address peers dial to reach this server instance over the
// direct transport. It is empty when direct delivery is not in use, and is published
// into the server registry by the heartbeat so peers can route targeted messages here.
type ServerAddress string

// String returns the string representation of the ServerAddress.
func (s ServerAddress) String() string {
	return string(s)
}
