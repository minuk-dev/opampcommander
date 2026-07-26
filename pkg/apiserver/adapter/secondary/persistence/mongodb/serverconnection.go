package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const (
	serverConnectionCollectionName = "serverconnections"
	serverHeartbeatCollectionName  = "serverheartbeats"

	// heartbeatLookupField is the transient $lookup field; $unset before the page is decoded.
	heartbeatLookupField = "heartbeat"
)

var _ agentport.ServerConnectionPersistencePort = (*ServerConnectionAdapter)(nil)

// ServerConnectionAdapter persists per-server connection records and liveness heartbeats in
// MongoDB. Membership (connections) is written incrementally; liveness (heartbeats) is a single
// O(1) upsert per cycle, and reads join the two so a stale server's connections drop out.
type ServerConnectionAdapter struct {
	collection *mongo.Collection
	heartbeats *mongo.Collection
	logger     *slog.Logger
}

// NewServerConnectionAdapter creates a new instance of ServerConnectionAdapter.
func NewServerConnectionAdapter(database *mongo.Database, logger *slog.Logger) *ServerConnectionAdapter {
	return &ServerConnectionAdapter{
		collection: database.Collection(serverConnectionCollectionName),
		heartbeats: database.Collection(serverHeartbeatCollectionName),
		logger:     logger,
	}
}

// SyncServerConnections implements agentport.ServerConnectionPersistencePort.
//
// Membership (upserts/deletes) is applied first and the heartbeat is refreshed last, so a
// partial failure never leaves a fresh heartbeat advertising an incomplete connection set: the
// server stays out of the cluster view until a fully-successful cycle. The steps are not
// transactional, which is acceptable for a periodic snapshot view.
func (a *ServerConnectionAdapter) SyncServerConnections(
	ctx context.Context,
	serverID string,
	heartbeatAt time.Time,
	upserts []*agentmodel.ServerConnection,
	deletes []uuid.UUID,
) error {
	if len(deletes) > 0 {
		deleteIDs := lo.Map(deletes, func(uid uuid.UUID, _ int) string { return uid.String() })

		_, err := a.collection.DeleteMany(ctx, bson.M{"serverId": serverID, "uid": bson.M{"$in": deleteIDs}})
		if err != nil {
			return fmt.Errorf("failed to delete server connections from mongodb: %w", err)
		}
	}

	for _, conn := range upserts {
		ent := entity.ServerConnectionFromDomain(conn)

		_, err := a.collection.ReplaceOne(ctx, bson.M{"uid": ent.UID}, ent, options.Replace().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("failed to upsert server connection to mongodb: %w", err)
		}
	}

	return a.refreshHeartbeat(ctx, serverID, heartbeatAt)
}

// RemoveServer implements agentport.ServerConnectionPersistencePort.
func (a *ServerConnectionAdapter) RemoveServer(ctx context.Context, serverID string) error {
	_, err := a.heartbeats.DeleteOne(ctx, bson.M{"serverId": serverID})
	if err != nil {
		return fmt.Errorf("failed to delete server heartbeat from mongodb: %w", err)
	}

	_, err = a.collection.DeleteMany(ctx, bson.M{"serverId": serverID})
	if err != nil {
		return fmt.Errorf("failed to delete server connections from mongodb: %w", err)
	}

	return nil
}

// ListServerConnections implements agentport.ServerConnectionPersistencePort. It returns the
// namespace's connections whose owning server is still live, joining each connection to its
// heartbeat with a single $lookup instead of a separate heartbeat query plus a large $in.
func (a *ServerConnectionAdapter) ListServerConnections(
	ctx context.Context,
	namespace string,
	serverID string,
	notBefore time.Time,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	if options == nil {
		//exhaustruct:ignore
		options = &model.ListOptions{}
	}

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	match := bson.M{"namespace": sanitizeResourceName(namespace)}
	if serverID != "" {
		match["serverId"] = sanitizeResourceName(serverID)
	}

	// Keep only connections whose server has a heartbeat at/after notBefore. A zero notBefore
	// still requires a heartbeat to exist, since an absent one yields no lastSeenAt to match.
	prefix := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: serverHeartbeatCollectionName},
			{Key: "localField", Value: "serverId"},
			{Key: "foreignField", Value: "serverId"},
			{Key: "as", Value: heartbeatLookupField},
		}}},
		bson.D{{Key: "$match", Value: bson.M{
			heartbeatLookupField + ".lastSeenAt": bson.M{"$gte": notBefore},
		}}},
		bson.D{{Key: "$unset", Value: heartbeatLookupField}},
	}

	entities, continueToken, remaining, err := aggregateListPage[entity.ServerConnection](
		ctx, a.logger, a.collection, prefix, continueTokenObjectID, options.Limit,
	)
	if err != nil {
		return nil, err
	}

	return &model.ListResponse[*agentmodel.ServerConnection]{
		Items: lo.Map(entities, func(item *entity.ServerConnection, _ int) *agentmodel.ServerConnection {
			return item.ToDomain()
		}),
		Continue:           continueToken,
		RemainingItemCount: remaining,
	}, nil
}

func (a *ServerConnectionAdapter) refreshHeartbeat(ctx context.Context, serverID string, at time.Time) error {
	heartbeat := &entity.ServerHeartbeat{ID: nil, ServerID: serverID, LastSeenAt: at}

	_, err := a.heartbeats.ReplaceOne(ctx, bson.M{"serverId": serverID}, heartbeat, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("failed to refresh server heartbeat in mongodb: %w", err)
	}

	return nil
}
