package agentgroup

import (
	"maps"
	"time"

	"github.com/google/uuid"
)

const (
	// Version1 is the initial version of the agent group.
	Version1 = "1"
)

// Version represents the version of the agent group.
type Version string

// Attributes represents a map of attributes for the agent group.
type Attributes map[string]string

// AgentGroup represents a group of agents with their associated metadata.
type AgentGroup struct {
	// Version is the version of the agent group.
	Version Version
	// UID is the unique identifier for the agent group.
	UID uuid.UUID
	// Name is the name of the agent group.
	Name string
	// Description is a brief description of the agent group.
	Attributes Attributes
	// CreatedAt is the timestamp when the agent group was created.
	CreatedAt time.Time
	// CreatedBy is the identifier of the user or system that created the agent group.
	CreatedBy string
}

// New creates a new instance of AgentGroup with the provided name, attributes, createdAt timestamp, and createdBy identifier.
func New(name string, attributes Attributes, createdAt time.Time, createdBy string) *AgentGroup {
	return &AgentGroup{
		Version:    Version1,
		UID:        uuid.New(),
		Name:       name,
		Attributes: attributes,
		CreatedAt:  createdAt,
		CreatedBy:  createdBy,
	}
}

// OfVersion returns a new AgentGroup with the specified version.
func OfAttributes(attributes map[string]string) Attributes {
	if attributes == nil {
		return nil
	}

	// deep copy the attributes to avoid mutation
	attr := maps.Clone(attributes)
	return attr
}
