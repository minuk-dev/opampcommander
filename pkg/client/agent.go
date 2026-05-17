package client

import (
	"context"
	"fmt"
	"time"

	uuid "github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentURL is the path to list all agents in a namespace.
	ListAgentURL = "/api/v1/namespaces/{namespace}/agents"
	// SearchAgentURL is the path to search agents in a namespace.
	SearchAgentURL = "/api/v1/namespaces/{namespace}/agents/search"
	// GetAgentURL is the path to get an agent by ID in a namespace.
	GetAgentURL = "/api/v1/namespaces/{namespace}/agents/{id}"
	// UpdateAgentURL is the path to update an agent in a namespace.
	UpdateAgentURL = "/api/v1/namespaces/{namespace}/agents/{id}"
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

// GetAgent retrieves an agent by its namespace and ID.
func (s *AgentService) GetAgent(
	ctx context.Context,
	namespace string,
	id uuid.UUID,
) (*v1.Agent, error) {
	var result v1.Agent

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", id.String()).
		SetResult(&result).
		Get(GetAgentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get agent: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// AgentListResponse represents a list of agents with metadata.
type AgentListResponse = v1.ListResponse[v1.Agent]

// ListAgents lists all agents in a namespace.
func (s *AgentService) ListAgents(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*AgentListResponse, error) {
	listSettings := newListSettings(opts)

	var result AgentListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetResult(&result)
	listSettings.applyTo(req)

	response, err := req.Get(ListAgentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	if response.IsError() {
		return nil, &ResponseError{
			StatusCode:   response.StatusCode(),
			ErrorMessage: response.String(),
		}
	}

	return &result, nil
}

// SearchAgents searches agents by query in a namespace.
func (s *AgentService) SearchAgents(
	ctx context.Context,
	namespace string,
	query string,
	opts ...ListOption,
) (*AgentListResponse, error) {
	listSettings := newListSettings(opts)

	var result AgentListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetQueryParam("q", query).
		SetResult(&result)
	listSettings.applyTo(req)

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

// UpdateAgent updates an agent with the given spec in a namespace.
func (s *AgentService) UpdateAgent(
	ctx context.Context,
	namespace string,
	id uuid.UUID,
	agent *v1.Agent,
) (*v1.Agent, error) {
	var result v1.Agent

	response, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
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
	namespace string,
	id uuid.UUID,
	request SetNewInstanceUIDRequest,
) (*v1.Agent, error) {
	//exhaustruct:ignore
	agent := &v1.Agent{
		Spec: v1.AgentSpec{
			NewInstanceUID: request.NewInstanceUID.String(),
		},
	}

	return s.UpdateAgent(ctx, namespace, id, agent)
}

// RestartAgent restarts an agent by its ID.
func (s *AgentService) RestartAgent(
	ctx context.Context,
	namespace string,
	id uuid.UUID,
) (*v1.Agent, error) {
	now := v1.NewTime(time.Now())
	//exhaustruct:ignore
	agent := &v1.Agent{
		Spec: v1.AgentSpec{
			RestartRequiredAt: &now,
		},
	}

	return s.UpdateAgent(ctx, namespace, id, agent)
}
