package model

// ListOptions is a struct that holds options for listing resources.
type ListOptions struct {
	Limit          int64
	Continue       string
	IncludeDeleted bool
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
