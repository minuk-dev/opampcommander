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

// GetConnection retrieves a connection by its ID.
func (c *Client) GetConnection(id uuid.UUID) (*connectionv1.Connection, error) {
	return getResource[connectionv1.Connection](c, GetConnectionPath, id)
}

// ListConnections lists all connections.
func (c *Client) ListConnections() ([]*connectionv1.Connection, error) {
	return listResources[*connectionv1.Connection](c, ListConnectionsPath)
}
