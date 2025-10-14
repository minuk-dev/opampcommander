// Package server provides the API models for server management.
package server

import (
	"time"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// Kind is the kind of the server.
	Kind = "Server"
)

// Server represents an API server instance.
type Server struct {
	ID              string    `json:"id"`
	LastHeartbeatAt time.Time `json:"lastHeartbeatAt"`
	CreatedAt       time.Time `json:"createdAt"`
} // @name Server

// ListResponse represents a list of servers with metadata.
type ListResponse = v1.ListResponse[Server]

// NewListResponse creates a new ListResponse with the given servers and metadata.
func NewListResponse(servers []Server, metadata v1.ListMeta) *ListResponse {
	return &ListResponse{
		Kind:       Kind,
		APIVersion: v1.APIVersion,
		Metadata:   metadata,
		Items:      servers,
	}
}
