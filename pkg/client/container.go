//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListContainerURL is the path to list all discovered containers.
	ListContainerURL = "/api/v1/containers"
	// GetContainerURL is the path to get a container by ID.
	GetContainerURL = "/api/v1/containers/{id}"
	// ListContainerAgentsURL is the path to list the agents running in a container.
	ListContainerAgentsURL = "/api/v1/containers/{id}/agents"
)

// ContainerService provides methods to interact with containers.
type ContainerService struct {
	service *service
}

// NewContainerService creates a new ContainerService.
func NewContainerService(service *service) *ContainerService {
	return &ContainerService{
		service: service,
	}
}

// ContainerListResponse represents a list of containers with metadata.
type ContainerListResponse = v1.ListResponse[v1.Container]

// GetContainer retrieves a container by its ID.
func (s *ContainerService) GetContainer(ctx context.Context, id string) (*v1.Container, error) {
	return getResource[v1.Container](ctx, s.service, GetContainerURL, id)
}

// ListContainers lists all discovered containers.
func (s *ContainerService) ListContainers(ctx context.Context, opts ...ListOption) (*ContainerListResponse, error) {
	return listResources[v1.Container](ctx, s.service, ListContainerURL, newListSettings(opts))
}

// ListAgentsByContainer lists the agents running in a container.
func (s *ContainerService) ListAgentsByContainer(
	ctx context.Context,
	id string,
	opts ...ListOption,
) (*AgentListResponse, error) {
	listSettings := newListSettings(opts)

	var result AgentListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetResult(&result)
	listSettings.applyTo(req)

	response, err := req.Get(ListContainerAgentsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list container agents: %w", err)
	}

	if response.IsError() {
		return nil, &ResponseError{
			StatusCode:   response.StatusCode(),
			ErrorMessage: response.String(),
		}
	}

	return &result, nil
}
