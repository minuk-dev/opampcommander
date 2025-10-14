package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
)

const serverCollectionName = "servers"

var _ ServerPersistencePort = (*ServerAdapter)(nil)

// ServerPersistencePort is an interface that defines the methods for server persistence.
type ServerPersistencePort interface {
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*domainmodel.Server, error)
	// PutServer saves or updates a server.
	PutServer(ctx context.Context, server *domainmodel.Server) error
	// ListServers retrieves a list of all servers.
	ListServers(ctx context.Context) ([]*domainmodel.Server, error)
}

// ServerAdapter is an adapter for server persistence in MongoDB.
type ServerAdapter struct {
	commonEntityAdapter[entity.Server, string]
}

// NewServerAdapter creates a new instance of ServerAdapter.
func NewServerAdapter(logger *slog.Logger, database *mongo.Database) *ServerAdapter {
	collection := database.Collection(serverCollectionName)

	return &ServerAdapter{
		commonEntityAdapter: newCommonAdapter[entity.Server, string](
			logger,
			collection,
			"serverId",
			func(e *entity.Server) string { return e.ServerID },
			func(key string) any { return key },
		),
	}
}

// GetServer retrieves a server by its ID.
func (a *ServerAdapter) GetServer(ctx context.Context, id string) (*domainmodel.Server, error) {
	e, err := a.get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server from mongodb: %w", err)
	}

	return e.ToDomainModel(), nil
}

// PutServer saves or updates a server.
func (a *ServerAdapter) PutServer(ctx context.Context, server *domainmodel.Server) error {
	e := entity.ToServerEntity(server)

	err := a.put(ctx, e)
	if err != nil {
		return fmt.Errorf("failed to put server to mongodb: %w", err)
	}

	return nil
}

// ListServers retrieves a list of all servers.
func (a *ServerAdapter) ListServers(ctx context.Context) ([]*domainmodel.Server, error) {
	response, err := a.list(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers from mongodb: %w", err)
	}

	servers := make([]*domainmodel.Server, 0, len(response.Items))
	for _, e := range response.Items {
		servers = append(servers, e.ToDomainModel())
	}

	return servers, nil
}
