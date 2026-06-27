package entity

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// ServerConnection represents a per-server connection snapshot in MongoDB. Each alive
// server periodically replaces its own set of records so the cluster-wide connection view
// can be queried from any node.
type ServerConnection struct {
	// ID is the MongoDB ObjectID.
	ID *bson.ObjectID `bson:"_id,omitempty"`
	// ServerID is the server instance that owns this connection.
	ServerID string `bson:"serverId"`
	// UID is the connection's unique identifier (string form of the UUID).
	UID string `bson:"uid"`
	// InstanceUID is the agent instance UID (string form of the UUID).
	InstanceUID string `bson:"instanceUid"`
	// Type is the connection type string (e.g. "HTTP", "WebSocket").
	Type string `bson:"type"`
	// Namespace is the namespace the connection belongs to.
	Namespace string `bson:"namespace"`
	// LastCommunicatedAt is the last time the connection was communicated with.
	LastCommunicatedAt time.Time `bson:"lastCommunicatedAt"`
	// SnapshotAt is when the owning server last refreshed this record.
	SnapshotAt time.Time `bson:"snapshotAt"`
}

// ToDomain converts the entity to a domain model. Unparseable UUIDs become uuid.Nil.
func (e *ServerConnection) ToDomain() *agentmodel.ServerConnection {
	if e == nil {
		return nil
	}

	uid, _ := uuid.Parse(e.UID)
	instanceUID, _ := uuid.Parse(e.InstanceUID)

	return &agentmodel.ServerConnection{
		ServerID:           e.ServerID,
		UID:                uid,
		InstanceUID:        instanceUID,
		Type:               agentmodel.ConnectionTypeFromString(e.Type),
		Namespace:          e.Namespace,
		LastCommunicatedAt: e.LastCommunicatedAt,
		SnapshotAt:         e.SnapshotAt,
	}
}

// ServerConnectionFromDomain converts a domain model to an entity.
func ServerConnectionFromDomain(conn *agentmodel.ServerConnection) *ServerConnection {
	if conn == nil {
		return nil
	}

	return &ServerConnection{
		ID:                 nil,
		ServerID:           conn.ServerID,
		UID:                conn.UID.String(),
		InstanceUID:        conn.InstanceUID.String(),
		Type:               conn.Type.String(),
		Namespace:          conn.Namespace,
		LastCommunicatedAt: conn.LastCommunicatedAt,
		SnapshotAt:         conn.SnapshotAt,
	}
}
