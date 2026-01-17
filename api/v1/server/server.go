// Package server provides the API models for server management.
package server

import (
	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// Kind is the kind of the server.
	Kind = "Server"
)

// Server represents an API server instance.
type Server struct {
	ID              string      `json:"id"`
	LastHeartbeatAt v1.Time     `json:"lastHeartbeatAt"`
	Conditions      []Condition `json:"conditions"`
} // @name Server

// Condition represents a condition of a server.
type Condition struct {
	Type               ConditionType   `json:"type"`
	LastTransitionTime v1.Time         `json:"lastTransitionTime"`
	Status             ConditionStatus `json:"status"`
	Reason             string          `json:"reason"`
	Message            string          `json:"message,omitempty"`
} // @name ServerCondition

// ConditionType represents the type of a server condition.
type ConditionType string // @name ServerConditionType

const (
	// ConditionTypeRegistered represents the condition when the server was registered.
	ConditionTypeRegistered ConditionType = "Registered"
	// ConditionTypeAlive represents the condition when the server is alive.
	ConditionTypeAlive ConditionType = "Alive"
)

// ConditionStatus represents the status of a server condition.
type ConditionStatus string // @name ServerConditionStatus

const (
	// ConditionStatusTrue represents a true condition status.
	ConditionStatusTrue ConditionStatus = "True"
	// ConditionStatusFalse represents a false condition status.
	ConditionStatusFalse ConditionStatus = "False"
	// ConditionStatusUnknown represents an unknown condition status.
	ConditionStatusUnknown ConditionStatus = "Unknown"
)

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
