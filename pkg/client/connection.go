package client

import (
	uuid "github.com/google/uuid"

	connectionv1 "github.com/minuk-dev/opampcommander/api/v1/connection"
)

const (
	ListConnectionsPath = "/v1/connections"
	GetConnectionPath   = "/v1/connections/:id"
)

func (c *Client) GetConnection(id uuid.UUID) (*connectionv1.Connection, error) {
	return getResource[connectionv1.Connection](c, GetConnectionPath, id)
}

func (c *Client) ListConnections() ([]*connectionv1.Connection, error) {
	return listResources[*connectionv1.Connection](c, ListConnectionsPath)
}
