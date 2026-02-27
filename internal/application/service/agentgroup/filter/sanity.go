// Package filter provides filters for agentgroup domain model.
package filter

import (
	"github.com/minuk-dev/opampcommander/internal/domain/model"
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
	existing *model.AgentGroup,
	updated *model.AgentGroup,
) *model.AgentGroup {
	if existing == nil || updated == nil {
		return updated
	}

	// Preserve immutable metadata fields
	updated.Metadata.CreatedAt = existing.Metadata.CreatedAt

	// Preserve existing conditions (caller will append Updated condition)
	updated.Status.Conditions = existing.Status.Conditions

	return updated
}
