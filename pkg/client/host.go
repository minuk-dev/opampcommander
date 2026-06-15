//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListHostURL is the path to list all discovered hosts.
	ListHostURL = "/api/v1/hosts"
	// GetHostURL is the path to get a host by ID.
	GetHostURL = "/api/v1/hosts/{id}"
	// ListHostAgentsURL is the path to list the agents running on a host.
	ListHostAgentsURL = "/api/v1/hosts/{id}/agents"
)

// HostService provides methods to interact with hosts.
type HostService struct {
	service *service
}

// NewHostService creates a new HostService.
func NewHostService(service *service) *HostService {
	return &HostService{
		service: service,
	}
}

// HostListResponse represents a list of hosts with metadata.
type HostListResponse = v1.ListResponse[v1.Host]

// GetHost retrieves a host by its ID.
func (s *HostService) GetHost(ctx context.Context, id string) (*v1.Host, error) {
	return getResource[v1.Host](ctx, s.service, GetHostURL, id)
}

// ListHosts lists all discovered hosts.
func (s *HostService) ListHosts(ctx context.Context, opts ...ListOption) (*HostListResponse, error) {
	return listResources[v1.Host](ctx, s.service, ListHostURL, newListSettings(opts))
}

// ListAgentsByHost lists the agents running on a host.
func (s *HostService) ListAgentsByHost(
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

	response, err := req.Get(ListHostAgentsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list host agents: %w", err)
	}

	if response.IsError() {
		return nil, &ResponseError{
			StatusCode:   response.StatusCode(),
			ErrorMessage: response.String(),
		}
	}

	return &result, nil
}
