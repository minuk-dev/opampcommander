package client

import (
	"context"
	"fmt"
	"strconv"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentGroupURL is the path to list all agent groups.
	ListAgentGroupURL = "/api/v1/agentgroups"
	// ListAgentsByAgentGroupURL is the path to list all agents in an agent group.
	ListAgentsByAgentGroupURL = "/api/v1/agentgroups/{name}/agents"
	// GetAgentGroupURL is the path to get an agent group by ID.
	GetAgentGroupURL = "/api/v1/agentgroups/{id}"
	// CreateAgentGroupURL is the path to create a new agent group.
	CreateAgentGroupURL = "/api/v1/agentgroups"
	// UpdateAgentGroupURL is the path to update an existing agent group.
	UpdateAgentGroupURL = "/api/v1/agentgroups/{id}"
	// DeleteAgentGroupURL is the path to delete an agent group.
	DeleteAgentGroupURL = "/api/v1/agentgroups/{id}"
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
) (*v1.AgentGroup, error) {
	return getResource[v1.AgentGroup](ctx, s.service, GetAgentGroupURL, name)
}

// AgentGroupListResponse represents a list of agent groups with metadata.
type AgentGroupListResponse = v1.ListResponse[v1.AgentGroup]

// ListAgentGroups lists all agent groups.
func (s *AgentGroupService) ListAgentGroups(
	ctx context.Context,
	opts ...ListOption,
) (*AgentGroupListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[v1.AgentGroup](
		ctx,
		s.service,
		ListAgentGroupURL,
		ListSettings{
			limit:         listSettings.limit,
			continueToken: listSettings.continueToken,
		},
	)
}

// ListAgentsByAgentGroup lists agents belonging to a specific agent group.
func (s *AgentGroupService) ListAgentsByAgentGroup(
	ctx context.Context,
	name string,
	opts ...ListOption,
) (*AgentListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	var listResponse AgentListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse)

	if listSettings.limit != nil {
		req.SetQueryParam("limit", strconv.Itoa(*listSettings.limit))
	}

	if listSettings.continueToken != nil {
		req.SetQueryParam("continue", *listSettings.continueToken)
	}

	req.SetPathParam("name", name)

	res, err := req.Get(ListAgentsByAgentGroupURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list resources(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to list resources: %w", ErrEmptyResponse)
	}

	return &listResponse, nil
}

// CreateAgentGroup creates a new agent group.
func (s *AgentGroupService) CreateAgentGroup(
	ctx context.Context,
	createRequest *v1.AgentGroup,
) (*v1.AgentGroup, error) {
	return createResource[v1.AgentGroup, v1.AgentGroup](
		ctx,
		s.service,
		CreateAgentGroupURL,
		createRequest,
	)
}

// UpdateAgentGroup updates an existing agent group.
func (s *AgentGroupService) UpdateAgentGroup(
	ctx context.Context,
	updateRequest *v1.AgentGroup,
) (*v1.AgentGroup, error) {
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
