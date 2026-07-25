package inmemory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.ServerConnectionPersistencePort = (*ServerConnectionRepository)(nil)

// ServerConnectionRepository is the in-memory implementation of
// [agentport.ServerConnectionPersistencePort]. It backs standalone mode, where there is
// only one server, so the cluster view is simply this process's own connections.
//
// Connections are stored keyed by UID (stable pagination); a heartbeats map tracks per-server
// liveness. A connection is visible only while its owning server's heartbeat is fresh.
type ServerConnectionRepository struct {
	store *store[uuid.UUID, *agentmodel.ServerConnection]

	heartbeatMu sync.RWMutex
	heartbeats  map[string]time.Time
}

// NewServerConnectionRepository creates a new in-memory ServerConnectionRepository.
func NewServerConnectionRepository() *ServerConnectionRepository {
	return &ServerConnectionRepository{
		store:       newStore[uuid.UUID](cloneServerConnection, nil),
		heartbeatMu: sync.RWMutex{},
		heartbeats:  make(map[string]time.Time),
	}
}

// SyncServerConnections implements agentport.ServerConnectionPersistencePort.
func (r *ServerConnectionRepository) SyncServerConnections(
	_ context.Context,
	serverID string,
	heartbeatAt time.Time,
	upserts []*agentmodel.ServerConnection,
	deletes []uuid.UUID,
) error {
	// Apply membership before the heartbeat, mirroring the MongoDB adapter's ordering so the
	// server only becomes visible once its connection set is in place.
	for _, conn := range upserts {
		r.store.put(conn.UID, conn)
	}

	for _, uid := range deletes {
		_ = r.store.delete(uid)
	}

	r.heartbeatMu.Lock()
	r.heartbeats[serverID] = heartbeatAt
	r.heartbeatMu.Unlock()

	return nil
}

// RemoveServer implements agentport.ServerConnectionPersistencePort.
func (r *ServerConnectionRepository) RemoveServer(_ context.Context, serverID string) error {
	r.heartbeatMu.Lock()
	delete(r.heartbeats, serverID)
	r.heartbeatMu.Unlock()

	owned := r.store.snapshot(false, func(sc *agentmodel.ServerConnection) bool {
		return sc.ServerID == serverID
	})
	for _, sc := range owned {
		_ = r.store.delete(sc.UID)
	}

	return nil
}

// ListServerConnections implements agentport.ServerConnectionPersistencePort.
func (r *ServerConnectionRepository) ListServerConnections(
	_ context.Context,
	namespace string,
	serverID string,
	notBefore time.Time,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	live := r.liveServers(notBefore)

	return r.store.list(options, func(conn *agentmodel.ServerConnection) bool {
		if conn.Namespace != namespace {
			return false
		}

		if serverID != "" && conn.ServerID != serverID {
			return false
		}

		return live[conn.ServerID]
	})
}

// liveServers returns the set of serverIDs whose heartbeat is at or after notBefore (all of
// them when notBefore is zero).
func (r *ServerConnectionRepository) liveServers(notBefore time.Time) map[string]bool {
	r.heartbeatMu.RLock()
	defer r.heartbeatMu.RUnlock()

	live := make(map[string]bool, len(r.heartbeats))

	for id, lastSeen := range r.heartbeats {
		if notBefore.IsZero() || !lastSeen.Before(notBefore) {
			live[id] = true
		}
	}

	return live
}

func cloneServerConnection(sc *agentmodel.ServerConnection) *agentmodel.ServerConnection {
	if sc == nil {
		return nil
	}

	clone := *sc

	return &clone
}
