package service

import (
	"errors"

	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v3"
	k8sclock "k8s.io/utils/clock"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var ErrNilArgument = errors.New("argument is nil")

var _ port.ConnectionUsecase = (*ConnectionManager)(nil)

type ConnectionManager struct {
	connectionMap *xsync.MapOf[string, *model.Connection]

	clock clock.Clock
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connectionMap: xsync.NewMapOf[string, *model.Connection](),

		clock: k8sclock.RealClock{},
	}
}

func (cm *ConnectionManager) SaveConnection(connection *model.Connection) error {
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

func (cm *ConnectionManager) GetOrCreateConnection(instanceUID uuid.UUID) (*model.Connection, error) {
	conn, err := cm.GetConnection(instanceUID)
	if err == nil {
		return conn, nil
	}

	conn = model.NewConnection(instanceUID)
	conn.RefreshLastCommunicatedAt(cm.clock.Now())

	if err := cm.SaveConnection(conn); err != nil {
		return nil, err
	}

	return conn, nil
}

func (cm *ConnectionManager) DeleteConnection(id uuid.UUID) error {
	_, err := cm.FetchAndDeleteConnection(id)

	return err
}

func (cm *ConnectionManager) FetchAndDeleteConnection(id uuid.UUID) (*model.Connection, error) {
	conn, exists := cm.connectionMap.Compute(id.String(), func(_ *model.Connection, _ bool) (*model.Connection, bool) {
		return nil, false
	})

	if !exists {
		return nil, port.ErrConnectionNotFound
	}

	return conn, nil
}

// GetConnection returns the connection by the given ID.
func (cm *ConnectionManager) GetConnection(id uuid.UUID) (*model.Connection, error) {
	connection, ok := cm.connectionMap.Load(id.String())
	if !ok {
		return nil, port.ErrConnectionNotFound
	}

	return connection, nil
}

// ListConnections returns the list of connections.
func (cm *ConnectionManager) ListConnections() []*model.Connection {
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
