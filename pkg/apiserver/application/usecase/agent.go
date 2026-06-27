package usecase

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentManageUsecase is the central use case for managing OpAMP-managed
// agents. It backs the /api/v1/agents controller with read, search,
// desired-state update and delete operations. Agents are scoped by their
// reported service.namespace; an agent that lives in another namespace is
// reported as not found from the caller's namespace.
type AgentManageUsecase interface {
	// GetAgent returns the agent identified by instanceUID within namespace. It
	// yields model.ErrResourceNotExist when the agent is unknown and
	// ErrAgentNamespaceMismatch when it belongs to another namespace (both 404).
	GetAgent(ctx context.Context, namespace string, instanceUID uuid.UUID) (*v1.Agent, error)
	// ListAgents returns a paged list of agents in namespace. options may restrict
	// the result to connected agents or to matching identifying/non-identifying
	// attributes.
	ListAgents(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
	// SearchAgents is ListAgents narrowed by a free-text query over the agent's
	// attributes.
	SearchAgents(ctx context.Context, namespace string, query string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
	// UpdateAgent applies a desired-state change to the agent (e.g. linking a
	// remote config). It is optimistic-concurrency controlled and returns
	// model.ErrConflict if the stored agent changed since it was read.
	UpdateAgent(ctx context.Context, namespace string, instanceUID uuid.UUID,
		agent *v1.Agent) (*v1.Agent, error)
	// DeleteAgent removes a disconnected agent. It returns ErrAgentConnected if
	// the agent still holds a live connection: only disconnected agents may be
	// deleted.
	DeleteAgent(ctx context.Context, namespace string, instanceUID uuid.UUID) error
	// ListAgentEndpoints returns a read-only view of the endpoints the agent exports
	// to, extracted from its reported effective configuration (not persisted).
	ListAgentEndpoints(ctx context.Context, namespace string,
		instanceUID uuid.UUID) (*v1.ListResponse[v1.Endpoint], error)
}
