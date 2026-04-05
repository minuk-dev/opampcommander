//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"
	"strconv"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentPackageURL is the path to list all agent packages.
	ListAgentPackageURL = "/api/v1/namespaces/{namespace}/agentpackages"
	// GetAgentPackageURL is the path to get an agent package by name.
	GetAgentPackageURL = "/api/v1/namespaces/{namespace}/agentpackages/{id}"
	// CreateAgentPackageURL is the path to create a new agent package.
	CreateAgentPackageURL = "/api/v1/namespaces/{namespace}/agentpackages"
	// UpdateAgentPackageURL is the path to update an existing agent package.
	UpdateAgentPackageURL = "/api/v1/namespaces/{namespace}/agentpackages/{id}"
	// DeleteAgentPackageURL is the path to delete an agent package.
	DeleteAgentPackageURL = "/api/v1/namespaces/{namespace}/agentpackages/{id}"
)

// AgentPackageService provides methods to interact with agent packages.
type AgentPackageService struct {
	service *service
}

// NewAgentPackageService creates a new AgentPackageService.
func NewAgentPackageService(service *service) *AgentPackageService {
	return &AgentPackageService{
		service: service,
	}
}

// GetAgentPackage retrieves an agent package by its namespace and name.
func (s *AgentPackageService) GetAgentPackage(
	ctx context.Context,
	namespace string,
	name string,
) (*v1.AgentPackage, error) {
	var agentPackage v1.AgentPackage

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&agentPackage).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Get(GetAgentPackageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent package(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get agent package(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &agentPackage, nil
}

// AgentPackageListResponse represents a list of agent packages with metadata.
type AgentPackageListResponse = v1.ListResponse[v1.AgentPackage]

// ListAgentPackages lists all agent packages in a namespace.
func (s *AgentPackageService) ListAgentPackages(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*AgentPackageListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	var listResponse AgentPackageListResponse

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

	res, err := req.Get(ListAgentPackageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent packages(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list agent packages(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &listResponse, nil
}

// CreateAgentPackage creates a new agent package.
func (s *AgentPackageService) CreateAgentPackage(
	ctx context.Context,
	namespace string,
	createRequest *v1.AgentPackage,
) (*v1.AgentPackage, error) {
	var result v1.AgentPackage

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetBody(createRequest).
		SetResult(&result).
		Post(CreateAgentPackageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent package(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to create agent package(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// UpdateAgentPackage updates an existing agent package.
func (s *AgentPackageService) UpdateAgentPackage(
	ctx context.Context,
	updateRequest *v1.AgentPackage,
) (*v1.AgentPackage, error) {
	var result v1.AgentPackage

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", updateRequest.Metadata.Namespace).
		SetPathParam("id", updateRequest.Metadata.Name).
		SetBody(updateRequest).
		SetResult(&result).
		Put(UpdateAgentPackageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent package(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to update agent package(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// DeleteAgentPackage deletes an agent package by its namespace and name.
func (s *AgentPackageService) DeleteAgentPackage(
	ctx context.Context,
	namespace string,
	name string,
) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Delete(DeleteAgentPackageURL)
	if err != nil {
		return fmt.Errorf("failed to delete agent package(restyError): %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete agent package(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}
