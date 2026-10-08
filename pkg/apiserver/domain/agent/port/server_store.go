package agentport

import (
	"context"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	domainport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

// ServerStore owns read-only registry access and its local cache. Implementations
// discard cached dead servers so newer persisted heartbeats can be discovered.
type ServerStore interface {
	domainport.Reader[string, *agentmodel.Server]
	GetFresh(ctx context.Context, id string) (*agentmodel.Server, error)
	ListServers(ctx context.Context) ([]*agentmodel.Server, error)
}
