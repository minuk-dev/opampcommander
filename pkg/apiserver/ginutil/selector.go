package ginutil

import (
	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// Query parameter names for server-side filtering, shared by every list endpoint.
const (
	// LabelSelectorParam filters on the resource's user-supplied metadata map.
	LabelSelectorParam = "labelSelector"
	// FieldSelectorParam filters on an allowlisted set of the resource's own fields.
	FieldSelectorParam = "fieldSelector"
	// NameParam filters on a case-sensitive prefix of the resource's name.
	NameParam = "name"
)

// Selectors holds the parsed server-side filters of a list request.
type Selectors struct {
	// Label is the parsed labelSelector, empty when the client sent none.
	Label selector.LabelSelector
	// Field is the parsed fieldSelector, already validated against the listed
	// resource's supported fields.
	Field selector.FieldSelector
	// NamePrefix is the raw name query, empty when the client sent none.
	NamePrefix string
}

// ParseSelectors reads the labelSelector, fieldSelector and name query
// parameters, validating the field selector against allowedFields — the fields
// the resource being listed documents as selectable.
//
// On a malformed selector, or one naming a field outside allowedFields, it writes
// a 400 that names the offending parameter and field and returns false; the caller
// must simply return. Rejecting rather than ignoring is the point: a client that
// asked to narrow a list must never be handed the whole one and mistake it for a
// filtered result.
func ParseSelectors(ctx *gin.Context, allowedFields []string) (Selectors, bool) {
	return parseSelectors(ctx, allowedFields, labelsSupported)
}

// ParseSelectorsWithoutLabels is [ParseSelectors] for a resource that carries no
// label map at all — a role, say. A labelSelector is answered with a 400 saying
// so, rather than with an empty page the client would read as "nothing matches".
func ParseSelectorsWithoutLabels(ctx *gin.Context, allowedFields []string) (Selectors, bool) {
	return parseSelectors(ctx, allowedFields, labelsAbsent)
}

// labelSupport says whether the resource being listed carries a label map.
type labelSupport bool

const (
	labelsSupported labelSupport = true
	labelsAbsent    labelSupport = false
)

func parseSelectors(ctx *gin.Context, allowedFields []string, labels labelSupport) (Selectors, bool) {
	//exhaustruct:ignore
	var zero Selectors

	rawLabel := ctx.Query(LabelSelectorParam)

	if rawLabel != "" && labels == labelsAbsent {
		InvalidQueryParamError(ctx, LabelSelectorParam, rawLabel,
			"this resource has no labels to select on")

		return zero, false
	}

	labelSelector, err := selector.ParseLabels(rawLabel)
	if err != nil {
		InvalidQueryParamError(ctx, LabelSelectorParam, rawLabel, err.Error())

		return zero, false
	}

	rawField := ctx.Query(FieldSelectorParam)

	fieldSelector, err := selector.ParseFields(rawField)
	if err != nil {
		InvalidQueryParamError(ctx, FieldSelectorParam, rawField, err.Error())

		return zero, false
	}

	err = fieldSelector.Validate(allowedFields)
	if err != nil {
		InvalidQueryParamError(ctx, FieldSelectorParam, rawField, err.Error())

		return zero, false
	}

	return Selectors{
		Label:      labelSelector,
		Field:      fieldSelector,
		NamePrefix: ctx.Query(NameParam),
	}, true
}
