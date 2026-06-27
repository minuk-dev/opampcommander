package usecase

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentGroupManageUsecase is a use case that handles agent group management operations.
type AgentGroupManageUsecase interface {
	GetAgentGroup(ctx context.Context, namespace string, name string,
		options *port.GetOptions) (*v1.AgentGroup, error)
	ListAgentGroups(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.AgentGroup], error)
	ListAgentsByAgentGroup(
		ctx context.Context,
		namespace string,
		agentGroupName string,
		options *port.ListOptions,
	) (*v1.ListResponse[v1.Agent], error)
	// ListAgentGroupsByAgent lists the agent groups in the given namespace whose selector
	// matches the agent identified by instanceUID.
	ListAgentGroupsByAgent(
		ctx context.Context,
		namespace string,
		instanceUID uuid.UUID,
	) (*v1.ListResponse[v1.AgentGroup], error)
	CreateAgentGroup(ctx context.Context, agentGroup *v1.AgentGroup) (*v1.AgentGroup, error)
	UpdateAgentGroup(ctx context.Context, namespace string, name string,
		agentGroup *v1.AgentGroup) (*v1.AgentGroup, error)
	DeleteAgentGroup(ctx context.Context, namespace string, name string) error
}
