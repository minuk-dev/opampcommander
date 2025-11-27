package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Server represents a server entity in MongoDB.
type Server struct {
	// ID is the MongoDB ObjectID.
	ID *bson.ObjectID `bson:"_id,omitempty"`
	// ServerID is the unique identifier for the server.
	ServerID string `bson:"serverId"`
	// LastHeartbeatAt is the last time the server sent a heartbeat.
	LastHeartbeatAt time.Time `bson:"lastHeartbeatAt"`
	// Conditions is a list of conditions that apply to the server.
	Conditions []ServerCondition `bson:"conditions,omitempty"`
}

// ServerCondition represents a condition of a server in MongoDB.
type ServerCondition struct {
	Type               string        `bson:"type"`
	LastTransitionTime bson.DateTime `bson:"lastTransitionTime"`
	Status             string        `bson:"status"`
	Reason             string        `bson:"reason"`
	Message            string        `bson:"message,omitempty"`
}

// ToDomainModel converts the Server entity to a domain model.
func (s *Server) ToDomainModel() *domainmodel.Server {
	if s == nil {
		return nil
	}

	conditions := make([]domainmodel.ServerCondition, len(s.Conditions))
	for i, condition := range s.Conditions {
		conditions[i] = domainmodel.ServerCondition{
			Type:               domainmodel.ServerConditionType(condition.Type),
			LastTransitionTime: condition.LastTransitionTime.Time(),
			Status:             domainmodel.ServerConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return &domainmodel.Server{
		ID:              s.ServerID,
		LastHeartbeatAt: s.LastHeartbeatAt,
		Conditions:      conditions,
	}
}

// ToServerEntity converts a domain model to a Server entity.
func ToServerEntity(server *domainmodel.Server) *Server {
	if server == nil {
		return nil
	}

	conditions := make([]ServerCondition, len(server.Conditions))
	for i, condition := range server.Conditions {
		conditions[i] = ServerCondition{
			Type:               string(condition.Type),
			LastTransitionTime: bson.NewDateTimeFromTime(condition.LastTransitionTime),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return &Server{
		ID:              nil,
		ServerID:        server.ID,
		LastHeartbeatAt: server.LastHeartbeatAt,
		Conditions:      conditions,
	}
}
