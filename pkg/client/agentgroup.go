//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"
	"strconv"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentGroupURL is the path to list all agent groups within a namespace.
	ListAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups"
	// ListAgentsByAgentGroupURL is the path to list all agents in an agent group.
	ListAgentsByAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{name}/agents"
	// GetAgentGroupURL is the path to get an agent group by namespace and name.
	GetAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{id}"
	// CreateAgentGroupURL is the path to create a new agent group.
	CreateAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups"
	// UpdateAgentGroupURL is the path to update an existing agent group.
	UpdateAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{id}"
	// DeleteAgentGroupURL is the path to delete an agent group.
	DeleteAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{id}"
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

// GetAgentGroup retrieves an agent group by its namespace and name.
func (s *AgentGroupService) GetAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	opts ...GetOption,
) (*v1.AgentGroup, error) {
	var getSettings GetSettings
	for _, opt := range opts {
		opt.Apply(&getSettings)
	}

	var agentGroup v1.AgentGroup

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&agentGroup).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name)

	if getSettings.includeDeleted != nil && *getSettings.includeDeleted {
		req.SetQueryParam("includeDeleted", "true")
	}

	res, err := req.Get(GetAgentGroupURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent group(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get agent group(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &agentGroup, nil
}

// AgentGroupListResponse represents a list of agent groups with metadata.
type AgentGroupListResponse = v1.ListResponse[v1.AgentGroup]

// ListAgentGroups lists all agent groups in a namespace.
func (s *AgentGroupService) ListAgentGroups(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*AgentGroupListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	var listResponse AgentGroupListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse).
		SetPathParam("namespace", namespace)

	if listSettings.limit != nil {
		req.SetQueryParam("limit", strconv.Itoa(*listSettings.limit))
	}

	if listSettings.continueToken != nil {
		req.SetQueryParam("continue", *listSettings.continueToken)
	}

	if listSettings.includeDeleted != nil && *listSettings.includeDeleted {
		req.SetQueryParam("includeDeleted", "true")
	}

	res, err := req.Get(ListAgentGroupURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent groups(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list agent groups(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &listResponse, nil
}

// ListAgentsByAgentGroup lists agents belonging to a specific agent group.
func (s *AgentGroupService) ListAgentsByAgentGroup(
	ctx context.Context,
	namespace string,
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

	req.SetPathParam("namespace", namespace)
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
	namespace string,
	createRequest *v1.AgentGroup,
) (*v1.AgentGroup, error) {
	var result v1.AgentGroup

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetBody(createRequest).
		SetResult(&result).
		Post(CreateAgentGroupURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent group(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to create agent group(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// UpdateAgentGroup updates an existing agent group.
func (s *AgentGroupService) UpdateAgentGroup(
	ctx context.Context,
	updateRequest *v1.AgentGroup,
) (*v1.AgentGroup, error) {
	var result v1.AgentGroup

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", updateRequest.Metadata.Namespace).
		SetPathParam("id", updateRequest.Metadata.Name).
		SetBody(updateRequest).
		SetResult(&result).
		Put(UpdateAgentGroupURL)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent group(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to update agent group(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// DeleteAgentGroup deletes an agent group by its namespace and name.
func (s *AgentGroupService) DeleteAgentGroup(ctx context.Context, namespace string, name string) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Delete(DeleteAgentGroupURL)
	if err != nil {
		return fmt.Errorf("failed to delete agent group(restyError): %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete agent group(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}
