package service

import (
	"errors"

	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v3"

	"github.com/minuk-dev/minuk-apiserver/internal/domain/model"
	"github.com/minuk-dev/minuk-apiserver/internal/domain/port"
)

var ErrNilArgument = errors.New("argument is nil")

var _ port.ConnectionUsecase = (*ConnectionManager)(nil)

type ConnectionManager struct {
	connectionMap *xsync.MapOf[string, *model.Connection]
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connectionMap: xsync.NewMapOf[string, *model.Connection](),
	}
}

func (cm *ConnectionManager) SetConnection(connection *model.Connection) error {
	if connection == nil {
		return ErrNilArgument
	}

	connID := connection.ID.String()

	_, ok := cm.connectionMap.Load(connID)
	if ok {
		return port.ErrConnectionAlreadyExists
	}

	cm.connectionMap.Store(connID, connection)

	return nil
}

func (cm *ConnectionManager) DeleteConnection(id uuid.UUID) error {
	var exists bool

	_, _ = cm.connectionMap.Compute(id.String(), func(_ *model.Connection, loaded bool) (*model.Connection, bool) {
		exists = loaded

		return nil, false
	})

	if !exists {
		return port.ErrConnectionNotFound
	}

	return nil
}

// GetConnection returns the connection by the given ID.
func (cm *ConnectionManager) GetConnection(id uuid.UUID) (*model.Connection, error) {
	connection, ok := cm.connectionMap.Load(id.String())
	if !ok {
		return nil, port.ErrConnectionNotFound
	}

	return connection, nil
}

// ListConnectionIDs returns the list of connection IDs.
func (cm *ConnectionManager) ListConnectionIDs() []uuid.UUID {
	var rawIDs []string

	cm.connectionMap.Range(func(key string, _ *model.Connection) bool {
		rawIDs = append(rawIDs, key)

		return true
	})

	ids := make([]uuid.UUID, len(rawIDs))

	for i, rawID := range rawIDs {
		id, _ := uuid.Parse(rawID)
		ids[i] = id
	}

	return ids
}
