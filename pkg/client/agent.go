package client

import (
	"context"
	"fmt"
	"strconv"
	"time"

	uuid "github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentURL is the path to list all agents.
	ListAgentURL = "/api/v1/agents"
	// SearchAgentURL is the path to search agents.
	SearchAgentURL = "/api/v1/agents/search"
	// GetAgentURL is the path to get an agent by ID.
	GetAgentURL = "/api/v1/agents/{id}"
	// UpdateAgentURL is the path to update an agent.
	UpdateAgentURL = "/api/v1/agents/{id}"
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
func (s *AgentService) GetAgent(ctx context.Context, id uuid.UUID) (*v1.Agent, error) {
	return getResource[v1.Agent](ctx, s.service, GetAgentURL, id.String())
}

// AgentListResponse represents a list of agents with metadata.
type AgentListResponse = v1.ListResponse[v1.Agent]

// ListAgents lists all agents.
func (s *AgentService) ListAgents(ctx context.Context, opts ...ListOption) (*AgentListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[v1.Agent](
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
) (*AgentListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	var result AgentListResponse

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

// SetNewInstanceUIDRequest is a struct that represents the request to set a new instance UID for the agent.
type SetNewInstanceUIDRequest struct {
	NewInstanceUID uuid.UUID `binding:"required" json:"newInstanceUid"`
}

// UpdateAgent updates an agent with the given spec.
func (s *AgentService) UpdateAgent(ctx context.Context, id uuid.UUID, agent *v1.Agent) (*v1.Agent, error) {
	var result v1.Agent

	response, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("id", id.String()).
		SetBody(agent).
		SetResult(&result).
		Put(UpdateAgentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
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
	request SetNewInstanceUIDRequest,
) (*v1.Agent, error) {
	//exhaustruct:ignore
	agent := &v1.Agent{
		Spec: v1.AgentSpec{
			NewInstanceUID: request.NewInstanceUID.String(),
		},
	}

	return s.UpdateAgent(ctx, id, agent)
}

// RestartAgent restarts an agent by its ID.
func (s *AgentService) RestartAgent(ctx context.Context, id uuid.UUID) (*v1.Agent, error) {
	now := v1.NewTime(time.Now())
	//exhaustruct:ignore
	agent := &v1.Agent{
		Spec: v1.AgentSpec{
			RestartRequiredAt: &now,
		},
	}

	return s.UpdateAgent(ctx, id, agent)
}
