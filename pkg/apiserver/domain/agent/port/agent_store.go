package agentport

import (
	"context"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// AgentStore owns durable agent access and the local read cache. Fresh reads
// bypass the cache for decisions that must observe other writers.
type AgentStore interface {
	AgentPersistencePort
	AgentCacheInvalidator
	GetAgentFresh(ctx context.Context, uid uuid.UUID) (*agentmodel.Agent, error)
}
