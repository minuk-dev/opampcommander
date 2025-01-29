package service

import (
	"github.com/minuk-dev/minuk-apiserver/internal/domain/model"
)

type ConnectionManager struct {
	connectionMap map[string]model.Connection
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connectionMap: make(map[string]model.Connection),
	}
}

func (cm *ConnectionManager) SetConnection(connection model.Connection) {
	id := connection.ID()
	cm.connectionMap[id.String()] = connection
}

// GetConnection returns the connection by the given ID.
//
//nolint:ireturn
func (cm *ConnectionManager) GetConnection(id string) model.Connection {
	return cm.connectionMap[id]
}
