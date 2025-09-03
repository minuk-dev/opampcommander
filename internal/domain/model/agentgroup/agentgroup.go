// Package agentgroup defines the AgentGroup model and related types.
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
	// Attributes is a map of attributes associated with the agent group.
	Attributes Attributes
	// Selector is a set of criteria used to select agents for the group.
	Selector AgentSelector
	// CreatedAt is the timestamp when the agent group was created.
	CreatedAt time.Time
	// CreatedBy is the identifier of the user or system that created the agent group.
	CreatedBy string
	// DeletedAt is the timestamp when the agent group was deleted. It is nil if the agent group is not deleted.
	DeletedAt *time.Time
	// DeletedBy is the identifier of the user or system that deleted the agent group. It
	DeletedBy *string
}

// AgentSelector defines the criteria for selecting agents to be included in the agent group.
type AgentSelector struct {
	// IdentifyingAttributes is a map of identifying attributes used to select agents.
	IdentifyingAttributes map[string]string
	// NonIdentifyingAttributes is a map of non-identifying attributes used to select agents.
	NonIdentifyingAttributes map[string]string
}

// IsDeleted returns true if the agent group is marked as deleted.
func (ag *AgentGroup) IsDeleted() bool {
	return ag.DeletedAt != nil
}

// MarkDeleted marks the agent group as deleted by setting the DeletedAt and DeletedBy fields.
func (ag *AgentGroup) MarkDeleted(deletedAt time.Time, deletedBy string) {
	ag.DeletedAt = &deletedAt
	ag.DeletedBy = &deletedBy
}

// New creates a new instance of AgentGroup with the provided name, attributes,
// createdAt timestamp, and createdBy identifier.
func New(
	name string,
	attributes Attributes,
	createdAt time.Time,
	createdBy string,
) *AgentGroup {
	return &AgentGroup{
		Version:    Version1,
		UID:        uuid.New(),
		Name:       name,
		Attributes: attributes,
		CreatedAt:  createdAt,
		CreatedBy:  createdBy,
		DeletedAt:  nil,
		DeletedBy:  nil,
	}
}

// OfAttributes creates an Attributes instance from a map of attributes.
func OfAttributes(attributes map[string]string) Attributes {
	if attributes == nil {
		return nil
	}

	// deep copy the attributes to avoid mutation
	attr := maps.Clone(attributes)

	return attr
}
