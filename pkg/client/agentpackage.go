//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListAgentPackageURL is the path to list all agent packages.
	ListAgentPackageURL = "/api/v1/agentpackages"
	// GetAgentPackageURL is the path to get an agent package by name.
	GetAgentPackageURL = "/api/v1/agentpackages/{id}"
	// CreateAgentPackageURL is the path to create a new agent package.
	CreateAgentPackageURL = "/api/v1/agentpackages"
	// UpdateAgentPackageURL is the path to update an existing agent package.
	UpdateAgentPackageURL = "/api/v1/agentpackages/{id}"
	// DeleteAgentPackageURL is the path to delete an agent package.
	DeleteAgentPackageURL = "/api/v1/agentpackages/{id}"
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

// GetAgentPackage retrieves an agent package by its name.
func (s *AgentPackageService) GetAgentPackage(
	ctx context.Context,
	name string,
) (*v1.AgentPackage, error) {
	return getResource[v1.AgentPackage](ctx, s.service, GetAgentPackageURL, name)
}

// AgentPackageListResponse represents a list of agent packages with metadata.
type AgentPackageListResponse = v1.ListResponse[v1.AgentPackage]

// ListAgentPackages lists all agent packages.
func (s *AgentPackageService) ListAgentPackages(
	ctx context.Context,
	opts ...ListOption,
) (*AgentPackageListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[v1.AgentPackage](
		ctx,
		s.service,
		ListAgentPackageURL,
		ListSettings{
			limit:         listSettings.limit,
			continueToken: listSettings.continueToken,
		},
	)
}

// CreateAgentPackage creates a new agent package.
func (s *AgentPackageService) CreateAgentPackage(
	ctx context.Context,
	createRequest *v1.AgentPackage,
) (*v1.AgentPackage, error) {
	return createResource[v1.AgentPackage, v1.AgentPackage](
		ctx,
		s.service,
		CreateAgentPackageURL,
		createRequest,
	)
}

// UpdateAgentPackage updates an existing agent package.
func (s *AgentPackageService) UpdateAgentPackage(
	ctx context.Context,
	updateRequest *v1.AgentPackage,
) (*v1.AgentPackage, error) {
	return updateResource(
		ctx,
		s.service,
		UpdateAgentPackageURL,
		updateRequest,
	)
}

// DeleteAgentPackage deletes an agent package by its name.
func (s *AgentPackageService) DeleteAgentPackage(ctx context.Context, name string) error {
	return deleteResource(ctx, s.service, DeleteAgentPackageURL, name)
}
