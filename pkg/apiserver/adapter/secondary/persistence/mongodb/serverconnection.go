package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const serverConnectionCollectionName = "serverconnections"

var _ agentport.ServerConnectionPersistencePort = (*ServerConnectionAdapter)(nil)

// ServerConnectionAdapter persists per-server connection snapshots in MongoDB.
type ServerConnectionAdapter struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// NewServerConnectionAdapter creates a new instance of ServerConnectionAdapter.
func NewServerConnectionAdapter(database *mongo.Database, logger *slog.Logger) *ServerConnectionAdapter {
	return &ServerConnectionAdapter{
		collection: database.Collection(serverConnectionCollectionName),
		logger:     logger,
	}
}

// ReplaceServerConnections implements agentport.ServerConnectionPersistencePort.
//
// It deletes the owning server's existing records and inserts the new set. The two steps
// are not transactional; a reader hitting the brief gap simply sees this server's
// connections momentarily missing, which is acceptable for a periodic snapshot view.
func (a *ServerConnectionAdapter) ReplaceServerConnections(
	ctx context.Context,
	serverID string,
	conns []*agentmodel.ServerConnection,
) error {
	_, err := a.collection.DeleteMany(ctx, bson.M{"serverId": serverID})
	if err != nil {
		return fmt.Errorf("failed to delete server connections from mongodb: %w", err)
	}

	if len(conns) == 0 {
		return nil
	}

	docs := lo.Map(conns, func(conn *agentmodel.ServerConnection, _ int) any {
		return entity.ServerConnectionFromDomain(conn)
	})

	_, err = a.collection.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to insert server connections to mongodb: %w", err)
	}

	return nil
}

// ListServerConnections implements agentport.ServerConnectionPersistencePort.
func (a *ServerConnectionAdapter) ListServerConnections(
	ctx context.Context,
	namespace string,
	notBefore time.Time,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	if options == nil {
		//exhaustruct:ignore
		options = &model.ListOptions{}
	}

	conditions := []bson.M{{"namespace": sanitizeResourceName(namespace)}}
	if !notBefore.IsZero() {
		conditions = append(conditions, bson.M{"snapshotAt": bson.M{"$gte": notBefore}})
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
