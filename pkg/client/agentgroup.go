package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentGroupURL is the path to list all agent groups within a namespace.
	ListAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups"
	// ListAgentsByAgentGroupURL is the path to list all agents in an agent group.
	ListAgentsByAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{name}/agents"
	// ListAgentGroupsByAgentURL is the path to list all agent groups that contain an agent.
	ListAgentGroupsByAgentURL = "/api/v1/namespaces/{namespace}/agents/{id}/agentgroups"
	// GetAgentGroupURL is the path to get an agent group by namespace and name.
	GetAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{id}"
	// CreateAgentGroupURL is the path to create a new agent group.
	CreateAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups"
	// UpdateAgentGroupURL is the path to update an existing agent group.
	UpdateAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{id}"
	// DeleteAgentGroupURL is the path to delete an agent group.
	DeleteAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{id}"
	// ReconcileAgentGroupURL is the path to reconcile an agent group by name in a namespace.
	ReconcileAgentGroupURL = "/api/v1/namespaces/{namespace}/agentgroups/{id}/reconcile"
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
	getSettings := newGetSettings(opts)

	var agentGroup v1.AgentGroup

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&agentGroup).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name)
	getSettings.applyTo(req)

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
	listSettings := newListSettings(opts)

	var listResponse AgentGroupListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse).
		SetPathParam("namespace", namespace)
	listSettings.applyTo(req)

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
	listSettings := newListSettings(opts)

	var listResponse AgentListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse).
		SetPathParam("namespace", namespace).
		SetPathParam("name", name)
	listSettings.applyTo(req)

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

// ListAgentGroupsByAgent lists the agent groups in the namespace whose selector matches
// the agent identified by agentID.
func (s *AgentGroupService) ListAgentGroupsByAgent(
	ctx context.Context,
	namespace string,
	agentID string,
) (*AgentGroupListResponse, error) {
	var listResponse AgentGroupListResponse

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse).
		SetPathParam("namespace", namespace).
		SetPathParam("id", agentID).
		Get(ListAgentGroupsByAgentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent groups by agent(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list agent groups by agent(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
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

// ReconcileAgentGroup re-applies the named agent group to its matching agents on demand.
func (s *AgentGroupService) ReconcileAgentGroup(ctx context.Context, namespace string, name string) error {
	// Reconcile runs synchronously on the server (re-applying the group to all its agents),
	// which can outlast the shared client's 15s timeout in a large namespace. Clone the client
	// and clear the timeout; the context deadline is the only limit.
	res, err := s.service.Resty.Clone().SetTimeout(0).R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Post(ReconcileAgentGroupURL)
	if err != nil {
		return fmt.Errorf("failed to reconcile agent group(restyError): %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to reconcile agent group(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}
