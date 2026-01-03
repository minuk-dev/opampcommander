package client

import (
	"context"
	"fmt"
	"strconv"

	uuid "github.com/google/uuid"

	agentv1 "github.com/minuk-dev/opampcommander/api/v1/agent"
)

const (
	// ListAgentURL is the path to list all agents.
	ListAgentURL = "/api/v1/agents"
	// SearchAgentURL is the path to search agents.
	SearchAgentURL = "/api/v1/agents/search"
	// GetAgentURL is the path to get an agent by ID.
	GetAgentURL = "/api/v1/agents/{id}"
	// SetAgentNewInstanceUIDURL is the path to set a new instance UID for an agent.
	SetAgentNewInstanceUIDURL = "/api/v1/agents/{id}/new-instance-uid"
	// SetAgentConnectionSettingsURL is the path to set connection settings for an agent.
	SetAgentConnectionSettingsURL = "/api/v1/agents/{id}/connection-settings"
	// RestartAgentURL is the path to restart an agent.
	RestartAgentURL = "/api/v1/agents/{id}/restart"
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

// SearchAgents searches agents by query.
func (s *AgentService) SearchAgents(
	ctx context.Context,
	query string,
	opts ...ListOption,
) (*agentv1.ListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	var result agentv1.ListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetQueryParam("q", query).
		SetResult(&result)

	if listSettings.limit != nil && *listSettings.limit > 0 {
		req.SetQueryParam("limit", strconv.Itoa(*listSettings.limit))
	}

	if listSettings.continueToken != nil && *listSettings.continueToken != "" {
		req.SetQueryParam("continue", *listSettings.continueToken)
	}

	response, err := req.Get(SearchAgentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	if response.IsError() {
		return nil, &ResponseError{
			StatusCode:   response.StatusCode(),
			ErrorMessage: response.String(),
		}
	}

	return &result, nil
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

// RestartAgent restarts an agent by its ID.
func (s *AgentService) RestartAgent(ctx context.Context, id uuid.UUID) error {
	response, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("id", id.String()).
		Post(RestartAgentURL)
	if err != nil {
		return fmt.Errorf("failed to restart agent: %w", err)
	}

	if response.IsError() {
		return &ResponseError{
			StatusCode:   response.StatusCode(),
			ErrorMessage: response.String(),
		}
	}

	return nil
}
