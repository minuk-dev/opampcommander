package model

import (
	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// ListOptions is a struct that holds options for listing resources.
type ListOptions struct {
	Limit          int64
	Continue       string
	IncludeDeleted bool

	// LabelSelector, when non-empty, restricts the listing to resources whose
	// user-supplied metadata map satisfies every requirement. Which field backs
	// that map is per-aggregate; see [SelectorValues.Labels]. It is a no-op for
	// resources that carry no such map.
	LabelSelector selector.LabelSelector

	// FieldSelector, when non-empty, restricts the listing to resources whose own
	// fields satisfy every requirement. Only the fields an aggregate documents as
	// selectable may appear; the API boundary rejects anything else with a 400, so
	// a client never silently receives an unfiltered list.
	FieldSelector selector.FieldSelector

	// NamePrefix, when non-empty, restricts the listing to resources whose name
	// starts with it. The match is case-sensitive so it can be served by an index
	// range scan rather than a collection-wide regex.
	NamePrefix string

	// ConnectedOnly, when true, restricts an agent listing to agents that are
	// currently considered connected. "Connected" here mirrors the per-agent
	// Connected field exactly (Status.Connected is set AND the agent reported
	// within the heartbeat-staleness window), so a filtered list and the
	// connected badge/count never disagree. It is a no-op for resources that
	// have no connection state.
	ConnectedOnly bool

	// IdentifyingAttributes, when non-empty, restricts an agent listing to agents
	// whose identifying attributes match every key=value pair exactly (an AND of
	// equality conditions, mirroring agent-group selector semantics). It is a
	// no-op for resources that have no identifying attributes.
	IdentifyingAttributes map[string]string

	// NonIdentifyingAttributes, when non-empty, restricts an agent listing to
	// agents whose non-identifying attributes match every key=value pair exactly
	// (an AND of equality conditions). It is combined with IdentifyingAttributes
	// via AND, and is a no-op for resources that have no non-identifying attributes.
	NonIdentifyingAttributes map[string]string
}

// GetOptions is a struct that holds options for getting a single resource.
type GetOptions struct {
	IncludeDeleted bool
}

// ListResponse is a generic struct that represents a paginated response for listing resources.
//
// Pagination convention, shared by every listing path: Continue is an opaque resume token,
// non-empty whenever Items is non-empty (even on the final page) and echoed back verbatim on
// the next request. End-of-list is signaled by RemainingItemCount reaching 0, not by Continue
// becoming empty.
type ListResponse[T any] struct {
	RemainingItemCount int64
	Continue           string
	Items              []T
}
