package agentport

import (
	"context"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	domainport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

// AgentStore owns durable agent access and the local read cache. Fresh reads
// bypass the cache for decisions that must observe other writers.
type AgentStore interface {
	domainport.Store[uuid.UUID, *agentmodel.Agent]
	AgentCacheInvalidator
	GetFresh(ctx context.Context, uid uuid.UUID) (*agentmodel.Agent, error)
	UpdateAgentLiveness(ctx context.Context, liveness *agentmodel.AgentLiveness) error
	ListAgents(ctx context.Context, namespace string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
	ListAgentsBySelector(ctx context.Context, selector agentmodel.AgentSelector,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
	SearchAgents(ctx context.Context, namespace, query string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
}
