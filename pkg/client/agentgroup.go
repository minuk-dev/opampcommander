package client

import (
	"context"

	agentgroupv1 "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
)

const (
	// ListAgentGroupURL is the path to list all agent groups.
	ListAgentGroupURL = "/api/v1/agentgroups"
	// GetAgentGroupURL is the path to get an agent group by ID.
	GetAgentGroupURL = "/api/v1/agentgroups/:id"
	// CreateAgentGroupURL is the path to create a new agent group.
	CreateAgentGroupURL = "/api/v1/agentgroups"
	// UpdateAgentGroupURL is the path to update an existing agent group.
	UpdateAgentGroupURL = "/api/v1/agentgroups/:id"
	// DeleteAgentGroupURL is the path to delete an agent group.
	DeleteAgentGroupURL = "/api/v1/agentgroups/:id"
)

// AgentGroupService provides methods to interact with agent groups.
type AgentGroupService struct {
	service *service
}

// NewAgentGroupService creates a new AgentGroupService.
func NewAgentGroupService(service *service) *AgentGroupService {
	return &AgentGroupService{
		service: service,
	}
}

// GetAgentGroup retrieves an agent group by its ID.
func (s *AgentGroupService) GetAgentGroup(
	ctx context.Context,
	name string,
) (*agentgroupv1.AgentGroup, error) {
	return getResource[agentgroupv1.AgentGroup](ctx, s.service, GetAgentGroupURL, name)
}

// ListAgentGroups lists all agent groups.
func (s *AgentGroupService) ListAgentGroups(
	ctx context.Context,
	opts ...ListOption,
) (*agentgroupv1.ListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[agentgroupv1.AgentGroup](
		ctx,
		s.service,
		ListAgentGroupURL,
		ListSettings{
			limit:         listSettings.limit,
			continueToken: listSettings.continueToken,
		},
	)
}

// CreateAgentGroup creates a new agent group.
func (s *AgentGroupService) CreateAgentGroup(
	ctx context.Context,
	createRequest *agentgroupv1.CreateRequest,
) (*agentgroupv1.AgentGroup, error) {
	return createResource[agentgroupv1.CreateRequest, agentgroupv1.AgentGroup](
		ctx,
		s.service,
		CreateAgentGroupURL,
		createRequest,
	)
}

// UpdateAgentGroup updates an existing agent group.
func (s *AgentGroupService) UpdateAgentGroup(
	ctx context.Context,
	updateRequest *agentgroupv1.AgentGroup,
) (*agentgroupv1.AgentGroup, error) {
	return updateResource(
		ctx,
		s.service,
		UpdateAgentGroupURL,
		updateRequest,
	)
}

// DeleteAgentGroup deletes an agent group by its ID.
func (s *AgentGroupService) DeleteAgentGroup(ctx context.Context, name string) error {
	return deleteResource(ctx, s.service, DeleteAgentGroupURL, name)
}
