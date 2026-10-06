package agentport

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// ServerStore owns the server read cache. A cached heartbeat outside the supplied
// liveness window is discarded so a fresh heartbeat can be read from persistence.
type ServerStore interface {
	GetServer(ctx context.Context, id string, now time.Time, timeout time.Duration) (*agentmodel.Server, error)
	GetServerFresh(ctx context.Context, id string) (*agentmodel.Server, error)
	ListServers(ctx context.Context) ([]*agentmodel.Server, error)
}
