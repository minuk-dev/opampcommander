package port

import (
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// ListOptions holds options for listing resources at the application boundary.
//
// It mirrors the domain-level list options but is owned by the application
// layer so that primary adapters (HTTP controllers, messaging consumers) can
// express paging/filtering without importing the domain package. The
// application layer converts it to the domain representation via ToDomain.
type ListOptions struct {
	Limit          int64
	Continue       string
	IncludeDeleted bool

	// LabelSelector, when non-empty, restricts the listing to resources whose
	// user-supplied metadata (metadata.labels, metadata.attributes, or an agent's
	// identifying and non-identifying attributes) satisfies every requirement.
	LabelSelector selector.LabelSelector

	// FieldSelector, when non-empty, restricts the listing to resources whose own
	// fields satisfy every requirement. Callers must have validated it against the
	// listed resource's supported fields (see [SelectableFields]) first.
	FieldSelector selector.FieldSelector

	// NamePrefix, when non-empty, restricts the listing to resources whose name
	// starts with it, case-sensitively.
	NamePrefix string

	// ConnectedOnly, when true, restricts an agent listing to agents that are
	// currently considered connected. It is a no-op for resources that have no
	// connection state. It is the alias for FieldSelector
	// "status.connected=true" and means exactly the same thing.
	ConnectedOnly bool

	// IdentifyingAttributes, when non-empty, restricts an agent listing to agents
	// whose identifying attributes match every key=value pair exactly (an AND of
	// equality conditions). It is a no-op for resources that have no identifying
	// attributes.
	//
	// Deprecated: LabelSelector expresses the same conditions and more.
	IdentifyingAttributes map[string]string

	// NonIdentifyingAttributes, when non-empty, restricts an agent listing to
	// agents whose non-identifying attributes match every key=value pair exactly.
	// It is combined with IdentifyingAttributes via AND, and is a no-op for
	// resources that have no non-identifying attributes.
	//
	// Deprecated: LabelSelector reaches the non-identifying attributes too.
	NonIdentifyingAttributes map[string]string
}

// ToDomain converts the application-level list options to the domain model.
// A nil receiver yields nil so callers can pass options through transparently.
func (o *ListOptions) ToDomain() *model.ListOptions {
	if o == nil {
		return nil
	}

	return &model.ListOptions{
		Limit:                    o.Limit,
		Continue:                 o.Continue,
		IncludeDeleted:           o.IncludeDeleted,
		LabelSelector:            o.LabelSelector,
		FieldSelector:            o.FieldSelector,
		NamePrefix:               o.NamePrefix,
		ConnectedOnly:            o.ConnectedOnly,
		IdentifyingAttributes:    o.IdentifyingAttributes,
		NonIdentifyingAttributes: o.NonIdentifyingAttributes,
	}
}

// GetOptions holds options for getting a single resource at the application boundary.
type GetOptions struct {
	IncludeDeleted bool
}

// ToDomain converts the application-level get options to the domain model.
// A nil receiver yields nil so callers can pass options through transparently.
func (o *GetOptions) ToDomain() *model.GetOptions {
	if o == nil {
		return nil
	}

	return &model.GetOptions{
		IncludeDeleted: o.IncludeDeleted,
	}
}
