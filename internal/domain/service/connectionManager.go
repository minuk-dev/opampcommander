package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v3"
	k8sclock "k8s.io/utils/clock"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// ErrNilArgument is an error that indicates that the argument passed to a function is nil.
var ErrNilArgument = errors.New("argument is nil")

var _ port.ConnectionUsecase = (*ConnectionManager)(nil)

// ConnectionManager is a struct that manages connections.
// It uses a map to store connections and provides methods to save, get, update, and delete connections.
type ConnectionManager struct {
	connectionMap *xsync.MapOf[string, *model.Connection]

	clock clock.Clock
}

// FindConnectionsByData implements port.ConnectionUsecase.
func (cm *ConnectionManager) FindConnectionsByData(ctx context.Context, data map[string]string) ([]*model.Connection, error) {
	panic("unimplemented")
}

// NewConnectionManager creates a new instance of ConnectionManager.
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connectionMap: xsync.NewMapOf[string, *model.Connection](),

		clock: k8sclock.RealClock{},
	}
}

// SaveConnection saves the connection to the map.
// It returns an error if the connection is nil or if the connection already exists.
func (cm *ConnectionManager) SaveConnection(ctx context.Context, connection *model.Connection) error {
	if connection == nil {
		return ErrNilArgument
	}

	// TODO: change connID as key
	// Data + connID is a unique key because anonymous connection
	connID := connection.ID.String()

	_, ok := cm.connectionMap.Load(connID)
	if ok {
		return port.ErrConnectionAlreadyExists
	}

	cm.connectionMap.Store(connID, connection)

	return nil
}

// GetOrCreateConnection returns the connection by the given ID.
// If the connection does not exist, it creates a new one and saves it to the map.
func (cm *ConnectionManager) GetOrCreateConnection(ctx context.Context, instanceUID uuid.UUID) (*model.Connection, error) {
	conn, err := cm.GetConnection(ctx, instanceUID)
	if err == nil {
		return conn, nil
	}

	conn = model.NewConnection(instanceUID)
	conn.RefreshLastCommunicatedAt(cm.clock.Now())

	if err := cm.SaveConnection(ctx, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

// DeleteConnection deletes the connection by the given ID.
// It returns an error if the connection does not exist.
func (cm *ConnectionManager) DeleteConnection(ctx context.Context, connection *model.Connection) error {
	// TODO: Implement
	return errors.New("not implemented")
}

// FetchAndDeleteConnection fetches the connection by the given ID and deletes it from the map.
// It returns the connection if it exists, otherwise it returns an error.
func (cm *ConnectionManager) FetchAndDeleteConnection(ctx context.Context, id uuid.UUID) (*model.Connection, error) {
	conn, exists := cm.connectionMap.Compute(id.String(), func(_ *model.Connection, _ bool) (*model.Connection, bool) {
		return nil, false
	})

	if !exists {
		return nil, port.ErrConnectionNotFound
	}

	return conn, nil
}

// GetConnection returns the connection by the given ID.
func (cm *ConnectionManager) GetConnection(ctx context.Context, id uuid.UUID) (*model.Connection, error) {
	connection, ok := cm.connectionMap.Load(id.String())
	if !ok {
		return nil, port.ErrConnectionNotFound
	}

	return connection, nil
}

// ListConnections returns the list of connections.
func (cm *ConnectionManager) ListConnections(ctx context.Context) []*model.Connection {
	connections := make([]*model.Connection, 0, cm.connectionMap.Size())
	cm.connectionMap.Range(func(_ string, conn *model.Connection) bool {
		connections = append(connections, conn)

		return true
	})

	return connections
}

// ListAliveConnections returns the list of alive connections.
func (cm *ConnectionManager) ListAliveConnections() []*model.Connection {
	var aliveConnections []*model.Connection

	cm.connectionMap.Range(func(_ string, conn *model.Connection) bool {
		if conn.IsAlive(cm.clock.Now()) {
			aliveConnections = append(aliveConnections, conn)
		}

		return true
	})

	return aliveConnections
}
