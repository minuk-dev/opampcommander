// Package v1 provides the API server for the application.
// It includes common API definitions and utilities for version 1 of the API.
package v1

const (
	// APIVersion is the version of the API.
	APIVersion = "v1"
)

type ListMeta struct {
	Continue           string `json:"continue"`
	RemainingItemCount int    `json:"remainingItemCount"`
} // @name ListMeta
