package client

import (
	uuid "github.com/google/uuid"

	connectionv1 "github.com/minuk-dev/opampcommander/api/v1/connection"
)

// API paths for connection resources.
const (
	// ListConnectionsPath is the path to list all connections.
	ListConnectionsPath = "/api/v1/connections"
	// GetConnectionPath is the path to get a connection by ID.
	GetConnectionPath = "/api/v1/connections/:id"
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
func (s *ConnectionService) GetConnection(id uuid.UUID) (*connectionv1.Connection, error) {
	return getResource[connectionv1.Connection](s.service, GetConnectionPath, id)
}

// ListConnections lists all connections.
func (s *ConnectionService) ListConnections() ([]*connectionv1.Connection, error) {
	return listResources[*connectionv1.Connection](s.service, ListConnectionsPath)
}
