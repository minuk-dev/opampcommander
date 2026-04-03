package agentmodel

import (
	"time"

	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Server represents a server that an agent communicates with.
// It contains the server's unique identifier and liveness information.
type Server struct {
	// ID is the unique identifier for the server.
	ID string
	// LastHeartbeatAt is the last time the server sent a heartbeat.
	LastHeartbeatAt time.Time
	// Conditions is a list of conditions that apply to the server.
	Conditions []model.Condition
}

// Clone creates a deep copy of the Server.
func (s *Server) Clone() *Server {
	conditionsCopy := make([]model.Condition, len(s.Conditions))
	copy(conditionsCopy, s.Conditions)

	return &Server{
		ID:              s.ID,
		LastHeartbeatAt: s.LastHeartbeatAt,
		Conditions:      conditionsCopy,
	}
}

// IsAlive returns true if the server is alive based on the heartbeat timeout.
// A server is considered alive if its last heartbeat was within the timeout period.
func (s *Server) IsAlive(now time.Time, timeout time.Duration) bool {
	return now.Sub(s.LastHeartbeatAt) < timeout
}

// SetCondition sets or updates a condition in the server's status.
func (s *Server) SetCondition(conditionType model.ConditionType, status model.ConditionStatus, reason, message string) {
	now := time.Now()

	// Check if condition already exists
	_, idx, ok := lo.FindIndexOf(s.Conditions, func(condition model.Condition) bool {
		return condition.Type == conditionType
	})
	if ok {
		if s.Conditions[idx].Status != status {
			// Update existing condition only if status changed
			s.Conditions[idx].Status = status
			s.Conditions[idx].LastTransitionTime = now
			s.Conditions[idx].Reason = reason
			s.Conditions[idx].Message = message
		}

		return
	}

	// Add new condition
	s.Conditions = append(s.Conditions, model.Condition{
		Type:               conditionType,
		LastTransitionTime: now,
		Status:             status,
		Reason:             reason,
		Message:            message,
	})
}

// GetCondition returns the condition of the specified type.
func (s *Server) GetCondition(conditionType model.ConditionType) *model.Condition {
	condition, ok := lo.Find(s.Conditions, func(condition model.Condition) bool {
		return condition.Type == conditionType
	})
	if !ok {
		return nil
	}

	return &condition
}

// IsConditionTrue checks if the specified condition type is true.
func (s *Server) IsConditionTrue(conditionType model.ConditionType) bool {
	condition := s.GetCondition(conditionType)

	return condition != nil && condition.Status == model.ConditionStatusTrue
}

// MarkRegistered marks the server as registered.
func (s *Server) MarkRegistered(reason string) {
	s.SetCondition(model.ConditionTypeCreated, model.ConditionStatusTrue, reason, "Server registered")
}

// GetRegisteredAt returns the time when the server was registered.
func (s *Server) GetRegisteredAt() *time.Time {
	condition := s.GetCondition(model.ConditionTypeCreated)
	if condition == nil {
		return nil
	}

	return &condition.LastTransitionTime
}

// GetRegisteredBy returns the reason/actor who registered the server.
func (s *Server) GetRegisteredBy() string {
	condition := s.GetCondition(model.ConditionTypeCreated)
	if condition == nil {
		return ""
	}

	return condition.Reason
}
