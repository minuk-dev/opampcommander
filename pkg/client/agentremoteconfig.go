//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentRemoteConfigURL is the path to list all agent remote configs.
	ListAgentRemoteConfigURL = "/api/v1/agentremoteconfigs"
	// GetAgentRemoteConfigURL is the path to get an agent remote config by name.
	GetAgentRemoteConfigURL = "/api/v1/agentremoteconfigs/{id}"
	// CreateAgentRemoteConfigURL is the path to create a new agent remote config.
	CreateAgentRemoteConfigURL = "/api/v1/agentremoteconfigs"
	// UpdateAgentRemoteConfigURL is the path to update an existing agent remote config.
	UpdateAgentRemoteConfigURL = "/api/v1/agentremoteconfigs/{id}"
	// DeleteAgentRemoteConfigURL is the path to delete an agent remote config.
	DeleteAgentRemoteConfigURL = "/api/v1/agentremoteconfigs/{id}"
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

// GetAgentRemoteConfig retrieves an agent remote config by its name.
func (s *AgentRemoteConfigService) GetAgentRemoteConfig(
	ctx context.Context,
	name string,
) (*v1.AgentRemoteConfig, error) {
	return getResource[v1.AgentRemoteConfig](ctx, s.service, GetAgentRemoteConfigURL, name)
}

// AgentRemoteConfigListResponse represents a list of agent remote configs with metadata.
type AgentRemoteConfigListResponse = v1.ListResponse[v1.AgentRemoteConfig]

// ListAgentRemoteConfigs lists all agent remote configs.
func (s *AgentRemoteConfigService) ListAgentRemoteConfigs(
	ctx context.Context,
	opts ...ListOption,
) (*AgentRemoteConfigListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[v1.AgentRemoteConfig](
		ctx,
		s.service,
		ListAgentRemoteConfigURL,
		ListSettings{
			limit:         listSettings.limit,
			continueToken: listSettings.continueToken,
		},
	)
}

// CreateAgentRemoteConfig creates a new agent remote config.
func (s *AgentRemoteConfigService) CreateAgentRemoteConfig(
	ctx context.Context,
	createRequest *v1.AgentRemoteConfig,
) (*v1.AgentRemoteConfig, error) {
	return createResource[v1.AgentRemoteConfig, v1.AgentRemoteConfig](
		ctx,
		s.service,
		CreateAgentRemoteConfigURL,
		createRequest,
	)
}

// UpdateAgentRemoteConfig updates an existing agent remote config.
func (s *AgentRemoteConfigService) UpdateAgentRemoteConfig(
	ctx context.Context,
	updateRequest *v1.AgentRemoteConfig,
) (*v1.AgentRemoteConfig, error) {
	return updateResource(
		ctx,
		s.service,
		UpdateAgentRemoteConfigURL,
		updateRequest.Metadata.Name,
		updateRequest,
	)
}

// DeleteAgentRemoteConfig deletes an agent remote config by its name.
func (s *AgentRemoteConfigService) DeleteAgentRemoteConfig(ctx context.Context, name string) error {
	return deleteResource(ctx, s.service, DeleteAgentRemoteConfigURL, name)
}
