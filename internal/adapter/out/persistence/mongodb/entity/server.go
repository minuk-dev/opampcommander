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
	// CreatedAt is the time the server was first registered.
	CreatedAt time.Time `bson:"createdAt"`
}

// ToDomainModel converts the Server entity to a domain model.
func (s *Server) ToDomainModel() *domainmodel.Server {
	if s == nil {
		return nil
	}

	return &domainmodel.Server{
		ID:              s.ServerID,
		LastHeartbeatAt: s.LastHeartbeatAt,
		CreatedAt:       s.CreatedAt,
	}
}

// ToServerEntity converts a domain model to a Server entity.
func ToServerEntity(server *domainmodel.Server) *Server {
	if server == nil {
		return nil
	}

	return &Server{
		ID:              nil,
		ServerID:        server.ID,
		LastHeartbeatAt: server.LastHeartbeatAt,
		CreatedAt:       server.CreatedAt,
	}
}
