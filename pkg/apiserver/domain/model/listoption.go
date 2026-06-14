package model

// ListOptions is a struct that holds options for listing resources.
type ListOptions struct {
	Limit          int64
	Continue       string
	IncludeDeleted bool

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
type ListResponse[T any] struct {
	RemainingItemCount int64
	Continue           string
	Items              []T
}
