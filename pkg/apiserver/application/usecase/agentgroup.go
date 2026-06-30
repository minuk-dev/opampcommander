package usecase

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentGroupManageUsecase manages agent groups: selector-based logical
// groupings of agents that drive group->agent remote-config propagation.
// It backs the /api/v1/agentgroups controller.
type AgentGroupManageUsecase interface {
	// GetAgentGroup returns the named group in namespace, or
	// model.ErrResourceNotExist if absent.
	GetAgentGroup(ctx context.Context, namespace string, name string,
		options *port.GetOptions) (*v1.AgentGroup, error)
	// ListAgentGroups returns a paged list of groups across namespaces.
	ListAgentGroups(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.AgentGroup], error)
	// ListAgentsByAgentGroup returns the agents whose attributes match the
	// named group's selector.
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
	// CreateAgentGroup persists a new group (namespace and name come from the
	// payload), returning model.ErrResourceAlreadyExist on a duplicate.
	CreateAgentGroup(ctx context.Context, agentGroup *v1.AgentGroup) (*v1.AgentGroup, error)
	// UpdateAgentGroup replaces the named group's spec; it is
	// optimistic-concurrency controlled (model.ErrConflict on a stale write).
	UpdateAgentGroup(ctx context.Context, namespace string, name string,
		agentGroup *v1.AgentGroup) (*v1.AgentGroup, error)
	// DeleteAgentGroup removes the named group.
	DeleteAgentGroup(ctx context.Context, namespace string, name string) error
}
