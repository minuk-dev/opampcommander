// Package filter provides filters for agentpackage domain model.
package filter

import (
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Sanity provides methods to sanitize AgentPackage during update operations.
// It preserves immutable fields (like CreatedAt) from the existing model to the new model.
type Sanity struct{}

// NewSanity creates a new instance of Sanity filter.
func NewSanity() *Sanity {
	return &Sanity{}
}

// Sanitize preserves immutable fields from the existing AgentPackage to the updated one.
// Immutable fields: Metadata.CreatedAt, Status (preserved entirely).
func (f *Sanity) Sanitize(
	existing *model.AgentPackage,
	updated *model.AgentPackage,
) *model.AgentPackage {
	if existing == nil || updated == nil {
		return updated
	}

	// Preserve immutable metadata fields
	updated.Metadata.CreatedAt = existing.Metadata.CreatedAt

	// Preserve existing status (struct) but avoid sharing mutable slices.
	updated.Status = existing.Status
	if existing.Status.Conditions != nil {
		clonedConditions := make([]model.Condition, len(existing.Status.Conditions))
		copy(clonedConditions, existing.Status.Conditions)
		updated.Status.Conditions = clonedConditions
	}

	return updated
}
