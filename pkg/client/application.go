//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListApplicationURL is the path to list all discovered applications.
	ListApplicationURL = "/api/v1/applications"
	// GetApplicationURL is the path to get a application by ID.
	GetApplicationURL = "/api/v1/applications/{id}"
	// ListApplicationAgentsURL is the path to list the agents running in a application.
	ListApplicationAgentsURL = "/api/v1/applications/{id}/agents"
)

// ApplicationService provides methods to interact with applications.
type ApplicationService struct {
	service *service
}

// NewApplicationService creates a new ApplicationService.
func NewApplicationService(service *service) *ApplicationService {
	return &ApplicationService{
		service: service,
	}
}

// ApplicationListResponse represents a list of applications with metadata.
type ApplicationListResponse = v1.ListResponse[v1.Application]

// GetApplication retrieves a application by its ID.
func (s *ApplicationService) GetApplication(ctx context.Context, id string) (*v1.Application, error) {
	return getResource[v1.Application](ctx, s.service, GetApplicationURL, id)
}

// ListApplications lists all discovered applications.
func (s *ApplicationService) ListApplications(
	ctx context.Context, opts ...ListOption,
) (*ApplicationListResponse, error) {
	return listResources[v1.Application](ctx, s.service, ListApplicationURL, newListSettings(opts))
}

// ListAgentsByApplication lists the agents running in a application.
func (s *ApplicationService) ListAgentsByApplication(
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

	response, err := req.Get(ListApplicationAgentsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list application agents: %w", err)
	}

	if response.IsError() {
		return nil, &ResponseError{
			StatusCode:   response.StatusCode(),
			ErrorMessage: response.String(),
		}
	}

	return &result, nil
}
