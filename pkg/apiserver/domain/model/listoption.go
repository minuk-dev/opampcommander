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
