// Package filter provides filters for certificate domain model.
package filter

import (
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Sanity provides methods to sanitize Certificate during update operations.
// It preserves immutable fields (like CreatedAt) from the existing model to the new model.
type Sanity struct{}

// NewSanity creates a new instance of Sanity filter.
func NewSanity() *Sanity {
	return &Sanity{}
}

// Sanitize preserves immutable fields from the existing Certificate to the updated one.
// Immutable fields: Metadata.CreatedAt, Status (preserved entirely).
func (f *Sanity) Sanitize(
	existing *model.Certificate,
	updated *model.Certificate,
) *model.Certificate {
	if existing == nil || updated == nil {
		return updated
	}

	// Preserve immutable metadata fields
	updated.Metadata.CreatedAt = existing.Metadata.CreatedAt

	// Preserve existing status
	updated.Status = existing.Status

	return updated
}
