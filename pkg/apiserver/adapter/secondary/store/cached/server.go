package cached

import (
	"context"
	"fmt"
	"time"

	"github.com/jellydator/ttlcache/v3"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.ServerStore = (*ServerStore)(nil)

const (
	defaultServerCacheTTL      = 30 * time.Second
	defaultServerCacheCapacity = 100
)

// ServerStore owns the bounded server read cache over durable persistence.
type ServerStore struct {
	persistence agentport.ServerPersistencePort
	cache       *ttlcache.Cache[string, *agentmodel.Server]
}

// NewServerStore creates a server store with the existing TTL and capacity.
func NewServerStore(persistence agentport.ServerPersistencePort) *ServerStore {
	return &ServerStore{
		persistence: persistence,
		cache: ttlcache.New[string, *agentmodel.Server](
			ttlcache.WithTTL[string, *agentmodel.Server](defaultServerCacheTTL),
			ttlcache.WithCapacity[string, *agentmodel.Server](defaultServerCacheCapacity),
		),
	}
}

// GetServer discards cached dead servers so a fresh heartbeat can be discovered.
func (s *ServerStore) GetServer(
	ctx context.Context, id string, now time.Time, timeout time.Duration,
) (*agentmodel.Server, error) {
	if item := s.cache.Get(id); item != nil {
		server := item.Value()
		if server.IsAlive(now, timeout) {
			return server.Clone(), nil
		}

		s.cache.Delete(id)
	}

	server, err := s.GetServerFresh(ctx, id)
	if err != nil {
		return nil, err
	}

	s.cache.Set(id, server.Clone(), ttlcache.DefaultTTL)

	return server, nil
}

// GetServerFresh bypasses the local cache for direct delivery lookups.
func (s *ServerStore) GetServerFresh(ctx context.Context, id string) (*agentmodel.Server, error) {
	server, err := s.persistence.GetServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read server: %w", err)
	}

	return server, nil
}

// ListServers reads the complete registry directly from persistence.
func (s *ServerStore) ListServers(ctx context.Context) ([]*agentmodel.Server, error) {
	servers, err := s.persistence.ListServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}

	return servers, nil
}

// Shutdown releases the local cache.
func (s *ServerStore) Shutdown() {
	s.cache.DeleteAll()
	s.cache.Stop()
}
