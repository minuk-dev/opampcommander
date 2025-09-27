package client

import (
	"context"

	uuid "github.com/google/uuid"

	connectionv1 "github.com/minuk-dev/opampcommander/api/v1/connection"
)

// API paths for connection resources.
const (
	// ListConnectionsPath is the path to list all connections.
	ListConnectionsPath = "/api/v1/connections"
	// GetConnectionPath is the path to get a connection by ID.
	GetConnectionPath = "/api/v1/connections/{id}"
)

// ConnectionService provides methods to interact with connection resources.
type ConnectionService struct {
	service *service
}

// NewConnectionService creates a new ConnectionService.
func NewConnectionService(service *service) *ConnectionService {
	return &ConnectionService{
		service: service,
	}
}

// GetConnection retrieves a connection by its ID.
func (s *ConnectionService) GetConnection(ctx context.Context, id uuid.UUID) (*connectionv1.Connection, error) {
	return getResource[connectionv1.Connection](ctx, s.service, GetConnectionPath, id.String())
}

// ListConnections lists all connections.
func (s *ConnectionService) ListConnections(
	ctx context.Context,
	opts ...ListOption,
) (*connectionv1.ListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[connectionv1.Connection](
		ctx,
		s.service,
		ListConnectionsPath,
		ListSettings{
			limit:         listSettings.limit,
			continueToken: listSettings.continueToken,
		},
	)
}
