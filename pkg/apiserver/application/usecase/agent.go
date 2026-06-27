package usecase

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentManageUsecase is a use case that handles agent management operations.
type AgentManageUsecase interface {
	GetAgent(ctx context.Context, namespace string, instanceUID uuid.UUID) (*v1.Agent, error)
	ListAgents(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
	SearchAgents(ctx context.Context, namespace string, query string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
	UpdateAgent(ctx context.Context, namespace string, instanceUID uuid.UUID,
		agent *v1.Agent) (*v1.Agent, error)
	DeleteAgent(ctx context.Context, namespace string, instanceUID uuid.UUID) error
	// ListAgentEndpoints returns a read-only view of the endpoints the agent exports
	// to, extracted from its reported effective configuration (not persisted).
	ListAgentEndpoints(ctx context.Context, namespace string,
		instanceUID uuid.UUID) (*v1.ListResponse[v1.Endpoint], error)
}
