// Package filter provides filters for agentremoteconfig domain model.
package filter

import (
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Sanity provides methods to sanitize AgentRemoteConfig during update operations.
type Sanity struct{}

// NewSanity creates a new instance of Sanity filter.
func NewSanity() *Sanity {
	return &Sanity{}
}

// Sanitize preserves immutable fields from the existing AgentRemoteConfig to the updated one.
func (f *Sanity) Sanitize(
	existing *agentmodel.AgentRemoteConfig,
	updated *agentmodel.AgentRemoteConfig,
) *agentmodel.AgentRemoteConfig {
	if existing == nil || updated == nil {
		return updated
	}

	// Preserve immutable metadata fields
	updated.Metadata.CreatedAt = existing.Metadata.CreatedAt

	// Preserve existing status but avoid sharing mutable slices.
	updated.Status = existing.Status
	if existing.Status.Conditions != nil {
		clonedConditions := make(
			[]model.Condition, len(existing.Status.Conditions),
		)
		copy(clonedConditions, existing.Status.Conditions)
		updated.Status.Conditions = clonedConditions
	}

	return updated
}
