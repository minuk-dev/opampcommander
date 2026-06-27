package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.ServerConnectionPersistencePort = (*ServerConnectionRepository)(nil)

// ServerConnectionRepository is the in-memory implementation of
// [agentport.ServerConnectionPersistencePort]. It backs standalone mode, where there is
// only one server, so the cluster view is simply this process's own snapshot.
type ServerConnectionRepository struct {
	store *store[string, *agentmodel.ServerConnection]
}

// NewServerConnectionRepository creates a new in-memory ServerConnectionRepository.
func NewServerConnectionRepository() *ServerConnectionRepository {
	return &ServerConnectionRepository{
		store: newStore[string](cloneServerConnection, nil),
	}
}

// ReplaceServerConnections implements agentport.ServerConnectionPersistencePort.
func (r *ServerConnectionRepository) ReplaceServerConnections(
	_ context.Context,
	serverID string,
	conns []*agentmodel.ServerConnection,
) error {
	existing := r.store.snapshot(false, func(sc *agentmodel.ServerConnection) bool {
		return sc.ServerID == serverID
	})
	for _, sc := range existing {
		_ = r.store.delete(serverConnectionKey(sc))
	}

	for _, sc := range conns {
		r.store.put(serverConnectionKey(sc), sc)
	}

	return nil
}

// ListServerConnections implements agentport.ServerConnectionPersistencePort.
func (r *ServerConnectionRepository) ListServerConnections(
	_ context.Context,
	namespace string,
	notBefore time.Time,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	return r.store.list(options, func(sc *agentmodel.ServerConnection) bool {
		if sc.Namespace != namespace {
			return false
		}

		if !notBefore.IsZero() && sc.SnapshotAt.Before(notBefore) {
			return false
		}

		return true
	})
}

func serverConnectionKey(sc *agentmodel.ServerConnection) string {
	return sc.ServerID + "/" + sc.UID.String()
}

func cloneServerConnection(sc *agentmodel.ServerConnection) *agentmodel.ServerConnection {
	if sc == nil {
		return nil
	}

	clone := *sc

	return &clone
}
