// Package agentgroup defines the AgentGroup model and related types.
package agentgroup

import (
	"maps"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Attributes represents a map of attributes for the agent group.
type Attributes map[string]string

// AgentGroup represents a group of agents with their associated metadata.
type AgentGroup struct {
	// Name is the name of the agent group.
	Name string
	// Attributes is a map of attributes associated with the agent group.
	Attributes Attributes
	// Selector is a set of criteria used to select agents for the group.
	Selector model.AgentSelector
	// CreatedAt is the timestamp when the agent group was created.
	CreatedAt time.Time
	// CreatedBy is the identifier of the user or system that created the agent group.
	CreatedBy string
	// DeletedAt is the timestamp when the agent group was deleted. It is nil if the agent group is not deleted.
	DeletedAt *time.Time
	// DeletedBy is the identifier of the user or system that deleted the agent group. It
	DeletedBy *string
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
		Name:       name,
		Attributes: attributes,
		Selector: model.AgentSelector{
			IdentifyingAttributes:    nil,
			NonIdentifyingAttributes: nil,
		},
		CreatedAt: createdAt,
		CreatedBy: createdBy,
		DeletedAt: nil,
		DeletedBy: nil,
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
