package agentmodel

// ServerID is a unique identifier for an API server instance.
type ServerID string

// String returns the string representation of the ServerID.
func (s ServerID) String() string {
	return string(s)
}
