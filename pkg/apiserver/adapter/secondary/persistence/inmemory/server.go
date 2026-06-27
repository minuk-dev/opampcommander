package inmemory

import (
	"context"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.ServerPersistencePort = (*ServerRepository)(nil)

// ServerRepository is the in-memory implementation of
// [agentport.ServerPersistencePort]. Servers have no soft-delete concept.
type ServerRepository struct {
	store *store[string, *agentmodel.Server]
}

// NewServerRepository creates a new in-memory ServerRepository.
func NewServerRepository() *ServerRepository {
	return &ServerRepository{
		store: newStore[string]((*agentmodel.Server).Clone, nil),
	}
}

// GetServer implements agentport.ServerPersistencePort.
func (r *ServerRepository) GetServer(_ context.Context, id string) (*agentmodel.Server, error) {
	return r.store.get(id, nil)
}

// PutServer implements agentport.ServerPersistencePort.
func (r *ServerRepository) PutServer(_ context.Context, server *agentmodel.Server) error {
	r.store.put(server.ID, server)

	return nil
}

// ListServers implements agentport.ServerPersistencePort.
func (r *ServerRepository) ListServers(_ context.Context) ([]*agentmodel.Server, error) {
	return r.store.snapshot(false, nil), nil
}
