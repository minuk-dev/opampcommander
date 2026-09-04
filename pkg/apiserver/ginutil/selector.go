package ginutil

import (
	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// Query parameter names for server-side filtering, shared by every list endpoint.
const (
	// LabelSelectorParam filters on metadata an operator sets and can edit —
	// metadata.labels, or metadata.attributes on the aggregates that call the same
	// thing attributes.
	LabelSelectorParam = "labelSelector"
	// AttributeSelectorParam filters on metadata the resource itself reports. Only
	// agents have any: their OpAMP AgentDescription, which an operator cannot set.
	AttributeSelectorParam = "attributeSelector"
	// FieldSelectorParam filters on an allowlisted set of the resource's own fields.
	FieldSelectorParam = "fieldSelector"
	// NameParam filters on a case-sensitive prefix of the resource's name.
	NameParam = "name"
	// NameContainsParam filters on a case-insensitive substring of the resource's
	// name. It is a scan rather than an index range scan; see [Selectors].
	NameContainsParam = "nameContains"
)

// MetadataSelector names which of the two metadata selectors a resource answers.
//
// The two are not interchangeable and the distinction is the point: a label is
// something an operator attaches and can change through this API, while an
// attribute is something the resource reported about itself. Selecting on one
// when a resource only has the other is a mistake worth an error, not a filter
// silently applied to the wrong map.
type MetadataSelector int

const (
	// NoMetadataSelector is a resource with neither — a role, say.
	NoMetadataSelector MetadataSelector = iota
	// LabelMetadataSelector is a resource whose metadata an operator sets.
	LabelMetadataSelector
	// AttributeMetadataSelector is a resource whose metadata it reports itself.
	AttributeMetadataSelector
)

// param returns the query parameter this kind of resource answers, and the one it
// does not. Both are empty for a resource that answers neither.
func (m MetadataSelector) param() (string, string) {
	switch m {
	case LabelMetadataSelector:
		return LabelSelectorParam, AttributeSelectorParam
	case AttributeMetadataSelector:
		return AttributeSelectorParam, LabelSelectorParam
	case NoMetadataSelector:
		return "", ""
	default:
		return "", ""
	}
}

// reason explains, for a 400, what the resource does support instead.
func (m MetadataSelector) reason() string {
	switch m {
	case LabelMetadataSelector:
		return "this resource is selected on labels an operator sets; use " + LabelSelectorParam
	case AttributeMetadataSelector:
		return "this resource is selected on the attributes it reports; use " + AttributeSelectorParam
	case NoMetadataSelector:
		return "this resource has no labels or attributes to select on"
	default:
		return "this resource has no labels or attributes to select on"
	}
}

// Selectors holds the parsed server-side filters of a list request.
type Selectors struct {
	// Metadata is the parsed labelSelector or attributeSelector — whichever the
	// resource answers — empty when the client sent none. Both share one grammar;
	// only which metadata they read differs.
	Metadata selector.LabelSelector
	// Field is the parsed fieldSelector, already validated against the listed
	// resource's supported fields.
	Field selector.FieldSelector
	// NamePrefix is the raw name query, empty when the client sent none. It is
	// served by an index range scan.
	NamePrefix string
	// NameContains is the raw nameContains query, empty when the client sent none.
	// No ordered index can answer "contains", so it is a scan; it is a separate
	// parameter so the prefix search stays the default fast path.
	NameContains string
}

// ParseSelectors reads the metadata, field and name query parameters, validating
// the field selector against allowedFields — the fields the resource being listed
// documents as selectable — and the metadata selector against metadata, which
// says whether this resource carries labels, attributes, or neither.
//
// On a malformed selector, one naming a field outside allowedFields, or one using
// the metadata parameter this resource does not have, it writes a 400 naming the
// offending parameter and returns false; the caller must simply return.
//
// Rejecting rather than ignoring is the point, and it is why the unsupported
// parameter is an error rather than an unread query value: gin ignores a
// parameter nothing reads, so a client asking to narrow a list the wrong way
// would otherwise be handed the whole one with a 200.
func ParseSelectors(ctx *gin.Context, metadata MetadataSelector, allowedFields []string) (Selectors, bool) {
	//exhaustruct:ignore
	var zero Selectors

	supported, unsupported := metadata.param()

	if unsupported != "" {
		if raw := ctx.Query(unsupported); raw != "" {
			InvalidQueryParamError(ctx, unsupported, raw, metadata.reason())

			return zero, false
		}
	}

	var metadataSelector selector.LabelSelector

	if supported == "" {
		// A resource with no metadata of either kind rejects both parameters.
		for _, param := range []string{LabelSelectorParam, AttributeSelectorParam} {
			if raw := ctx.Query(param); raw != "" {
				InvalidQueryParamError(ctx, param, raw, metadata.reason())

				return zero, false
			}
		}
	} else {
		raw := ctx.Query(supported)

		parsed, err := selector.ParseLabels(raw)
		if err != nil {
			InvalidQueryParamError(ctx, supported, raw, err.Error())

			return zero, false
		}

		metadataSelector = parsed
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
		Metadata:     metadataSelector,
		Field:        fieldSelector,
		NamePrefix:   ctx.Query(NameParam),
		NameContains: ctx.Query(NameContainsParam),
	}, true
}
