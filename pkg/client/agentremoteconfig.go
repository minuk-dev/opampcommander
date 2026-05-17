//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"
	"strconv"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentRemoteConfigURL is the path to list all agent remote configs.
	ListAgentRemoteConfigURL = "/api/v1/namespaces/{namespace}/agentremoteconfigs"
	// GetAgentRemoteConfigURL is the path to get an agent remote config by name.
	GetAgentRemoteConfigURL = "/api/v1/namespaces/{namespace}/agentremoteconfigs/{id}"
	// CreateAgentRemoteConfigURL is the path to create a new agent remote config.
	CreateAgentRemoteConfigURL = "/api/v1/namespaces/{namespace}/agentremoteconfigs"
	// UpdateAgentRemoteConfigURL is the path to update an existing agent remote config.
	UpdateAgentRemoteConfigURL = "/api/v1/namespaces/{namespace}/agentremoteconfigs/{id}"
	// DeleteAgentRemoteConfigURL is the path to delete an agent remote config.
	DeleteAgentRemoteConfigURL = "/api/v1/namespaces/{namespace}/agentremoteconfigs/{id}"
)

// AgentRemoteConfigService provides methods to interact with agent remote configs.
type AgentRemoteConfigService struct {
	service *service
}

// NewAgentRemoteConfigService creates a new AgentRemoteConfigService.
func NewAgentRemoteConfigService(service *service) *AgentRemoteConfigService {
	return &AgentRemoteConfigService{
		service: service,
	}
}

// GetAgentRemoteConfig retrieves an agent remote config by its namespace and name.
func (s *AgentRemoteConfigService) GetAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	opts ...GetOption,
) (*v1.AgentRemoteConfig, error) {
	var getSettings GetSettings
	for _, opt := range opts {
		opt.Apply(&getSettings)
	}

	var agentRemoteConfig v1.AgentRemoteConfig

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&agentRemoteConfig).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name)

	if getSettings.includeDeleted != nil && *getSettings.includeDeleted {
		req.SetQueryParam("includeDeleted", "true")
	}

	res, err := req.Get(GetAgentRemoteConfigURL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get agent remote config(restyError): %w", err,
		)
	}

	if res.IsError() {
		return nil, fmt.Errorf(
			"failed to get agent remote config(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &agentRemoteConfig, nil
}

// AgentRemoteConfigListResponse represents a list of agent remote configs with metadata.
type AgentRemoteConfigListResponse = v1.ListResponse[v1.AgentRemoteConfig]

// ListAgentRemoteConfigs lists all agent remote configs in a namespace.
func (s *AgentRemoteConfigService) ListAgentRemoteConfigs(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*AgentRemoteConfigListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	var listResponse AgentRemoteConfigListResponse

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

	res, err := req.Get(ListAgentRemoteConfigURL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list agent remote configs(restyError): %w", err,
		)
	}

	if res.IsError() {
		return nil, fmt.Errorf(
			"failed to list agent remote configs(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &listResponse, nil
}

// CreateAgentRemoteConfig creates a new agent remote config.
func (s *AgentRemoteConfigService) CreateAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	createRequest *v1.AgentRemoteConfig,
) (*v1.AgentRemoteConfig, error) {
	var result v1.AgentRemoteConfig

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetBody(createRequest).
		SetResult(&result).
		Post(CreateAgentRemoteConfigURL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create agent remote config(restyError): %w", err,
		)
	}

	if res.IsError() {
		return nil, fmt.Errorf(
			"failed to create agent remote config(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &result, nil
}

// UpdateAgentRemoteConfig updates an existing agent remote config.
func (s *AgentRemoteConfigService) UpdateAgentRemoteConfig(
	ctx context.Context,
	updateRequest *v1.AgentRemoteConfig,
) (*v1.AgentRemoteConfig, error) {
	var result v1.AgentRemoteConfig

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", updateRequest.Metadata.Namespace).
		SetPathParam("id", updateRequest.Metadata.Name).
		SetBody(updateRequest).
		SetResult(&result).
		Put(UpdateAgentRemoteConfigURL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to update agent remote config(restyError): %w", err,
		)
	}

	if res.IsError() {
		return nil, fmt.Errorf(
			"failed to update agent remote config(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return &result, nil
}

// DeleteAgentRemoteConfig deletes an agent remote config by namespace and name.
func (s *AgentRemoteConfigService) DeleteAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Delete(DeleteAgentRemoteConfigURL)
	if err != nil {
		return fmt.Errorf(
			"failed to delete agent remote config(restyError): %w", err,
		)
	}

	if res.IsError() {
		return fmt.Errorf(
			"failed to delete agent remote config(responseError): %w",
			&ResponseError{
				StatusCode:   res.StatusCode(),
				ErrorMessage: res.String(),
			},
		)
	}

	return nil
}
