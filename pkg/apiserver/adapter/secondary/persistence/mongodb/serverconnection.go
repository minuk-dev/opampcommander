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

// ListServerConnections implements agentport.ServerConnectionPersistencePort. It first resolves
// the set of live servers from the heartbeat collection, then returns only their connections.
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

	liveServerIDs, err := a.liveServerIDs(ctx, serverID, notBefore)
	if err != nil {
		return nil, err
	}

	if len(liveServerIDs) == 0 {
		//exhaustruct:ignore
		return &model.ListResponse[*agentmodel.ServerConnection]{}, nil
	}

	conditions := []bson.M{
		{"namespace": sanitizeResourceName(namespace)},
		{"serverId": bson.M{"$in": liveServerIDs}},
	}

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	if continueTokenFilter := withContinueToken(continueTokenObjectID); continueTokenFilter != nil {
		conditions = append(conditions, continueTokenFilter)
	}

	filter := buildFilter(conditions)

	count, err := a.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count server connections in mongodb: %w", err)
	}

	entities, continueToken, err := a.findServerConnections(ctx, filter, options.Limit)
	if err != nil {
		return nil, err
	}

	return &model.ListResponse[*agentmodel.ServerConnection]{
		Items: lo.Map(entities, func(item *entity.ServerConnection, _ int) *agentmodel.ServerConnection {
			return item.ToDomain()
		}),
		Continue:           continueToken,
		RemainingItemCount: count - int64(len(entities)),
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

// liveServerIDs returns the serverIDs whose heartbeat is at or after notBefore. A non-empty
// serverID restricts the query to that one server; a zero notBefore matches every heartbeat.
func (a *ServerConnectionAdapter) liveServerIDs(
	ctx context.Context,
	serverID string,
	notBefore time.Time,
) ([]string, error) {
	filter := bson.M{}
	if serverID != "" {
		filter["serverId"] = sanitizeResourceName(serverID)
	}

	if !notBefore.IsZero() {
		filter["lastSeenAt"] = bson.M{"$gte": notBefore}
	}

	cursor, err := a.heartbeats.Find(ctx, filter, options.Find().SetProjection(bson.M{"serverId": 1}))
	if err != nil {
		return nil, fmt.Errorf("failed to find server heartbeats in mongodb: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var heartbeats []*entity.ServerHeartbeat

	err = cursor.All(ctx, &heartbeats)
	if err != nil {
		return nil, fmt.Errorf("failed to decode server heartbeats from mongodb: %w", err)
	}

	return lo.Map(heartbeats, func(hb *entity.ServerHeartbeat, _ int) string { return hb.ServerID }), nil
}

// findServerConnections runs the paginated find and returns the decoded entities plus the
// continue token for the next page.
func (a *ServerConnectionAdapter) findServerConnections(
	ctx context.Context,
	filter bson.M,
	limit int64,
) ([]*entity.ServerConnection, string, error) {
	cursor, err := a.collection.Find(ctx, filter, withPageOptions(limit))
	if err != nil {
		return nil, "", fmt.Errorf("failed to find server connections in mongodb: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var entities []*entity.ServerConnection

	err = cursor.All(ctx, &entities)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode server connections from mongodb: %w", err)
	}

	continueToken, err := getContinueTokenFromEntities(entities)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get continue token from entities: %w", err)
	}

	return entities, continueToken, nil
}
