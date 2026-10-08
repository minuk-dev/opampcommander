// Package inmemory implements node-local state stores.
package inmemory

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/xsync"
)

var _ agentport.ConnectionStore = (*ConnectionStore)(nil)

// ConnectionStore owns mutable connection state for one APIServer process.
type ConnectionStore struct {
	mu       sync.RWMutex
	byID     map[string]agentmodel.Connection
	byAgent  map[uuid.UUID]string
	sessions xsync.KeyedMutex[uuid.UUID]
	closed   chan any
	snapshot agentport.ConnectionSnapshotState
}

// NewConnectionStore creates the store shared by connection and OpAMP services.
func NewConnectionStore() *ConnectionStore {
	return &ConnectionStore{
		mu:       sync.RWMutex{},
		byID:     make(map[string]agentmodel.Connection),
		byAgent:  make(map[uuid.UUID]string),
		sessions: xsync.KeyedMutex[uuid.UUID]{},
		closed:   make(chan any, 1),
		snapshot: agentport.ConnectionSnapshotState{},
	}
}

// Get returns a copy of the connection identified by its transport ID.
func (s *ConnectionStore) Get(_ context.Context, id any) (*agentmodel.Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getConnection(agentmodel.ConvertConnIDToString(id))
}

// GetConnectionByInstanceUID returns a copy of the current agent connection.
func (s *ConnectionStore) GetConnectionByInstanceUID(
	_ context.Context, uid uuid.UUID,
) (*agentmodel.Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getConnection(s.byAgent[uid])
}

// ListConnections returns copies of all node-local connections.
func (s *ConnectionStore) ListConnections(_ context.Context) ([]*agentmodel.Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	connections := make([]*agentmodel.Connection, 0, len(s.byID))
	for _, connection := range s.byID {
		connections = append(connections, &connection)
	}

	return connections, nil
}

// Put stores a copy and updates the agent index atomically.
func (s *ConnectionStore) Put(_ context.Context, connection *agentmodel.Connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := connection.IDString()
	if previous, ok := s.byID[key]; ok && s.byAgent[previous.InstanceUID] == key {
		delete(s.byAgent, previous.InstanceUID)
	}

	s.byID[key] = *connection
	if !connection.IsAnonymous() {
		s.byAgent[connection.InstanceUID] = key
	}

	return nil
}

// Delete removes only the matching connection and its owned index.
func (s *ConnectionStore) Delete(_ context.Context, connection *agentmodel.Connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := connection.IDString()
	stored, ok := s.byID[key]

	if !ok || stored.UID != connection.UID {
		return nil
	}

	delete(s.byID, key)

	if s.byAgent[stored.InstanceUID] == key {
		delete(s.byAgent, stored.InstanceUID)
	}

	return nil
}

// WithinSession orders agent work without exposing locks to callers.
func (s *ConnectionStore) WithinSession(
	ctx context.Context, uid uuid.UUID, operation func(context.Context) error,
) error {
	s.sessions.Lock(uid)
	defer s.sessions.Unlock(uid)

	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("session operation canceled: %w", err)
	}

	err = operation(ctx)
	if err != nil {
		return fmt.Errorf("session operation failed: %w", err)
	}

	return nil
}

// EnqueueClosedConnection records a close for background cleanup without blocking.
func (s *ConnectionStore) EnqueueClosedConnection(id any) bool {
	select {
	case s.closed <- id:
		return true
	default:
		return false
	}
}

// NextClosedConnection waits for a close or context cancellation.
func (s *ConnectionStore) NextClosedConnection(ctx context.Context) (any, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for closed connection: %w", ctx.Err())
	case id := <-s.closed:
		return id, nil
	}
}

// SnapshotState returns a copy of the local snapshot baseline.
func (s *ConnectionStore) SnapshotState() agentport.ConnectionSnapshotState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return agentport.ConnectionSnapshotState{
		Reconciled:  s.snapshot.Reconciled,
		Connections: maps.Clone(s.snapshot.Connections),
	}
}

// SaveSnapshotState retains a copy so callers cannot mutate the baseline.
func (s *ConnectionStore) SaveSnapshotState(state agentport.ConnectionSnapshotState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot = agentport.ConnectionSnapshotState{
		Reconciled:  state.Reconciled,
		Connections: maps.Clone(state.Connections),
	}
}

// getConnection requires the caller to hold s.mu.
func (s *ConnectionStore) getConnection(key string) (*agentmodel.Connection, error) {
	connection, ok := s.byID[key]
	if !ok {
		return nil, agentport.ErrConnectionNotFound
	}

	return &connection, nil
}
