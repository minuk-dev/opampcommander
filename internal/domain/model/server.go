package model

import (
	"time"

	"github.com/samber/lo"
)

// Server represents a server that an agent communicates with.
// It contains the server's unique identifier and liveness information.
type Server struct {
	// ID is the unique identifier for the server.
	ID string
	// LastHeartbeatAt is the last time the server sent a heartbeat.
	LastHeartbeatAt time.Time
	// Conditions is a list of conditions that apply to the server.
	Conditions []ServerCondition
}

// ServerCondition represents a condition of a server.
type ServerCondition struct {
	// Type is the type of the condition.
	Type ServerConditionType
	// LastTransitionTime is the last time the condition transitioned.
	LastTransitionTime time.Time
	// Status is the status of the condition.
	Status ServerConditionStatus
	// Reason is the identifier of the user or system that triggered the condition.
	Reason string
	// Message is a human readable message indicating details about the condition.
	Message string
}

// ServerConditionType represents the type of a server condition.
type ServerConditionType string

const (
	// ServerConditionTypeRegistered represents the condition when the server was registered.
	ServerConditionTypeRegistered ServerConditionType = "Registered"
	// ServerConditionTypeAlive represents the condition when the server is alive.
	ServerConditionTypeAlive ServerConditionType = "Alive"
)

// ServerConditionStatus represents the status of a server condition.
type ServerConditionStatus string

const (
	// ServerConditionStatusTrue represents a true condition status.
	ServerConditionStatusTrue ServerConditionStatus = "True"
	// ServerConditionStatusFalse represents a false condition status.
	ServerConditionStatusFalse ServerConditionStatus = "False"
	// ServerConditionStatusUnknown represents an unknown condition status.
	ServerConditionStatusUnknown ServerConditionStatus = "Unknown"
)

// IsAlive returns true if the server is alive based on the heartbeat timeout.
// A server is considered alive if its last heartbeat was within the timeout period.
func (s *Server) IsAlive(now time.Time, timeout time.Duration) bool {
	return now.Sub(s.LastHeartbeatAt) < timeout
}

// SetCondition sets or updates a condition in the server's status.
func (s *Server) SetCondition(conditionType ServerConditionType, status ServerConditionStatus, reason, message string) {
	now := time.Now()

	// Check if condition already exists
	_, idx, ok := lo.FindIndexOf(s.Conditions, func(condition ServerCondition) bool {
		return condition.Type == conditionType
	})
	if ok {
		if s.Conditions[idx].Status == status {
			// Update existing condition only if status changed
			s.Conditions[idx].Status = status
			s.Conditions[idx].LastTransitionTime = now
			s.Conditions[idx].Reason = reason
			s.Conditions[idx].Message = message
		}

		return
	}

	// Add new condition
	s.Conditions = append(s.Conditions, ServerCondition{
		Type:               conditionType,
		LastTransitionTime: now,
		Status:             status,
		Reason:             reason,
		Message:            message,
	})
}

// GetCondition returns the condition of the specified type.
func (s *Server) GetCondition(conditionType ServerConditionType) *ServerCondition {
	condition, ok := lo.Find(s.Conditions, func(condition ServerCondition) bool {
		return condition.Type == conditionType
	})
	if !ok {
		return nil
	}

	return &condition
}

// IsConditionTrue checks if the specified condition type is true.
func (s *Server) IsConditionTrue(conditionType ServerConditionType) bool {
	condition := s.GetCondition(conditionType)

	return condition != nil && condition.Status == ServerConditionStatusTrue
}

// MarkRegistered marks the server as registered.
func (s *Server) MarkRegistered(reason string) {
	s.SetCondition(ServerConditionTypeRegistered, ServerConditionStatusTrue, reason, "Server registered")
}

// MarkAlive marks the server as alive.
func (s *Server) MarkAlive(reason string) {
	s.SetCondition(ServerConditionTypeAlive, ServerConditionStatusTrue, reason, "Server is alive")
}

// MarkNotAlive marks the server as not alive.
func (s *Server) MarkNotAlive(reason string) {
	s.SetCondition(ServerConditionTypeAlive, ServerConditionStatusFalse, reason, "Server is not responding")
}

// GetRegisteredAt returns the timestamp when the server was registered.
func (s *Server) GetRegisteredAt() *time.Time {
	for _, condition := range s.Conditions {
		if condition.Type == ServerConditionTypeRegistered && condition.Status == ServerConditionStatusTrue {
			return &condition.LastTransitionTime
		}
	}

	return nil
}

// GetRegisteredBy returns the identifier of the user or system that registered the server.
func (s *Server) GetRegisteredBy() string {
	for _, condition := range s.Conditions {
		if condition.Type == ServerConditionTypeRegistered && condition.Status == ServerConditionStatusTrue {
			return condition.Reason
		}
	}

	return ""
}
