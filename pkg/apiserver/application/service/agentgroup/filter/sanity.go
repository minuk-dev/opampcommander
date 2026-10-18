// Package filter provides filters for agentgroup domain model.
package filter

import (
	"slices"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// Sanity provides methods to sanitize AgentGroup during update operations.
// It preserves immutable fields (like CreatedAt) from the existing model to the new model.
type Sanity struct{}

// NewSanity creates a new instance of Sanity filter.
func NewSanity() *Sanity {
	return &Sanity{}
}

// Sanitize preserves immutable fields from the existing AgentGroup to the updated one.
// Immutable fields: Metadata.CreatedAt, Status.Conditions (preserved, then updated condition appended by caller).
func (f *Sanity) Sanitize(
	existing *agentmodel.AgentGroup,
	updated *agentmodel.AgentGroup,
) *agentmodel.AgentGroup {
	if existing == nil || updated == nil {
		return updated
	}

	// Preserve immutable metadata fields
	updated.Metadata.Name = existing.Metadata.Name
	updated.Metadata.ResourceVersion = existing.Metadata.ResourceVersion
	updated.Metadata.DeletedAt = existing.Metadata.DeletedAt
	updated.Status = existing.Status
	updated.Metadata.Namespace = existing.Metadata.Namespace
	updated.Metadata.CreatedAt = existing.Metadata.CreatedAt

	// Preserve existing conditions (caller will append Updated condition).
	// Clone the slice to keep the existing model immutable.
	updated.Status.Conditions = slices.Clone(existing.Status.Conditions)

	return updated
}
