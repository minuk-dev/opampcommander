// Package filter provides filters for endpoint domain model.
package filter

import (
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// Sanity provides methods to sanitize Endpoint during update operations.
type Sanity struct{}

// NewSanity creates a new instance of Sanity filter.
func NewSanity() *Sanity {
	return &Sanity{}
}

// Sanitize preserves immutable fields from the existing Endpoint to the updated one.
func (f *Sanity) Sanitize(
	existing *agentmodel.Endpoint,
	updated *agentmodel.Endpoint,
) *agentmodel.Endpoint {
	if existing == nil || updated == nil {
		return updated
	}

	// Preserve immutable identity and metadata fields. Name and Namespace are the
	// resource's identity and must come from the existing record, not the request
	// body (a PUT body may omit them), so an update never forks into a phantom
	// record under a different key.
	updated.Metadata.Name = existing.Metadata.Name
	updated.Metadata.Namespace = existing.Metadata.Namespace
	updated.Metadata.CreatedAt = existing.Metadata.CreatedAt
	updated.Metadata.DeletedAt = existing.Metadata.DeletedAt

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
