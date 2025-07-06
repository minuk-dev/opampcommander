// Package v1 provides the API server for the application.
// It includes common API definitions and utilities for version 1 of the API.
package v1

const (
	// APIVersion is the version of the API.
	APIVersion = "v1"
)

// ListMeta is a struct that contains metadata for list responses.
type ListMeta struct {
	Continue           string `json:"continue"`
	RemainingItemCount int64  `json:"remainingItemCount"`
} // @name ListMeta

// ListResponse is a struct that represents the response for listing agents.
type ListResponse[T any] struct {
	Kind       string   `json:"kind"`
	APIVersion string   `json:"apiVersion"`
	Metadata   ListMeta `json:"metadata"`
	Items      []T      `json:"items"`
} // @name ListResponse
