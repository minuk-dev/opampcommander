package client

import (
	"context"
	"fmt"

	uuid "github.com/google/uuid"

	agentv1 "github.com/minuk-dev/opampcommander/api/v1/agent"
)

const (
	// ListAgentURL is the path to list all agents.
	ListAgentURL = "/api/v1/agents"
	// GetAgentURL is the path to get an agent by ID.
	GetAgentURL = "/api/v1/agents/{id}"
	// SetAgentNewInstanceUIDURL is the path to set a new instance UID for an agent.
	SetAgentNewInstanceUIDURL = "/api/v1/agents/{id}/new-instance-uid"
)

// AgentService provides methods to interact with agents.
type AgentService struct {
	service *service
}

// NewAgentService creates a new AgentService.
func NewAgentService(service *service) *AgentService {
	return &AgentService{
		service: service,
	}
}

// GetAgent retrieves an agent by its ID.
func (s *AgentService) GetAgent(ctx context.Context, id uuid.UUID) (*agentv1.Agent, error) {
	return getResource[agentv1.Agent](ctx, s.service, GetAgentURL, id.String())
}

// ListAgents lists all agents.
func (s *AgentService) ListAgents(ctx context.Context, opts ...ListOption) (*agentv1.ListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[agentv1.Agent](
		ctx,
		s.service,
		ListAgentURL,
		ListSettings{
			limit:         listSettings.limit,
			continueToken: listSettings.continueToken,
		},
	)
}

// SetAgentNewInstanceUID sets a new instance UID for an agent.
func (s *AgentService) SetAgentNewInstanceUID(
	ctx context.Context,
	id uuid.UUID,
	request agentv1.SetNewInstanceUIDRequest,
) (*agentv1.Agent, error) {
	var result agentv1.Agent

	response, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("id", id.String()).
		SetBody(request).
		SetResult(&result).
		Put(SetAgentNewInstanceUIDURL)
	if err != nil {
		return nil, fmt.Errorf("failed to set new instance UID for agent: %w", err)
	}

	if response.IsError() {
		return nil, &ResponseError{
			StatusCode:   response.StatusCode(),
			ErrorMessage: response.String(),
		}
	}

	return &result, nil
}
